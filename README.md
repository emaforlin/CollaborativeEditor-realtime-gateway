# Collaborative Editor WebSocket Gateway

A scalable and extensible WebSocket gateway for real-time collaborative editing applications.

## 🏗️ Architecture

The project follows a clean architecture pattern with clear separation of concerns:

```
├── main.go                 # Application entry point
├── config/                 # Configuration management
│   └── config.go
├── server/                 # HTTP server management
│   └── server.go
├── websocket/             # WebSocket handling
│   └── handler.go
├── middleware/            # HTTP middleware
│   └── middleware.go
├── handlers/              # HTTP request handlers
│   └── handlers.go
└── ws/                    # Legacy (to be removed)
    └── server.go
```

## 🔧 Features

### Configuration Management

- Environment variable based configuration
- Configurable timeouts, buffer sizes, and security settings
- Development defaults with production overrides

### WebSocket Features

- **Extensible Handler Interface**: Easy to implement custom message handlers
- **Connection Management**: Centralized hub for connection lifecycle
- **Message Broadcasting**: Built-in support for broadcasting to multiple connections
- **Connection Metadata**: Store custom data per connection

### Middleware Support

- **Logging**: Request/response logging with timing
- **CORS**: Cross-origin resource sharing support
- **Recovery**: Panic recovery and logging
- **Rate Limiting**: Simple IP-based rate limiting
- **Chainable**: Compose multiple middlewares

### HTTP Endpoints

- `/health` - Health check with uptime and version info
- `/info` - Server information and available endpoints
- `/ws/echo` - WebSocket echo endpoint

### Server Management

- **Graceful Shutdown**: Proper connection cleanup on shutdown
- **Signal Handling**: SIGINT/SIGTERM support
- **Configurable Timeouts**: Read/write timeout configuration

## 🚀 Usage

### Basic Usage

```go
package main

import (
    "github.com/emaforlin/ce-realtime-gateway/config"
    "github.com/emaforlin/ce-realtime-gateway/server"
    "github.com/emaforlin/ce-realtime-gateway/websocket"
)

func main() {
    cfg := config.Load()
    srv := server.New(cfg)

    // Create WebSocket components
    hub := websocket.NewHub()
    go hub.Run()

    upgrader := websocket.NewUpgrader(cfg)
    handler := &websocket.EchoHandler{}

    // Register WebSocket endpoint
    srv.RegisterHandler("/ws/echo",
        websocket.HandleWebSocket(upgrader, hub, handler))

    srv.Start()
}
```

### Custom WebSocket Handler

```go
type CustomHandler struct{}

func (h *CustomHandler) HandleMessage(conn *websocket.Connection, message websocket.Message) error {
    // Custom message processing logic
    conn.SetMetadata("lastMessage", string(message.Data))
    return conn.SendMessage(message)
}

func (h *CustomHandler) OnConnect(conn *websocket.Connection) error {
    // Custom connection logic
    log.Printf("User connected: %s", conn.GetClientID())
    return nil
}

func (h *CustomHandler) OnDisconnect(conn *websocket.Connection) error {
    // Custom disconnection logic
    log.Printf("User disconnected: %s", conn.GetClientID())
    return nil
}
```

### Environment Configuration

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_HOST=localhost
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# WebSocket Configuration
WS_CHECK_ORIGIN=true
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
WS_HANDSHAKE_TIMEOUT=10s
WS_ENABLE_COMPRESSION=false

# JWT Configuration (for future use)
JWT_SECRET=your-secret-key
JWT_TOKEN_DURATION=24h
JWT_ISSUER=collaborative-editor
```

## 🔌 Extensibility

### Adding New WebSocket Handlers

1. Implement the `websocket.Handler` interface
2. Register with the hub and server
3. Add any required middleware

### Adding New Middleware

```go
func CustomMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Custom logic before request
        next.ServeHTTP(w, r)
        // Custom logic after request
    }
}

// Usage
srv.RegisterHandlerWithMiddleware("/endpoint", handler, CustomMiddleware)
```

### Adding New HTTP Handlers

```go
type CustomHandler struct{}

func (h *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Custom HTTP handler logic
}

// Register
srv.RegisterHandler("/custom", customHandler.ServeHTTP)
```

## 🏃 Running the Server

```bash
# Development
go run main.go

# Production build
go build -o gateway main.go
./gateway
```

## 📡 API Endpoints

### WebSocket

- `ws://localhost:9001/ws/echo` - Echo WebSocket endpoint

### HTTP

- `GET /health` - Health check
- `GET /info` - Server information

## 🔍 Testing

Test the WebSocket connection:

```javascript
const ws = new WebSocket("ws://localhost:9001/ws/echo");
ws.onopen = () => ws.send("Hello, WebSocket!");
ws.onmessage = (event) => console.log("Received:", event.data);
```

Test health endpoint:

```bash
curl http://localhost:9001/health
```

## 📋 Next Steps

This architecture provides a solid foundation for:

- JWT authentication middleware
- Database integration
- Message persistence
- Room-based messaging
- User session management
- Real-time collaborative editing features

The modular design makes it easy to extend and modify components independently.
