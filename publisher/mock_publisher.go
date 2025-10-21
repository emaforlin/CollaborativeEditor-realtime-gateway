package publisher

import "log"

type MockEventPublisher struct{}

// Close implements Publisher.
func (m *MockEventPublisher) Close() {
	log.Println("Mock Publisher closed")
}

// PublishDocumentEvent implements Publisher.
func (m *MockEventPublisher) PublishDocumentEvent(event DocumentEvent) error {
	log.Printf("Publish: %+v", event)
	return nil
}
