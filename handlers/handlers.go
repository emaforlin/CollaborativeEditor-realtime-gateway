package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/config"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
	Uptime    string    `json:"uptime"`
}

// HealthHandler handles health check requests
type HealthHandler struct {
	startTime time.Time
	version   string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		startTime: time.Now(),
		version:   version,
	}
}

// ServeHTTP implements http.Handler for health checks
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	uptime := time.Since(h.startTime)
	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    uptime.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// InfoResponse represents the server information response
type InfoResponse struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Endpoints   map[string]string `json:"endpoints"`
}

// InfoHandler handles server information requests
type InfoHandler struct {
	config *config.Config
}

// NewInfoHandler creates a new info handler
func NewInfoHandler(cfg *config.Config) *InfoHandler {
	return &InfoHandler{
		config: cfg,
	}
}

// ServeHTTP implements http.Handler for server information
func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := InfoResponse{
		Name:        "Collaborative Editor WebSocket Gateway",
		Version:     "1.0.0",
		Description: "Real-time WebSocket gateway for collaborative editing",
		Endpoints: map[string]string{
			"websocket_echo": h.config.GetWebSocketURL("/ws/echo"),
			"health":         h.config.GetHTTPURL("/health"),
			"info":           h.config.GetHTTPURL("/info"),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"error":   "Not Found",
		"message": "The requested resource was not found",
		"path":    r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(response)
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"error":   "Method Not Allowed",
		"message": "The requested method is not allowed for this resource",
		"method":  r.Method,
		"path":    r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(response)
}
