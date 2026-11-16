# 🤖 **AI-GENERATED CODE DISCLOSURE**
**The code and documentation for this tool were generated with the assistance of an AI (Large Language Model). Please review the code thoroughly before deploying it in a production environment.**

# OBS Timestamp Commenter

A lightweight, native Windows desktop tool for OBS Studio. It allows you to instantly capture the current recording timestamp and attach a written comment to it, saving the notes to a simple text file next to your video.

## The Problem It Solves
If you review OBS recordings (for highlights, bug reports, or training), scrubbing through hours of footage to find a specific moment is painful. This tool lets you click a button while recording, type a quick note, and hit Enter. It automatically saves the timestamp and note to a text file. 

## Features
* **Native Windows Window:** No browser tabs, no local web servers. Runs as a standalone `.exe`.
* **Zero Dependencies:** Uses the WebView2 runtime built into Windows 10 and 11.
* **Instant Marking:** Connects directly to the OBS WebSocket API.
* **Persistent Settings:** Saves your OBS IP/Password locally so you only have to log in once.
* **Simple Output:** Creates a `.txt` file with the exact same name as your video (e.g., `Recording_001.mkv` -> `Recording_001.txt`).

## Requirements
* **OS:** Windows 10 or Windows 11.
* **OBS Studio:** Version 28 or newer (OBS 28+ has the WebSocket server built-in).
* **Network:** The tool requires OBS and this application to be on the same machine (or the same local network).

## Setup & Usage

### 1. Enable OBS WebSocket
1. Open OBS Studio.
2. Go to **Tools** -> **WebSocket Server Settings**.
3. Check **"Enable WebSocket server"**.
4. Note the **Port** (default is `4455`) and the **Server Password** (you can set this in OBS).

### 2. Start the Tool
1. Launch `OBSComments.exe`.
2. On the first run, the setup screen will appear.
3. Enter the OBS Address (default: `127.0.0.1:4455`) and the Password you set in OBS.
4. Click **Connect**. (These settings will be saved for next time).

### 3. How to Mark a Timestamp
1. Ensure OBS is actively **Recording**.
2. Click the **Comment Timestamp** button in the tool.
3. The tool will grab the exact timecode and enable the text box.
4. Type your note and press **Enter**.
5. The tool will automatically append the timestamp and comment to a `.txt` file located in the same folder as your OBS recording.

*(Note: If you press `Escape`, it will cancel the comment.)*

## Where are my files saved?
If OBS is recording to `C:\Videos\Recording_001.mkv`, your comments will be saved directly into `C:\Videos\Recording_001.txt`.

The file format is plain text, making it easy to parse, read, or import into other video editing software:
```text
00:01:23  Here is a great highlight!
00:15:40  Oops, the microphone cut out here.