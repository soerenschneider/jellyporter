package events

type EventSyncRequest struct {
	Source   string
	Metadata string
	Response chan error
}
