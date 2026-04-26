package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// MCPServer handles Model Context Protocol tools for Hermes Agent
type MCPServer struct {
	tools      map[string]func([]byte) ([]byte, error)
	mutex      sync.RWMutex
	httpClient *http.Client
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer() *MCPServer {
	return &MCPServer{
		tools:      make(map[string]func([]byte) ([]byte, error)),
		httpClient: &http.Client{},
	}
}

// RegisterTool registers a custom tool with the MCP server
func (m *MCPServer) RegisterTool(name string, handler func([]byte) ([]byte, error)) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tools[name] = handler
}

// HandleRequest handles incoming MCP requests from Hermes Agent
func (m *MCPServer) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		log.Printf("MCP server JSON decode error: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	m.mutex.RLock()
	handler, exists := m.tools[request.Method]
	m.mutex.RUnlock()

	if !exists {
		log.Printf("MCP tool not found: %s", request.Method)
		http.Error(w, "Method not found", http.StatusNotFound)
		return
	}

	responseData, err := handler(request.Params)
	if err != nil {
		log.Printf("MCP tool execution error: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  json.RawMessage(responseData),
		"id":      nil,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("MCP server response error: %v", err)
	}
}

// getUserProfileHandler is an example custom tool
func getUserProfileHandler(payload []byte) ([]byte, error) {
	var params struct {
		UserID string `json:"userId"`
	}

	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	// In a real implementation, this would fetch from a database
	response := map[string]interface{}{
		"userId":    params.UserID,
		"name":      "Test User",
		"email":     fmt.Sprintf("%s@example.com", params.UserID),
		"createdAt": "2024-01-01T00:00:00Z",
	}

	return json.Marshal(response)
}

// getWeatherHandler is another example custom tool
func getWeatherHandler(payload []byte) ([]byte, error) {
	var params struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	// In a real implementation, this would call a weather API
	response := map[string]interface{}{
		"location":    params.Location,
		"temperature": fmt.Sprintf("%d°C", 20), // Mock temperature
		"condition":   "Sunny",
		"humidity":    65,
	}

	return json.Marshal(response)
}
