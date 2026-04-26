# Superlink Web Debugger & Chat UI Architecture

## 1. Overview
To interact with the Project Superlink Go backend without requiring the Android client, we are implementing a lightweight Web Client. This UI connects to the exact same WebSocket endpoint (`ws://localhost:8080/ws`) that the Android app uses, allowing for real-time testing of Hermes' responses, state management, and tool execution directly from a desktop browser.

To ensure maximum portability and simplify deployment—especially within our sandboxed Podman/Docker containers—the entire Web UI will be compiled directly into the Go backend binary using Go's native `embed` package.

---

## 2. UI Layout & Wireframe
The web interface is a pure Vanilla HTML/JS/CSS single-page application (SPA) designed for side-by-side debugging and chatting.

* **Left Panel (The Chat Interface):**
    * Connection Status Indicator (Red/Green dot).
    * Scrollable message history distinguishing User inputs from Hermes' responses.
    * Text input field and "Send" button.
* **Right Panel (The Debugger/Raw Logs):**
    * Real-time stream of the raw JSON payloads being sent (TX) and received (RX) over the WebSocket.
    * A configuration input to dynamically define the Go Gateway target URL.

---

## 3. Go Implementation: `embedfs` Strategy

Instead of serving files from a local `./web` directory on the host machine at runtime, we will bake the `index.html` directly into the compiled Go executable. This guarantees that wherever the Go binary goes, the UI goes with it, with zero risk of missing asset files.

### 3.1 Directory Structure (Pre-Compilation)
```text
backend/
├── main.go
├── ...
└── web/
    └── index.html (Contains all structural HTML, inline CSS, and Vanilla JS)
```

### 3.2 Go Gateway Integration (`main.go`)
We will use the `//go:embed` directive to capture the `web/` directory at compile time and serve it using Go's standard `http.FileServer`.

```go
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed web/*
var webFS embed.FS

func main() {
	// 1. Extract the "web" subdirectory from the embedded filesystem
	uiFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	// 2. Serve the embedded files at the root endpoint "/"
	http.Handle("/", http.FileServer(http.FS(uiFS)))

	// 3. Your existing WebSocket handler
	// http.HandleFunc("/ws", handleWebSocket)

	log.Println("Superlink Gateway started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```

---

## 4. Execution Plan
* [ ] Create the `backend/web/` directory.
* [ ] Add the `index.html` file containing the split-pane chat and debug UI.
* [ ] Update `main.go` to import the `embed` and `io/fs` packages.
* [ ] Add the `//go:embed web/*` directive and configure the HTTP multiplexer to serve the embedded filesystem.
* [ ] Rebuild the Go binary (or restart `podman compose up -d --build`) to verify the UI is accessible at `http://localhost:8080` without requiring local volume mounts for the web assets.
