# Superlink Project MVP Architecture & Execution Plan (Hermes Agent Integration)

## 1. System Architecture

The following describes the high-level architectural design of the Superlink Assistant MVP. It utilizes an Edge-First strategy to maximize privacy and minimize latency by performing STT (Speech-to-Text) and TTS (Text-to-Speech) locally on the Google Pixel, while leveraging Hermes Agent for autonomous reasoning and state management.

### Logical Components

* **Google Pixel Device**: The physical edge device running the Android implementation.
    * **Superlink Android App**: The client-side application.
        * **VoiceInteractionService**: The entry point that allows the app to be registered as the system's default assistant.
        * **STT Module (On-Device)**: Uses Android SpeechRecognizer to convert local audio to text locally.
        * **TTS Module (On-Device)**: Uses Android TextToSpeech to convert received text to speech locally.
        * **WebSocket Client**: Manages a lightweight, persistent connection of text and JSON payloads to the backend.
        * **Push Receiver**: Intercepts silent data payloads (via FCM/WebPush) in the background without triggering standard Android UI banners.
* **Superlink Backend**: The secure gateway and capability augmentation layer built in Go.
    * **Go Gateway**: Authenticates, validates, and routes data securely between the mobile client and the agent.
    * **Go MCP Server**: Exposes custom Project Superlink functions and app-specific data to Hermes via the Model Context Protocol.
* **Hermes Agent Container**: The autonomous AI framework running persistently in Docker.
    * Handles its own user state, persistent memory (`MEMORY.md`, `USER.md`, SQLite FTS5), and internal learning loops.
* **External Services**:
    * **Firecrawl**: Native web provider for Hermes to fetch live data and summarize web pages.
    * **Firebase Cloud Messaging (FCM)**: Delivers silent data payloads to the Android device when background tasks are triggered.

### Data Flow Paths

#### The Main Interaction Loop
1.  **Input**: STT Module $\rightarrow$ WebSocket Client (Transcribed Text).
2.  **Ingress**: WebSocket Client $\rightarrow$ Go Gateway (Validates & Sanitizes).
3.  **Reasoning**: Go Gateway $\rightarrow$ Hermes Agent.
4.  **Execution**: Hermes autonomously evaluates if it needs an MCP Tool (Go) or Web Search (Firecrawl). It executes and synthesizes the response internally.
5.  **Egress**: Hermes Agent $\rightarrow$ Go Gateway $\rightarrow$ WebSocket Server $\rightarrow$ WebSocket Client $\rightarrow$ TTS Module (Local Speech Synthesis).

#### The Background Notification Flow
1.  **Trigger**: Hermes Agent internal cron job fires based on natural language scheduling.
2.  **Relay**: Hermes hits a dedicated Go Webhook.
3.  **Delivery**: Go checks session state. If closed, Go $\rightarrow$ FCM $\rightarrow$ Android Push Receiver.
4.  **Action**: Android App wakes silently $\rightarrow$ TTS Module speaks the reminder.

---

## 2. Execution Plan

The mission is divided into four tactical phases focusing on wiring up the autonomous Hermes Agent securely.

### Phase 1: Infrastructure & The Gateway
*Goal: Establish the secure containerized environment and core routing.*
- [ ] Deploy the Hermes Agent via Docker.
- [ ] Implement the thin Go Gateway to securely route WebSocket text into the Hermes container.
- [ ] Verify basic end-to-end connectivity between a mock client, the Go Gateway, and Hermes.

### Phase 2: Android Edge Implementation
*Goal: Get the app registered as a system assistant and handle local audio and background wakes.*
- [ ] Implement VoiceInteractionService in Kotlin to allow system-level triggering.
- [ ] Build the STT Module using Android SpeechRecognizer.
- [ ] Build the TTS Module using Android TextToSpeech.
- [ ] Implement the WebSocket Client using OkHttp for text/JSON streaming.
- [ ] Implement a Firebase Cloud Messaging (or similar WebPush) receiver to intercept silent data payloads in the background.

### Phase 3: Intelligence & Skill Integration
*Goal: Enable the agent to perform real-world tasks and access custom logic.*
- [ ] Configure Hermes to use Firecrawl out-of-the-box for native web search.
- [ ] Implement an MCP Server in Go to expose custom Project Superlink functions to Hermes.
- [ ] Verify Hermes can successfully recall persistent memory across sessions.

### Phase 4: Optimization & Polishing
*Goal: Improve UX, system responsiveness, and background reliability.*
- [ ] Implement "Barge-in": Ability for the user to interrupt the local TTS playback via a new STT trigger.
- [ ] Refine the Android UI overlay for seamless integration with the OS.
- [ ] Stress test the cron scheduling and push notification delivery pipeline to ensure background wake-ups are reliable.
