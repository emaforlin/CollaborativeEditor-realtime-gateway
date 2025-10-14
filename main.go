package main

import (
	"log"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/emaforlin/ce-realtime-gateway/handlers"
	"github.com/emaforlin/ce-realtime-gateway/middleware"
	"github.com/emaforlin/ce-realtime-gateway/server"
	"github.com/emaforlin/ce-realtime-gateway/websocket"
)

const version = "1.0.0"

func main() {
	// Load configuration
	cfg := config.Load()

	// Create server
	srv := server.New(cfg)

	// Create WebSocket hub and start it
	hub := websocket.NewHub()
	go hub.Run()

	// Create WebSocket upgrader and handler
	upgrader := websocket.NewUpgrader(cfg)
	echoHandler := &websocket.EchoHandler{}
	documentHandler := &websocket.DocumentHandler{}

	// Create HTTP handlers
	healthHandler := handlers.NewHealthHandler(version)
	infoHandler := handlers.NewInfoHandler(cfg)

	// Register routes with middleware
	srv.RegisterHandlerWithMiddleware("/health",
		healthHandler.ServeHTTP,
		middleware.Logger,
		middleware.Recovery,
		middleware.CORS,
	)

	srv.RegisterHandlerWithMiddleware("/info",
		infoHandler.ServeHTTP,
		middleware.Logger,
		middleware.Recovery,
		middleware.CORS,
	)

	// Register WebSocket endpoint
	srv.RegisterHandlerWithMiddleware("/ws/echo",
		websocket.HandleWebSocket(upgrader, hub, echoHandler),
		middleware.WebSocketLogger,
		middleware.Recovery,
	)

	// Register WebSocket endpoint for document collaboration
	srv.RegisterHandlerWithMiddleware("/ws/document",
		websocket.HandleWebSocket(upgrader, hub, documentHandler),
		middleware.WebSocketLogger,
		middleware.Recovery,
	)

	// Start server with graceful shutdown
	log.Fatal(srv.Start())
}
