package publisher

type Publisher interface {
	PublishDocumentEvent(event DocumentEvent) error
	Close()
}
