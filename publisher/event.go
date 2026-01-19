package publisher

// Message type constants for sync protocol
const (
	MsgTypeSyncRequest  = "sync_request"
	MsgTypeSyncResponse = "sync_response"
	MsgTypeSyncInit     = "sync_init"
	MsgTypeSyncAck      = "sync_ack"
	MsgTypeYjsUpdate    = "yjs_update"
)

type WebsocketMessagePayload struct {
	Type       string `json:"type"`
	ClientID   string `json:"client_id"`
	DocumentID string `json:"document_id"`
	Content    []byte `json:"content"`   // base64 encoded
	Timestamp  int64  `json:"timestamp"` // unix timestamp
}

// SyncRequestPayload is sent to existing clients to request their Yjs state
type SyncRequestPayload struct {
	Type        string `json:"type"`
	RequesterID string `json:"requester_id"`
}

// SyncResponsePayload contains the Yjs state from an existing client
type SyncResponsePayload struct {
	Type     string `json:"type"`
	TargetID string `json:"target_id"` // The new client who needs the state
	State    string `json:"state"`     // Base64 encoded Yjs state
}

// SyncInitPayload is sent to new clients with the initial document state
type SyncInitPayload struct {
	Type   string `json:"type"`
	Source string `json:"source"` // "db" or "peer"
	State  string `json:"state"`  // Base64 encoded Yjs state
}
