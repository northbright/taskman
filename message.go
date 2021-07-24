package taskman

import ()

const (
	STATUS   = "status_updated"
	PROGRESS = "progress_updated"
	ERROR    = "error_occurred"
)

type Message interface {
	Type() string
}

type StatusUpdatedMessage struct {
	ID     int    `json:"id"`
	Status Status `json:"status"`
	// Only available for STOPPED status.
	State []byte `json:"state"`
	// Only available for DONE status.
	Result []byte `json:"result"`
}

func (m *StatusUpdatedMessage) Type() string {
	return STATUS
}

type ProgressUpdatedMessage struct {
	ID            int     `json:"id"`
	Progress      float32 `json:"progress"`
	TotalProgress float32 `json:"total_progress"`
}

func (m *ProgressUpdatedMessage) Type() string {
	return PROGRESS
}

type ErrorMessage struct {
	ID  int   `json:"id"`
	Err error `json:"err"`
}

func (m *ErrorMessage) Type() string {
	return ERROR
}
