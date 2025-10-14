package websocket

import "log"

type DocumentHandler struct{}

func (h *DocumentHandler) HandleMessage(conn *Connection, message Message) error {
	log.Printf("Received: %s", message.Data)
	return nil
}

func (h *DocumentHandler) OnConnect(conn *Connection) error {
	log.Printf("New document connection: %s", conn.clientID)
	return nil
}

func (h *DocumentHandler) OnDisconnect(conn *Connection) error {
	log.Printf("Document connection closed: %s", conn.clientID)
	return nil
}
