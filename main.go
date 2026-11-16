package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/andreykaipov/goobs"
	webview "github.com/jchv/go-webview2"
)

const defaultAddr = "127.0.0.1:4455"

type mark struct{ ts, file string }

type config struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
}

var (
	client *goobs.Client
	mu     sync.Mutex
	marks  = map[string]mark{}
	nextID int
	cfg    config
)

func configPath() string {
	dir, _ := os.UserConfigDir()
	return filepath.Join(dir, "obs-comments.json")
}

func loadConfig() config {
	b, err := os.ReadFile(configPath())
	if err != nil {
		return config{Addr: defaultAddr}
	}
	var c config
	if json.Unmarshal(b, &c) != nil || c.Addr == "" {
		return config{Addr: defaultAddr}
	}
	return c
}

func saveConfig(c config) error {
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(configPath(), b, 0600)
}

func tryConnect(c config) error {
	cl, err := goobs.New(c.Addr, goobs.WithPassword(c.Password))
	if err != nil {
		return err
	}
	if _, err := cl.Record.GetRecordStatus(); err != nil {
		cl.Disconnect()
		return err
	}
	client = cl
	return nil
}

func main() {
	cfg = loadConfig()
	tryConnect(cfg)

	// Create the native window
	w := webview.New(false)
	defer w.Destroy()

	w.SetTitle("OBS Comments")
	w.SetSize(440, 420, webview.HintNone)

	// Expose Go functions to JavaScript
	// 1. Connection
	w.Bind("goConnect", func(addr, pw string) string {
		c := config{Addr: addr, Password: pw}
		if err := tryConnect(c); err != nil {
			return "Error: " + err.Error()
		}
		cfg = c
		saveConfig(cfg)
		return "OK"
	})

	// 2. Mark
	w.Bind("goMark", func() string {
		if client == nil {
			return `{"error":"OBS not connected"}`
		}
		status, err := client.Record.GetRecordStatus()
		if err != nil {
			return `{"error":"OBS error: ` + err.Error() + `"}`
		}
		if !status.OutputActive {
			return `{"error":"OBS is not recording"}`
		}
		file, err := sidecarPath(client)
		if err != nil {
			return `{"error":"` + err.Error() + `"}`
		}
		ts := strings.Split(status.OutputTimecode, ".")[0]

		mu.Lock()
		nextID++
		id := fmt.Sprint(nextID)
		marks[id] = mark{ts: ts, file: file}
		mu.Unlock()

		return fmt.Sprintf(`{"id":"%s","ts":"%s"}`, id, ts)
	})

	// 3. Comment
	w.Bind("goComment", func(id, text string) string {
		mu.Lock()
		m, ok := marks[id]
		delete(marks, id)
		mu.Unlock()

		if !ok || text == "" {
			return `{"error":"unknown mark or empty text"}`
		}
		f, err := os.OpenFile(m.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return `{"error":"` + err.Error() + `"}`
		}
		_, werr := fmt.Fprintf(f, "%s  %s\n", m.ts, text)
		cerr := f.Close()
		if werr != nil || cerr != nil {
			return `{"error":"write failed"}`
		}
		return `{"ok":"saved"}`
	})

	// 4. Get initial config
	w.Bind("goGetConfig", func() string {
		return fmt.Sprintf(`{"addr":"%s","password":"%s"}`, cfg.Addr, cfg.Password)
	})

	// 5. Get connected status
	w.Bind("goIsConnected", func() bool {
		return client != nil
	})

	// Load the HTML directly into the window
	w.SetHtml(page)
	w.Run()
}

func sidecarPath(client *goobs.Client) (string, error) {
	dirResp, err := client.Config.GetRecordDirectory()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(dirResp.RecordDirectory)
	if err != nil {
		return "", err
	}

	var newest, newestExt string
	var newestTime time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		switch ext {
		case ".mkv", ".mp4", ".mov", ".flv", ".ts":
		default:
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = filepath.Join(dirResp.RecordDirectory, e.Name())
			newestExt = ext
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no video file found in %s", dirResp.RecordDirectory)
	}
	return strings.TrimSuffix(newest, newestExt) + ".txt", nil
}

// --- HTML / CSS / JS ---
// Notice I combined the setup page and the main page into one file.
// When the window loads, it checks if OBS is connected and shows the right screen.

const page = `<!doctype html>
<html><head><meta charset="utf-8"><title>OBS Comments</title>
<style>
  body{font-family:Segoe UI,sans-serif;background:#1e1e24;color:#eee;margin:0;padding:16px;user-select:none;}
  button{width:100%;padding:16px;font-size:18px;border:0;border-radius:8px;background:#4f7cff;color:#fff;cursor:pointer}
  button:hover{background:#6a90ff}
  button:disabled{opacity:0.5;cursor:default}
  #status{margin:14px 0 8px;min-height:22px;color:#aab}
  #status.err{color:#ff7b7b}
  input{width:100%;box-sizing:border-box;padding:12px;font-size:16px;border-radius:6px;border:1px solid #444;background:#2a2a32;color:#fff}
  input:disabled{opacity:.4}
  #log{margin-top:16px;max-height:200px;overflow-y:auto;font-size:14px;line-height:1.6}
  #log div span{color:#7fa2ff;margin-right:10px;font-family:Consolas,monospace}
  .hidden{display:none}
  h3{margin:0 0 12px;font-weight:600}
  label{display:block;font-size:13px;color:#aab;margin:10px 0 4px}
  .hint{font-size:12px;color:#777;margin-top:10px;line-height:1.5}
</style></head>
<body>

  <!-- Setup Screen -->
  <div id="setup" class="hidden">
    <h3>Connect to OBS</h3>
    <label for="addr">OBS WebSocket address</label>
    <input id="addr" value="127.0.0.1:4455" placeholder="127.0.0.1:4455">
    <label for="pw">Password</label>
    <input id="pw" type="password" placeholder="OBS WebSocket password">
    <button id="btnConnect" style="margin-top:16px">Connect</button>
    <div id="setupStatus" style="margin-top:12px;min-height:20px;color:#aab;font-size:14px"></div>
    <div class="hint">Enable the WebSocket server in OBS under Tools → WebSocket Server Settings. The address and password are saved locally.</div>
  </div>

  <!-- Main Screen -->
  <div id="main" class="hidden">
    <button id="btn">Comment Timestamp</button>
    <div id="status">Ready</div>
    <input id="inp" placeholder="Click the button first, then type here" disabled>
    <div id="log"></div>
  </div>

<script>
const setupDiv = document.getElementById('setup');
const mainDiv = document.getElementById('main');
const addrInp = document.getElementById('addr');
const pwInp = document.getElementById('pw');
const btnConnect = document.getElementById('btnConnect');
const setupStatus = document.getElementById('setupStatus');

const btn = document.getElementById('btn');
const inp = document.getElementById('inp');
const st = document.getElementById('status');
const log = document.getElementById('log');
let pending = null;

function reset(msg, err) {
  pending = null; inp.value = ''; inp.disabled = true;
  st.textContent = msg || 'Ready'; st.className = err ? 'err' : '';
}

// On load, check if already connected
window.onload = async () => {
  if (await goIsConnected()) {
    setupDiv.classList.add('hidden');
    mainDiv.classList.remove('hidden');
  } else {
    const cfg = JSON.parse(await goGetConfig());
    addrInp.value = cfg.addr;
    pwInp.value = cfg.password;
    setupDiv.classList.remove('hidden');
  }
};

// Connect flow
btnConnect.onclick = async () => {
  setupStatus.className = ''; setupStatus.textContent = 'Connecting...'; btnConnect.disabled = true;
  const res = await goConnect(addrInp.value.trim(), pwInp.value);
  if (res === 'OK') {
    setupStatus.className = ''; setupStatus.textContent = 'Connected. Loading...';
    setupDiv.classList.add('hidden');
    mainDiv.classList.remove('hidden');
  } else {
    setupStatus.className = 'err'; setupStatus.textContent = res; btnConnect.disabled = false;
  }
};

// Mark flow
btn.onclick = async () => {
  const resStr = await goMark();
  const j = JSON.parse(resStr);
  if (j.error) { reset(j.error, true); return; }
  pending = j; st.className = ''; st.textContent = 'Comment @ ' + j.ts;
  inp.disabled = false; inp.value = ''; inp.focus();
};

// Comment flow
inp.onkeydown = async (e) => {
  if (e.key === 'Escape') { reset('Cancelled'); return; }
  if (e.key === 'Enter' && pending) {
    const text = inp.value.trim(); if (!text) return;
    const ts = pending.ts;
    const resStr = await goComment(pending.id, text);
    const j = JSON.parse(resStr);
    if (j.error) { reset(j.error, true); return; }
    
    const d = document.createElement('div');
    const s = document.createElement('span');
    s.textContent = ts;
    d.appendChild(s);
    d.appendChild(document.createTextNode(text));
    log.prepend(d);
    reset('Saved @ ' + ts);
  }
};
</script></body></html>`