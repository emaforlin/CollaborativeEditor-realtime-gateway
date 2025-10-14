package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/config"
)

// Server represents the HTTP server with graceful shutdown
type Server struct {
	config     *config.Config
	httpServer *http.Server
	mux        *http.ServeMux
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	mux := http.NewServeMux()

	return &Server{
		config: cfg,
		mux:    mux,
		httpServer: &http.Server{
			Addr:         cfg.GetServerAddress(),
			Handler:      mux,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
	}
}

// RegisterHandler registers a handler for the given pattern
func (s *Server) RegisterHandler(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// RegisterHandlerWithMiddleware registers a handler with middleware
func (s *Server) RegisterHandlerWithMiddleware(pattern string, handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	// Apply middlewares in reverse order
	finalHandler := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		finalHandler = middlewares[i](finalHandler)
	}
	s.mux.HandleFunc(pattern, finalHandler)
}

// Start starts the server with graceful shutdown
func (s *Server) Start() error {
	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s", s.config.GetServerAddress())
		log.Printf("WebSocket endpoint: %s/ws/echo", s.config.GetWebSocketURL(""))
		log.Printf("Health check: %s/health", s.config.GetHTTPURL(""))

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests a deadline to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
		return err
	}

	log.Println("Server exited")
	return nil
}

// Stop stops the server immediately
func (s *Server) Stop() error {
	return s.httpServer.Close()
}

// GetConfig returns the server configuration
func (s *Server) GetConfig() *config.Config {
	return s.config
}
