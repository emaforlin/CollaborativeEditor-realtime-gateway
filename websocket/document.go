package websocket

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/emaforlin/ce-realtime-gateway/config"
	"github.com/emaforlin/ce-realtime-gateway/nats"
	"github.com/emaforlin/ce-realtime-gateway/publisher"
	natsPkg "github.com/nats-io/nats.go"
)

type DocumentHandler struct {
	natsManager *nats.Manager
	hub         *Hub

	// pendingSyncs tracks clients waiting for sync (clientID -> requesterID)
	pendingSyncs map[string]string
	syncMutex    sync.RWMutex
}

func NewDocumentHandler(natsManager *nats.Manager, hub *Hub) *DocumentHandler {
	return &DocumentHandler{
		natsManager:  natsManager,
		hub:          hub,
		pendingSyncs: make(map[string]string),
	}
}

func (h *DocumentHandler) HandleMessage(conn *Connection, message WebsocketMessage) error {
	documentID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	if !ok {
		documentID = ""
	}

	clientID := conn.GetClientID()

	log.Printf("Received: %s from %s on %s", message.Data, clientID, documentID)

	// Decode into a temporary struct to handle Content as RawMessage
	type rawWebsocketMessage struct {
		Type       string          `json:"type"`
		ClientID   string          `json:"client_id"`
		DocumentID string          `json:"document_id"`
		Content    json.RawMessage `json:"content"`
		Timestamp  int64           `json:"timestamp"`
	}

	var rawMsg rawWebsocketMessage
	if err := json.Unmarshal(message.Data, &rawMsg); err != nil {
		log.Printf("failed to parse document message: %v", err)
		return err
	}

	// Map to the domain model
	var messagePayload publisher.WebsocketMessagePayload
	messagePayload.Type = rawMsg.Type
	messagePayload.ClientID = rawMsg.ClientID
	messagePayload.DocumentID = rawMsg.DocumentID
	messagePayload.Timestamp = rawMsg.Timestamp

	// Handle Content field smartly
	if len(rawMsg.Content) > 0 {
		if rawMsg.Content[0] == '"' {
			// It's a string, likely base64 encoded
			var contentStr string
			if err := json.Unmarshal(rawMsg.Content, &contentStr); err != nil {
				log.Printf("failed to unmarshal content string: %v", err)
				messagePayload.Content = []byte(rawMsg.Content) // fallback
			} else {
				// Try to decode base64
				decoded, err := base64.StdEncoding.DecodeString(contentStr)
				if err == nil {
					messagePayload.Content = decoded
				} else {
					// Not base64, use string bytes
					messagePayload.Content = []byte(contentStr)
				}
			}
		} else {
			// Not a string (e.g. null, object, number)
			messagePayload.Content = []byte(rawMsg.Content)
		}
	} else {
		messagePayload.Content = []byte{}
	}

	// Handle sync protocol messages
	switch messagePayload.Type {
	case publisher.MsgTypeSyncResponse:
		return h.handleSyncResponse(conn, messagePayload)
	case publisher.MsgTypeSyncAck:
		return h.handleSyncAck(conn, documentID)
	default:
		// Regular document update - publish to NATS
		messagePayload.ClientID = clientID
		messagePayload.DocumentID = documentID

		fmt.Printf("Parsed message: %+v", messagePayload)

		go func() {
			if err := h.natsManager.PublishDocumentEvent(messagePayload); err != nil {
				log.Printf("Failed to publish document event: %v", err)
			}
		}()

		log.Printf("Document event processed: type=%s, doc=%s, user=%s",
			messagePayload.Type, messagePayload.DocumentID, messagePayload.ClientID)
	}

	return nil
}

// handleSyncResponse handles sync_response from existing clients
func (h *DocumentHandler) handleSyncResponse(conn *Connection, payload publisher.WebsocketMessagePayload) error {
	log.Printf("Received sync_response from %s, content length: %d", conn.GetClientID(), len(payload.Content))

	// Parse the sync response to get the target client ID and state
	var syncResp publisher.SyncResponsePayload
	if err := json.Unmarshal(payload.Content, &syncResp); err != nil {
		log.Printf("Failed to parse sync_response payload: %v", err)

		// Fallback: Look up who is waiting for sync in this document
		h.syncMutex.RLock()
		targetID := h.pendingSyncs[conn.GetClientID()]
		h.syncMutex.RUnlock()

		if targetID == "" {
			log.Printf("No pending sync request for response from %s", conn.GetClientID())
			return nil
		}

		// Forward the state to the waiting client (assume content is base64 state)
		syncInit := publisher.SyncInitPayload{
			Type:   publisher.MsgTypeSyncInit,
			Source: "peer",
			State:  string(payload.Content),
		}

		data, err := json.Marshal(syncInit)
		if err != nil {
			log.Printf("Failed to marshal sync_init: %v", err)
			return err
		}

		h.hub.SendToClient(targetID, data)
		log.Printf("✅ Forwarded sync state from %s to %s (fallback)", conn.GetClientID(), targetID)

		// Clean up pending sync
		h.syncMutex.Lock()
		delete(h.pendingSyncs, conn.GetClientID())
		h.syncMutex.Unlock()

		return nil
	}

	// Structured sync response with explicit target
	targetID := syncResp.TargetID
	if targetID == "" {
		log.Printf("sync_response missing target_id")
		return nil
	}

	syncInit := publisher.SyncInitPayload{
		Type:   publisher.MsgTypeSyncInit,
		Source: "peer",
		State:  syncResp.State,
	}

	data, err := json.Marshal(syncInit)
	if err != nil {
		log.Printf("Failed to marshal sync_init: %v", err)
		return err
	}

	h.hub.SendToClient(targetID, data)
	log.Printf("✅ Forwarded sync state to %s (source: %s)", targetID, conn.GetClientID())

	return nil
}

// handleSyncAck handles sync_ack from clients after they applied sync state
func (h *DocumentHandler) handleSyncAck(conn *Connection, documentID string) error {
	clientID := conn.GetClientID()
	log.Printf("✅ Sync acknowledged by %s for document %s", clientID, documentID)

	// Client is now fully synced - could trigger additional actions here if needed
	// For example: add to session participants, notify other clients, etc.

	return nil
}

func (h *DocumentHandler) OnConnect(conn *Connection) error {
	documentID, ok := conn.GetMetadata(config.MetaDocumentIDKey).(string)
	clientID := conn.GetClientID()

	if !ok {
		log.Printf("⚠️ No document ID found in connection metadata for user %s", clientID)
		return nil
	}

	log.Printf("🔗 User %s joining document %s", conn.GetClientID(), documentID)

	// Dynamically subscribe to the document's NATS subject
	err := h.natsManager.Subscribe(documentID, h.createNATSHandler(documentID))
	if err != nil {
		log.Printf("❌ Failed to subscribe to NATS for document %s: %v", documentID, err)
		return err
	}

	// Check if there are existing connections for this document
	// Note: At this point, the new connection is already registered in the hub
	existingConn := h.hub.GetFirstDocumentConnection(documentID, clientID)

	if existingConn != nil {
		// There's an existing collaborator - request sync from them
		log.Printf("📤 Requesting sync from existing collaborator %s for new client %s", existingConn.GetClientID(), clientID)

		// Track the pending sync
		h.syncMutex.Lock()
		h.pendingSyncs[existingConn.GetClientID()] = clientID
		h.syncMutex.Unlock()

		// Send sync_request to the existing client
		syncRequest := publisher.SyncRequestPayload{
			Type:        publisher.MsgTypeSyncRequest,
			RequesterID: clientID,
		}

		data, err := json.Marshal(syncRequest)
		if err != nil {
			log.Printf("Failed to marshal sync_request: %v", err)
			return err
		}

		existingConn.SendMessage(WebsocketMessage{
			MessageType: TextMessage,
			Data:        data,
		})

		log.Printf("✅ Sent sync_request to %s for new client %s", existingConn.GetClientID(), clientID)
	} else {
		// First collaborator - they'll need to fetch from DB
		// Send sync_init with source "db" and empty state (client will fetch)
		log.Printf("📄 First collaborator %s for document %s - notifying to load from DB", clientID, documentID)

		syncInit := publisher.SyncInitPayload{
			Type:   publisher.MsgTypeSyncInit,
			Source: "db",
			State:  "", // Empty - client fetches from documents service
		}

		data, err := json.Marshal(syncInit)
		if err != nil {
			log.Printf("Failed to marshal sync_init: %v", err)
			return err
		}

		conn.SendMessage(WebsocketMessage{
			MessageType: TextMessage,
			Data:        data,
		})

		log.Printf("✅ Sent sync_init (source: db) to first collaborator %s", clientID)
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

	// Clean up any pending syncs for this client
	h.syncMutex.Lock()
	delete(h.pendingSyncs, conn.GetClientID())
	h.syncMutex.Unlock()

	// Dynamically unsubscribe from the document's NATS subject
	err := h.natsManager.Unsubscribe(documentID)
	if err != nil {
		log.Printf("❌ Failed to unsubscribe from NATS for document %s: %v", documentID, err)
	}

	log.Printf("🚪 Document connection closed: %s from document %s", conn.clientID, documentID)
	return nil
}

// createNATSHandler creates a NATS message handler for a specific document
func (h *DocumentHandler) createNATSHandler(documentID string) func(*natsPkg.Msg) {
	return func(msg *natsPkg.Msg) {
		log.Printf("📥 Received NATS message for document %s on subject %s", documentID, msg.Subject)

		// Parse the NATS message to extract the original sender
		var event publisher.WebsocketMessagePayload
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			// Fallback: broadcast without exclusion
			h.hub.BroadcastToDocument(documentID, msg.Data)
			return
		}

		originalSenderID := event.ClientID

		h.hub.BroadcastToDocument(documentID, msg.Data, originalSenderID)

		log.Printf("📡 Forwarded NATS message to WebSocket clients in document %s (excluded sender: %s)", documentID, originalSenderID)
	}
}
