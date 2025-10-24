package nats

import (
	"fmt"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// SubscriptionManager handles dynamic NATS subscriptions per document
type SubscriptionManager struct {
	conn          *nats.Conn
	subscriptions map[string]*DocumentSubscription
	mutex         sync.RWMutex
}

// DocumentSubscription represents a subscription to a specific document
type DocumentSubscription struct {
	documentID      string
	subscription    *nats.Subscription
	connectionCount int
	mutex           sync.RWMutex
	messageHandler  func(documentID string, data []byte)
}

// NewSubscriptionManager crea un nuevo manager de suscripciones
func NewSubscriptionManager(conn *nats.Conn) *SubscriptionManager {
	return &SubscriptionManager{
		conn:          conn,
		subscriptions: make(map[string]*DocumentSubscription),
	}
}

// Subscribe se suscribe al subject NATS de un documento
func (sm *SubscriptionManager) Subscribe(documentID string, messageHandler func(string, []byte)) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	docSub, exists := sm.subscriptions[documentID]
	if !exists {
		// Crear nueva suscripción
		subject := fmt.Sprintf("document.%s.edit", documentID)

		// Handler que procesa mensajes NATS
		natsHandler := func(msg *nats.Msg) {
			log.Printf("📥 Received NATS message for document %s on subject %s", documentID, msg.Subject)

			// Llamar al handler personalizado
			if messageHandler != nil {
				messageHandler(documentID, msg.Data)
			}
		}

		sub, err := sm.conn.Subscribe(subject, natsHandler)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}

		docSub = &DocumentSubscription{
			documentID:      documentID,
			subscription:    sub,
			connectionCount: 0,
			messageHandler:  messageHandler,
		}
		sm.subscriptions[documentID] = docSub
		log.Printf("✅ Created NATS subscription for document: %s (subject: %s)", documentID, subject)
	}

	// Incrementar contador de conexiones
	docSub.mutex.Lock()
	docSub.connectionCount++
	count := docSub.connectionCount
	docSub.mutex.Unlock()

	log.Printf("👤 User subscribed to document %s (active connections: %d)", documentID, count)
	return nil
}

// Unsubscribe se desuscribe del subject NATS de un documento
func (sm *SubscriptionManager) Unsubscribe(documentID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	docSub, exists := sm.subscriptions[documentID]
	if !exists {
		return nil // Ya está desuscrito
	}

	docSub.mutex.Lock()
	docSub.connectionCount--
	count := docSub.connectionCount
	docSub.mutex.Unlock()

	log.Printf("👋 User unsubscribed from document %s (remaining connections: %d)", documentID, count)

	// Si no hay más conexiones, remover suscripción
	if count <= 0 {
		if err := docSub.subscription.Unsubscribe(); err != nil {
			log.Printf("❌ Error unsubscribing from document %s: %v", documentID, err)
		}
		delete(sm.subscriptions, documentID)
		log.Printf("🗑️ Removed NATS subscription for document: %s", documentID)
	}

	return nil
}

// GetActiveSubscriptions retorna el número de suscripciones activas
func (sm *SubscriptionManager) GetActiveSubscriptions() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return len(sm.subscriptions)
}

// GetDocumentConnectionCount retorna el número de conexiones para un documento
func (sm *SubscriptionManager) GetDocumentConnectionCount(documentID string) int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	if docSub, exists := sm.subscriptions[documentID]; exists {
		docSub.mutex.RLock()
		count := docSub.connectionCount
		docSub.mutex.RUnlock()
		return count
	}
	return 0
}

// Close cierra todas las suscripciones
func (sm *SubscriptionManager) Close() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for documentID, docSub := range sm.subscriptions {
		if err := docSub.subscription.Unsubscribe(); err != nil {
			log.Printf("❌ Error closing subscription for document %s: %v", documentID, err)
		}
	}
	sm.subscriptions = make(map[string]*DocumentSubscription)
	log.Printf("🔒 Closed all NATS subscriptions")
}
