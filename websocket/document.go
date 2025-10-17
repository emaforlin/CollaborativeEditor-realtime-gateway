package websocket

import (
	"log"

	"github.com/emaforlin/ce-realtime-gateway/config"
)

type DocumentHandler struct{}

func (h *DocumentHandler) HandleMessage(conn *Connection, message Message) error {
	documentId, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		documentId = ""
	}
	log.Printf("Received: %s from %s on %s", message.Data, conn.GetClientID(), documentId)
	return nil
}

func (h *DocumentHandler) OnConnect(conn *Connection) error {
	documentId, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		documentId = ""
	}
	log.Printf("User %s joined %s", conn.GetClientID(), documentId)
	return nil
}

func (h *DocumentHandler) OnDisconnect(conn *Connection) error {
	log.Printf("Document connection closed: %s", conn.clientID)
	return nil
}
