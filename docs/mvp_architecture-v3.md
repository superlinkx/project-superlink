# Superlink Project MVP Architecture & Execution Plan (Hermes + Local Edge Strategy)

## 1. System Architecture

The following describes the high-level architectural design of the Superlink Assistant MVP. It utilizes a strict Edge-First and Local-Network strategy to maximize privacy. STT (Speech-to-Text) and TTS (Text-to-Speech) are performed locally on the Google Pixel. Hermes Agent manages autonomous reasoning and state, powered by a local Ollama inference engine running Gemma 4 26B A4B via AMD ROCm passthrough.

### Logical Components

* **Google Pixel Device**: The physical edge device running the Android implementation.
    * **Superlink Android App**: The client-side application.
        * **VoiceInteractionService**: The entry point that allows the app to be registered as the system's default assistant.
        * **STT Module (On-Device)**: Uses Android `SpeechRecognizer` to convert local audio to text locally.
        * **TTS Module (On-Device)**: Uses Android `TextToSpeech` to convert received text to speech locally.
        * **WebSocket Client**: Manages a lightweight connection of text and JSON payloads to the backend during active use.
        * **Foreground WebSocket Service**: A persistent Android Foreground Service (`SuperlinkWebSocketService`) that maintains a constant local connection to the Go Gateway, completely eliminating the need for third-party push services (like FCM) for background wake-ups and cron events.
* **Superlink Backend**: The secure gateway and capability augmentation layer built in Go.
    * **Go Gateway**: Authenticates, validates, and routes data securely between the mobile client and the agent over the local network/VPN.
    * **Go MCP Server**: Exposes custom backend functions (like device integrations or local data access) to Hermes via the Model Context Protocol.
* **The AI Brain (Hermes Container)**: The autonomous agent framework.
    * **State & Memory**: Maintains `MEMORY.md`, `USER.md`, and SQLite contextual databases on a persistent host volume.
    * **Cron/Task Loop**: Capable of scheduling future actions and pinging the Go Gateway's Webhook URL when a task is due.
* **The AI Muscle (Ollama Container)**: The local inference engine.
    * **Gemma 4 26B A4B**: The specific LLM quantization used for reasoning.
    * **AMD ROCm Passthrough**: Hardware acceleration mapping `/dev/kfd` and `/dev/dri` to the container for local GPU inference.
* **Capabilities & Tools**:
    * **Firecrawl Container**: Sandboxed web search and scraping capability exposed directly to Hermes.

---

## 2. Data Flow (Text & Background Pipeline)

### Active Chat Flow
`User Speech` -> `Pixel STT` -> `WebSocket (Text)` -> `Go Gateway` -> `Hermes API` -> `Ollama (Gemma)` -> `Hermes (Decision/Memory)` -> `Go Gateway` -> `WebSocket (Text)` -> `Pixel TTS` -> `User Hears Audio`

### Background Notification Flow (100% Local)
1.  **Trigger**: Hermes Agent internal cron job fires based on its autonomous loop.
2.  **Relay**: Hermes hits a dedicated Go Webhook on the Gateway (`/hermes/webhook`).
3.  **Delivery**: Go sends the JSON payload down the persistent Foreground WebSocket.
4.  **Action**: The Kotlin `SuperlinkWebSocketService` intercepts the payload in the background, wakes the local TTS module, and speaks the reminder.

---

## 3. Execution Plan

### Phase 1: Core Agent & Infrastructure Bootstrapping
**Goal**: Stand up the Hermes agent, local Go Gateway, and AMD-accelerated LLM in Docker Compose (Podman).
* [x] Set up Podman Compose with Hermes, Go backend, and Firecrawl.
* [x] Configure Ollama container with AMD ROCm (`/dev/kfd` and `/dev/dri`) passthrough.
* [x] Implement an initialization container to automatically pull `gemma4:26b-a4b` on startup.
* [ ] Configure the Go Gateway to accept local WebSocket connections.
* [ ] Build a simple API proxy in Go to route WebSocket text into the Hermes container.
* [ ] Verify basic end-to-end connectivity between a mock client, the Go Gateway, and Hermes.

### Phase 2: Android Edge Implementation
**Goal**: Get the app registered as a system assistant and handle local audio and persistent background wakes without Firebase.
* [x] Remove Firebase Cloud Messaging dependencies.
* [ ] Implement `VoiceInteractionService` in Kotlin to allow system-level triggering.
* [ ] Build the STT Module using Android `SpeechRecognizer`.
* [ ] Build the TTS Module using Android `TextToSpeech`.
* [ ] Implement `SuperlinkWebSocketService` as an Android Foreground Service to maintain a persistent background connection to the Go Gateway.

### Phase 3: Intelligence & Skill Integration
**Goal**: Enable the agent to perform real-world tasks and access custom logic.
* [x] Configure Hermes to use Firecrawl out-of-the-box for native web search.
* [ ] Implement an MCP Server in Go to expose custom Project Superlink functions to Hermes.
* [ ] Verify Hermes can successfully recall persistent memory across sessions.
* [ ] Test background cron-trigger webhook flow from Hermes to Go.

### Phase 4: Optimization & Polishing
**Goal**: Improve UX, system responsiveness, and background reliability.
* [ ] Implement "Barge-in": Ability for the user to interrupt the local TTS playback via a new STT trigger.
* [ ] Refine the Android UI overlay for seamless integration with the OS.
* [ ] Stress test the Foreground Service to ensure it survives Doze mode and reliably delivers audio notifications.
