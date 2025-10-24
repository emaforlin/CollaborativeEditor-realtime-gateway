package nats

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/emaforlin/ce-realtime-gateway/publisher"
	"github.com/nats-io/nats.go"
)

// Manager handles both publishing and subscription with a single NATS connection
type Manager struct {
	conn          *nats.Conn
	subscriptions map[string]*DocumentSubscription
	mutex         sync.RWMutex
}

// NewManager creates a new NATS manager with a single connection
func NewManager(natsURL string) (*Manager, error) {
	opts := []nats.Option{
		nats.Name("CollaborativeEditor-Gateway"),
		nats.Timeout(10 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	log.Printf("Connected to NATS at %s", natsURL)

	return &Manager{
		conn:          conn,
		subscriptions: make(map[string]*DocumentSubscription),
	}, nil
}

// PublishDocumentEvent publishes a document event (Publisher functionality)
func (m *Manager) PublishDocumentEvent(event publisher.DocumentEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Use the same subject pattern for consistency
	subject := fmt.Sprintf("document.%s.edit", event.DocumentID)

	if err := m.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	log.Printf("Published event to NATS: %s -> %s", subject, event.Payload.Action)
	return nil
}

// Subscribe creates or increments subscription for a document
func (m *Manager) Subscribe(documentID string, handler func(msg *nats.Msg)) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	docSub, exists := m.subscriptions[documentID]
	if !exists {
		// Create new subscription
		subject := fmt.Sprintf("document.%s.edit", documentID)
		sub, err := m.conn.Subscribe(subject, handler)
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
		}

		docSub = &DocumentSubscription{
			documentID:      documentID,
			subscription:    sub,
			connectionCount: 0,
		}
		m.subscriptions[documentID] = docSub
		log.Printf("Created NATS subscription for document: %s", documentID)
	}

	// Increment connection count
	docSub.mutex.Lock()
	docSub.connectionCount++
	count := docSub.connectionCount
	docSub.mutex.Unlock()

	log.Printf("User subscribed to document %s (active connections: %d)", documentID, count)
	return nil
}

// Unsubscribe decrements subscription count and removes if no more connections
func (m *Manager) Unsubscribe(documentID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	docSub, exists := m.subscriptions[documentID]
	if !exists {
		return nil // Already unsubscribed
	}

	docSub.mutex.Lock()
	docSub.connectionCount--
	count := docSub.connectionCount
	docSub.mutex.Unlock()

	log.Printf("User unsubscribed from document %s (remaining connections: %d)", documentID, count)

	// If no more connections, remove subscription
	if count <= 0 {
		if err := docSub.subscription.Unsubscribe(); err != nil {
			log.Printf("Error unsubscribing from document %s: %v", documentID, err)
		}
		delete(m.subscriptions, documentID)
		log.Printf("Removed NATS subscription for document: %s", documentID)
	}

	return nil
}

// GetConnection returns the underlying NATS connection (if needed for advanced operations)
func (m *Manager) GetConnection() *nats.Conn {
	return m.conn
}

// Close closes all subscriptions and the NATS connection
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Close all active subscriptions
	for documentID, docSub := range m.subscriptions {
		if err := docSub.subscription.Unsubscribe(); err != nil {
			log.Printf("Error closing subscription for document %s: %v", documentID, err)
		}
	}
	m.subscriptions = make(map[string]*DocumentSubscription)

	// Close NATS connection
	if m.conn != nil {
		m.conn.Close()
		log.Println("NATS connection closed")
	}

	return nil
}

// IsConnected checks if the NATS connection is still active
func (m *Manager) IsConnected() bool {
	return m.conn != nil && m.conn.IsConnected()
}

// GetStats returns statistics about active subscriptions
func (m *Manager) GetStats() map[string]int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := make(map[string]int)
	for docID, docSub := range m.subscriptions {
		docSub.mutex.RLock()
		stats[docID] = docSub.connectionCount
		docSub.mutex.RUnlock()
	}
	return stats
}
