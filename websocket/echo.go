package websocket

import "log"

// EchoHandler implements a simple echo handler
type EchoHandler struct{}

// HandleMessage echoes the received message back to the sender
func (h *EchoHandler) HandleMessage(conn *Connection, message DocumentMessage) error {
	log.Printf("Echoing message from %s: %s", conn.clientID, string(message.Data))
	return conn.SendMessage(message)
}

// OnConnect is called when a new connection is established
func (h *EchoHandler) OnConnect(conn *Connection) error {
	log.Printf("New WebSocket connection: %s", conn.clientID)
	return nil
}

// OnDisconnect is called when a connection is closed
func (h *EchoHandler) OnDisconnect(conn *Connection) error {
	log.Printf("WebSocket connection closed: %s", conn.clientID)
	return nil
}
