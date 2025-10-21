package publisher

type DocumentEvent struct {
	UserID     string               `json:"user_id"`
	DocumentID string               `json:"document_id"`
	Payload    DocumentEventPayload `json:"payload"`
	Timestamp  int64                `json:"timestamp"`
}

type DocumentEventPayload struct {
	Action   string `json:"action"`
	Position int    `json:"position"`
	Data     string `json:"data"`
}
