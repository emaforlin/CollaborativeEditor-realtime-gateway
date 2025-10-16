package websocket

import "log"

type DocumentHandler struct{}

func (h *DocumentHandler) HandleMessage(conn *Connection, message Message) error {
	log.Printf("Received: %s from %s on %s", message.Data, conn.GetClientID(), conn.GetMetadata("DocumentID").(string))
	return nil
}

func (h *DocumentHandler) OnConnect(conn *Connection) error {
	log.Printf("User %s joined %s", conn.GetClientID(), conn.GetMetadata("DocumentID"))
	return nil
}

func (h *DocumentHandler) OnDisconnect(conn *Connection) error {
	log.Printf("Document connection closed: %s", conn.clientID)
	return nil
}
