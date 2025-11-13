package publisher

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	conn *nats.Conn
}

// Close implements Publisher.
func (n *NATSPublisher) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

// PublishDocumentEvent implements Publisher.
func (n *NATSPublisher) PublishDocumentEvent(event DocumentEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("document.%s.edit.%s", event.DocumentID, event.Payload.Action)

	if err := n.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}
	log.Printf("Published event to NATS: %s -> %d:%s", subject, event.Payload.Position, event.Payload.Data)
	return nil
}

func NewNATSPublisher(connection *nats.Conn) (*NATSPublisher, error) {

	return &NATSPublisher{
		conn: connection,
	}, nil
}
