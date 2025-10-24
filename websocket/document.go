package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/emaforlin/ce-realtime-gateway/nats"
	"github.com/emaforlin/ce-realtime-gateway/publisher"
	natsPkg "github.com/nats-io/nats.go"
)

type DocumentHandler struct {
	natsManager *nats.Manager
	hub         *Hub
}

func NewDocumentHandler(natsManager *nats.Manager, hub *Hub) *DocumentHandler {
	return &DocumentHandler{
		natsManager: natsManager,
		hub:         hub,
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
		if err := h.natsManager.PublishDocumentEvent(event); err != nil {
			log.Printf("Failed to publish document event: %v", err)
		}
	}()

	log.Printf("Document event processed: type=%s, doc=%s, user=%s",
		event.Payload.Action, event.DocumentID, event.UserID)

	return nil
}

func (h *DocumentHandler) OnConnect(conn *Connection) error {
	documentID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		log.Printf("⚠️ No document ID found in connection metadata for user %s", conn.GetClientID())
		return nil
	}

	log.Printf("🔗 User %s joining document %s", conn.GetClientID(), documentID)

	// Dynamically subscribe to the document's NATS subject
	err := h.natsManager.Subscribe(documentID, h.createNATSHandler(documentID))
	if err != nil {
		log.Printf("❌ Failed to subscribe to NATS for document %s: %v", documentID, err)
		return err
	}

	log.Printf("✅ User %s successfully joined document %s", conn.GetClientID(), documentID)
	return nil
}

func (h *DocumentHandler) OnDisconnect(conn *Connection) error {
	documentID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		log.Printf("⚠️ No document ID found in connection metadata for user %s", conn.GetClientID())
		return nil
	}

	log.Printf("👋 User %s leaving document %s", conn.GetClientID(), documentID)

	// Desuscribirse dinámicamente del subject NATS del documento
	err := h.natsManager.Unsubscribe(documentID)
	if err != nil {
		log.Printf("❌ Failed to unsubscribe from NATS for document %s: %v", documentID, err)
	}

	log.Printf("🚪 Document connection closed: %s from document %s", conn.clientID, documentID)
	return nil
}

// createNATSHandler creates a NATS message handler of an specific document
func (h *DocumentHandler) createNATSHandler(documentID string) func(*natsPkg.Msg) {
	return func(msg *natsPkg.Msg) {
		log.Printf("📥 Received NATS message for document %s on subject %s", documentID, msg.Subject)

		// Parse the NATS message to extract the original sender
		var event publisher.DocumentEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			// Fallback: broadcast without exclusion
			h.hub.BroadcastToDocument(documentID, msg.Data)
			return
		}

		originalSenderID := event.UserID

		h.hub.BroadcastToDocument(documentID, msg.Data, originalSenderID)

		log.Printf("📡 Forwarded NATS message to WebSocket clients in document %s (excluded sender: %s)", documentID, originalSenderID)
	}
}
