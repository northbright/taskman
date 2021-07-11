package taskman

import (
	"encoding/json"
)

// MessageType represents the type of messages,
// which are generated when computing file hashes.
type MessageType uint

const (
	// An error occurs when task is running.
	ERROR MessageType = iota
	// Task is scheduled.
	SCHEDULED
	// Task is started.
	STARTED
	// Task is stopped.
	STOPPED
	// Task is deleted.
	DELETED
	// Task is restored.
	RESTORED
	// Task is suspended.
	SUSPENDED
	// Task is resumed.
	RESUMED
	// Progress of task is updated.
	PROGRESS_UPDATED
	// Task is done.
	DONE
	// Goroutine of the task exited.
	EXITED
	// All tasks exited.
	ALL_EXITED
	// Unknown message type.
	UNKNOWN
	maxTYPE
)

var (
	messageTypeStrs = map[MessageType]string{
		ERROR:            "error",
		SCHEDULED:        "scheduled",
		STARTED:          "started",
		STOPPED:          "stopped",
		DELETED:          "deleted",
		RESTORED:         "restored",
		SUSPENDED:        "suspended",
		RESUMED:          "resumed",
		PROGRESS_UPDATED: "progress_updated",
		DONE:             "done",
		EXITED:           "exited",
		ALL_EXITED:       "all_exited",
		UNKNOWN:          "unknown",
	}
)

// Message represents the messages.
type Message struct {
	// Task ID
	// Ignored for ALL_EXITED type.
	TaskID int64 `json:"task_id"`
	// Type is the type code(uint) of message.
	Type MessageType `json:"type"`
	// TypeStr is the type in string.
	TypeStr string `json:"type_str"`
	// Data stores the data of message.
	// Each type has its own data type.
	//
	// ERROR: data is a string to store error message.
	// SCHEDULED: data is nil.
	// STARTED: data is nil.
	// STOPPED: data is []byte to store saved state.
	// DELETED: data is []byte to store saved state.
	// RESTORED: data is []byte to store restored state.
	// SUSPENDED: data is nil.
	// RESUMED: data is nil.
	// PROGRESS_UPDATED: data is a int to store the percent(0 - 100).
	// DONE: data is []byte to store final saved state.
	// EXITED: data is nil.
	// ALL_EXITED: data is nil.
	Data interface{} `json:"data,omitempty"`
}

// newMessage returns a new message.
func newMessage(id int64, t MessageType, data interface{}) Message {
	typeStr, ok := messageTypeStrs[t]
	if !ok {
		t = UNKNOWN
		typeStr = messageTypeStrs[t]
	}

	return Message{id, t, typeStr, data}
}

// JSON serializes a message as JSON.
// All types of messages are JSON-friendly.
// To get the meaning of Message.Data for different types of messages,
// check comments of Message struct.
func (m Message) JSON() ([]byte, error) {
	return json.Marshal(m)
}
