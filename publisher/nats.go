package publisher

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

type NATSPublisher struct {
	client *nats.Conn
}

// Close implements Publisher.
func (n *NATSPublisher) Close() {
	if n.client != nil {
		n.client.Close()
	}
}

// PublishDocumentEvent implements Publisher.
func (n *NATSPublisher) PublishDocumentEvent(event DocumentEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("documents.%s.events.%s", event.DocumentID, event.Payload.Action)

	if err := n.client.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}
	log.Printf("Published event to NATS: %s -> %d:%s", subject, event.Payload.Position, event.Payload.Data)
	return nil
}

func NewNATSPublisher(url string) (*NATSPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connection to NATS failed: %w", err)
	}
	return &NATSPublisher{
		client: nc,
	}, nil
}
