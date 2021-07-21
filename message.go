package taskman

import ()

const (
	ADDED            = "added"
	SCHEDULED        = "scheduled"
	STARTED          = "started"
	STOPPED          = "stopped"
	REMOVED          = "removed"
	SUSPENDED        = "suspended"
	RESUMED          = "resumed"
	DONE             = "done"
	PROGRESS_UPDATED = "progress_updated"
)

type Message interface {
	Type() string
}

type AddedMessage struct {
	ID int `json:"id"`
}

func (m *AddedMessage) Type() string {
	return ADDED
}

type ScheduledMessage struct {
	ID int `json:"id"`
}

func (m *ScheduledMessage) Type() string {
	return SCHEDULED
}

type StartedMessage struct {
	ID    int    `json:"id"`
	State []byte `json:"state"`
}

func (m *StartedMessage) Type() string {
	return STARTED
}

type StoppedMessage struct {
	ID     int    `json:"id"`
	State  []byte `json:"state"`
	ErrMsg error  `json:"err_msg"`
}

func (m *StoppedMessage) Type() string {
	return STOPPED
}

type RemoveMessage struct {
	ID int `json:"id"`
}

func (m *RemoveMessage) Type() string {
	return REMOVED
}

type SuspendedMessage struct {
	ID int `json:"id"`
}

func (m *SuspendedMessage) Type() string {
	return SUSPENDED
}

type ResumedMessage struct {
	ID int `json:"id"`
}

func (m *ResumedMessage) Type() string {
	return RESUMED
}

type DoneMessage struct {
	ID     int    `json:"id"`
	Result []byte `json:"result"`
}

func (m *DoneMessage) Type() string {
	return DONE
}

type ProgressMessage struct {
	ID            int     `json:"id"`
	Progress      float32 `json:"progress"`
	TotalProgress float32 `json:"total_progress"`
}

func (m *ProgressMessage) Type() string {
	return PROGRESS_UPDATED
}
