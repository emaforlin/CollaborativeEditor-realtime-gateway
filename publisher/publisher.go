package publisher

type Publisher interface {
	PublishDocumentEvent(event WebsocketMessagePayload) error
	Close()
}
