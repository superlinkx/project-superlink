package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// Session represents an individual user connection
type Session struct {
	ID          string
	Conn        *websocket.Conn
	ReadChan    chan []byte
	WriteChan   chan []byte
	CloseChan   chan struct{}
	IsConnected bool
}

// SessionManager manages all active sessions
type SessionManager struct {
	sessions   map[string]*Session
	mutex      sync.RWMutex
	shutdown   chan struct{}
	shutdownWg sync.WaitGroup
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * 2 * 10e9 // 2 minutes
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

// Message types
const (
	messageTypeText = iota
	messageTypeToolCall
)

// HermesClient handles communication with the Hermes Agent API
type HermesClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewHermesClient(baseURL string, apiKey string) *HermesClient {
	return &HermesClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *HermesClient) Generate(prompt string) (string, error) {
	url := h.baseURL + "/chat/completions"

	payload := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"model": "hermes",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("hermes agent returned error: %s (status: %d)", string(body), resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("invalid response format from hermes agent")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid choice format from hermes agent")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid message format from hermes agent")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("no content in response from hermes agent")
	}

	return content, nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		return true
	},
}

func main() {
	// Initialize session manager
	sessionManager := NewSessionManager()

	// Set up WebSocket route
	http.HandleFunc("/ws", sessionManager.handleWebSocket)

	// Initialize MCP Server
	mcpServer := NewMCPServer()
	mcpServer.RegisterTool("getUserProfile", getUserProfileHandler)
	mcpServer.RegisterTool("getWeather", getWeatherHandler)

	// Set up Hermes webhook endpoint
	http.HandleFunc("/hermes/webhook", handleHermesWebhook)

	// Set up MCP server endpoint
	http.HandleFunc("/mcp", mcpServer.HandleRequest)

	// Start HTTP server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting Superlink orchestrator on :%s", port)
	go func() {
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Superlink orchestrator...")
	close(sessionManager.shutdown)

	// Wait for all sessions to close
	sessionManager.shutdownWg.Wait()
	log.Println("Orchestrator shutdown complete")
}

// NewSessionManager creates a new SessionManager instance
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		shutdown: make(chan struct{}),
	}
}

// getHermesClient returns a singleton Hermes client
var hermesOnce sync.Once
var hermesClient *HermesClient

func getHermesClient() *HermesClient {
	hermesOnce.Do(func() {
		hermesURL := os.Getenv("HERMES_API_URL")
		if hermesURL == "" {
			hermesURL = "http://hermes:8642/v1"
		}
		hermesKey := os.Getenv("HERMES_API_KEY")
		if hermesKey == "" {
			hermesKey = "default-key"
		}
		hermesClient = NewHermesClient(hermesURL, hermesKey)
	})
	return hermesClient
}

// handleWebSocket handles incoming WebSocket connections
func (sm *SessionManager) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Generate session ID
	sessionID := fmt.Sprintf("%s-%d", r.RemoteAddr, os.Getpid())

	// Create new session
	session := &Session{
		ID:        sessionID,
		Conn:      conn,
		ReadChan:  make(chan []byte, 100),
		WriteChan: make(chan []byte, 100),
		CloseChan: make(chan struct{}),
	}

	// Add session to manager
	sm.addSession(sessionID, session)

	log.Printf("New connection established. Session ID: %s", sessionID)

	// Start concurrent loops for this session
	go sm.readLoop(session)
	go sm.orchestrator(session)
	go sm.writeLoop(session)

	// Wait for session to close
	<-session.CloseChan

	// Remove session from manager
	sm.removeSession(sessionID)
	log.Printf("Connection closed. Session ID: %s", sessionID)
}

// addSession adds a session to the manager
func (sm *SessionManager) addSession(id string, session *Session) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.sessions[id] = session
}

// removeSession removes a session from the manager
func (sm *SessionManager) removeSession(id string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	delete(sm.sessions, id)
}

// readLoop reads messages from WebSocket and forwards to ReadChan
func (sm *SessionManager) readLoop(session *Session) {
	defer func() {
		close(session.ReadChan)
		session.IsConnected = false
	}()

	// Set up ping/pong handling
	session.Conn.SetPingHandler(func(appData string) error {
		return session.Conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(writeWait))
	})
	session.Conn.SetPongHandler(func(appData string) error {
		return nil
	})

	for {
		select {
		case <-sm.shutdown:
			return
		default:
			_, message, err := session.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Session %s read error: %v", session.ID, err)
				}
				return
			}

			// Forward message to ReadChan
			select {
			case session.ReadChan <- message:
			case <-sm.shutdown:
				return
			}
		}
	}
}

// orchestrator processes incoming messages and generates responses
func (sm *SessionManager) orchestrator(session *Session) {
	defer close(session.CloseChan)

	for {
		select {
		case <-sm.shutdown:
			return
		case message, ok := <-session.ReadChan:
			if !ok {
				return
			}

			// Process the incoming message
			var input struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(message, &input); err != nil {
				log.Printf("Session %s JSON unmarshal error: %v", session.ID, err)
				continue
			}

			// Get response from Hermes Agent
			hermesClient := getHermesClient()
			response, err := hermesClient.Generate(input.Text)
			if err != nil {
				log.Printf("Session %s Hermes generation error: %v", session.ID, err)
				// Fallback to mock response if Hermes fails
				response = fmt.Sprintf("Superlink received: %s", input.Text)
			}

			// Create response message
			responseMsg := map[string]any{
				"type":    "text",
				"content": response,
			}
			responseBytes, err := json.Marshal(responseMsg)
			if err != nil {
				log.Printf("Session %s JSON marshal error: %v", session.ID, err)
				continue
			}

			// Send to WriteChan for transmission
			select {
			case session.WriteChan <- responseBytes:
			case <-sm.shutdown:
				return
			}
		}
	}
}

// writeLoop writes messages from WriteChan to WebSocket
func (sm *SessionManager) writeLoop(session *Session) {
	defer func() {
		session.Conn.Close()
	}()

	for {
		select {
		case <-sm.shutdown:
			return
		case message, ok := <-session.WriteChan:
			if !ok {
				return
			}

			// Set write deadline
			session.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := session.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Session %s write error: %v", session.ID, err)
				return
			}
		}
	}
}

// handleHermesWebhook handles incoming webhooks from Hermes Agent
func handleHermesWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Webhook JSON decode error: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Printf("Received webhook from Hermes: %v", payload)

	// TODO: Implement logic to send push notifications to Android clients
	// This would use Firebase Cloud Messaging or similar

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Webhook received")
}
