package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/emaforlin/ce-realtime-gateway/publisher"
)

type DocumentHandler struct {
	broker publisher.Publisher
}

func NewDocumentHandler(pub publisher.Publisher) *DocumentHandler {
	return &DocumentHandler{
		broker: pub,
	}
}

func (h *DocumentHandler) HandleMessage(conn *Connection, message DocumentMessage) error {
	documentID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		documentID = ""
	}

	userID := conn.GetClientID()

	log.Printf("Received: %s from %s on %s", message.Data, userID, documentID)

	var docMsg publisher.DocumentEventPayload
	if err := json.Unmarshal(message.Data, &docMsg); err != nil {
		log.Printf("failed to parse document message: %v", err)
		return err
	}

	event := publisher.DocumentEvent{
		DocumentID: documentID,
		UserID:     userID,
		Payload:    docMsg,
		Timestamp:  time.Now().Unix(),
	}

	go func() {
		if err := h.broker.PublishDocumentEvent(event); err != nil {
			log.Printf("Failed to publish document event: %v", err)
		}
	}()

	log.Printf("Document event processed: type=%s, doc=%s, user=%s",
		event.Payload.Action, event.DocumentID, event.UserID)

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
