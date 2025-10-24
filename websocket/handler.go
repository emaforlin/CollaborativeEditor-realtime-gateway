package websocket

import (
	"log"
	"net/http"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/emaforlin/ce-realtime-gateway/middleware"
	"github.com/gorilla/websocket"
)

// MessageType represents different types of WebSocket messages
type MessageType int

const (
	// TextMessage represents a text message
	TextMessage MessageType = websocket.TextMessage
	// BinaryMessage represents a binary message
	BinaryMessage MessageType = websocket.BinaryMessage
)

// Message represents a WebSocket message
type DocumentMessage struct {
	Type       MessageType `json:"type"`
	DocumentID string      `json:"document_id"`
	Data       []byte      `json:"data"`
}

// Connection wraps a WebSocket connection with additional functionality
type Connection struct {
	conn     *websocket.Conn
	clientID string
	metadata map[string]interface{}
	send     chan DocumentMessage
	hub      *Hub
}

// Hub manages WebSocket connections
type Hub struct {
	connections map[string]*Connection
	register    chan *Connection
	unregister  chan *Connection
	broadcast   chan DocumentMessage
}

// Handler represents a WebSocket message handler
type Handler interface {
	HandleMessage(conn *Connection, message DocumentMessage) error
	OnConnect(conn *Connection) error
	OnDisconnect(conn *Connection) error
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		connections: make(map[string]*Connection),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		broadcast:   make(chan DocumentMessage),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.connections[conn.clientID] = conn
			docID := conn.GetMetadata(config.MetaDocumentIDKey)
			log.Printf("Connection registered: %s (Document: %v)", conn.clientID, docID)

		case conn := <-h.unregister:
			if _, ok := h.connections[conn.clientID]; ok {
				delete(h.connections, conn.clientID)
				close(conn.send)
				docID := conn.GetMetadata(config.MetaDocumentIDKey)
				log.Printf("Connection unregistered: %s (Document: %v)", conn.clientID, docID)
			}

		case message := <-h.broadcast:
			for clientID, conn := range h.connections {
				select {
				case conn.send <- message:
				default:
					delete(h.connections, clientID)
					close(conn.send)
				}
			}
		}
	}
}

// BroadcastToDocument sends a message to all the connections on a specific document
func (h *Hub) BroadcastToDocument(documentID string, data []byte, excludeClientID ...string) {
	count := 0
	log.Printf("🔍 Broadcasting to document: %s", documentID)
	log.Printf("🔍 Total connections: %d", len(h.connections))

	excludeID := ""
	if len(excludeClientID) > 0 {
		excludeID = excludeClientID[0]
	}

	for _, conn := range h.connections {

		// Verify if the connection belongs to the document
		connDocID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
		log.Printf("🔍 Connection %s has document ID: %v (type: %T)", conn.clientID, connDocID, conn.GetMetadata(config.MetaDocumentIDKey))

		if ok && connDocID == documentID {
			if excludeID != "" && conn.clientID == excludeID {
				continue
			}

			select {
			case conn.send <- DocumentMessage{
				Type: TextMessage,
				Data: data,
			}:
				count++
				log.Printf("✅ Sent message to connection %s", conn.clientID)
			default:
				// Locked connection, close it
				delete(h.connections, conn.clientID)
				close(conn.send)
				log.Printf("❌ Closed blocked connection: %s", conn.clientID)
			}
		} else {
			log.Printf("❌ Connection %s doesn't match document %s (has: %s)", conn.clientID, documentID, connDocID)
		}
	}
	log.Printf("📡 Broadcasted message to %d connections in document %s", count, documentID)
}

// SendMessage sends a message to a specific connection
func (c *Connection) SendMessage(message DocumentMessage) error {
	select {
	case c.send <- message:
		return nil
	default:
		return &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "connection closed"}
	}
}

// GetMetadata returns connection metadata
func (c *Connection) GetMetadata(key string) interface{} {
	return c.metadata[key]
}

// SetMetadata sets connection metadata
func (c *Connection) SetMetadata(key string, value interface{}) {
	c.metadata[key] = value
}

// GetClientID returns the client ID
func (c *Connection) GetClientID() string {
	return c.clientID
}

// NewUpgrader creates a WebSocket upgrader with the given configuration
func NewUpgrader(cfg *config.Config) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return !cfg.WebSocket.CheckOrigin // Allow all origins when CheckOrigin is false
		},
		ReadBufferSize:   cfg.WebSocket.ReadBufferSize,
		WriteBufferSize:  cfg.WebSocket.WriteBufferSize,
		HandshakeTimeout: cfg.WebSocket.HandshakeTimeout,
		//EnableCompression:  cfg.WebSocket.EnableCompression,
	}
}

// HandleWebSocket creates a WebSocket handler function
func HandleWebSocket(upgrader websocket.Upgrader, hub *Hub, handler Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientId, ok := middleware.GetUserID(r)
		if !ok || clientId == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		docId := r.PathValue("id")

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}

		// Create connection wrapper
		wsConn := &Connection{
			conn:     conn,
			clientID: clientId,
			metadata: make(map[string]interface{}),
			send:     make(chan DocumentMessage, 256),
			hub:      hub,
		}
		wsConn.SetMetadata(config.MetaRemoteAddrKey, r.RemoteAddr)
		wsConn.SetMetadata(config.MetaDocumentIDKey, docId)

		// Register connection with hub
		hub.register <- wsConn

		// Call connect handler
		if err := handler.OnConnect(wsConn); err != nil {
			log.Printf("Connection handler error: %v", err)
			return
		}

		// Start goroutines for reading and writing
		go wsConn.writePump()
		go wsConn.readPump(handler)
	}
}

// readPump handles incoming messages from the WebSocket connection
func (c *Connection) readPump(handler Handler) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		handler.OnDisconnect(c)
	}()

	for {
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		message := DocumentMessage{
			Type: MessageType(messageType),
			Data: data,
		}

		if err := handler.HandleMessage(c, message); err != nil {
			log.Printf("Message handler error: %v", err)
		}
	}
}

// writePump handles outgoing messages to the WebSocket connection
func (c *Connection) writePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(int(message.Type), message.Data); err != nil {
			log.Printf("Write error: %v", err)
			return
		}
	}
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
