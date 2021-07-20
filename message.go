package taskman

import (
	"encoding/json"
)

type Message interface {
	json.Marshaler
}

type AddedMessage struct {
	ID string `json:"id"`
}

func (m *AddedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type ScheduledMessage struct {
	ID string `json:"id"`
}

func (m *ScheduledMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type StartedMessage struct {
	ID    string `json:"id"`
	State []byte `json:"state"`
}

func (m *StartedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type StoppedMessage struct {
	ID     string `json:"id"`
	State  []byte `json:"state"`
	ErrMsg error  `json:"err_msg"`
}

func (m *StoppedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type SuspendedMessage struct {
	ID string `json:"id"`
}

func (m *SuspendedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type ResumedMessage struct {
	ID string `json:"id"`
}

func (m *ResumedMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type DoneMessage struct {
	ID     string `json:"id"`
	Result []byte `json:"result"`
}

func (m *DoneMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}

type ProgressMessage struct {
	ID            string  `json:"id"`
	Progress      float32 `json:"progress"`
	TotalProgress float32 `json:"total_progress"`
}

func (m *ProgressMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m)
}
