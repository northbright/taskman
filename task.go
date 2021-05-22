package taskman

import (
	//"context"
	"encoding"
)

/*
type Task interface {
	// UniqueChecksum returns a byte slice as checksum to identify the task in the TaskMan.
	UniqueChecksum() []byte

	// Run does the real work.
	// It's context aware: should exit ASAP when the context is canceled or timeout.
	// state is a byte slice which contains previously state which can be restored to resume the task.
	// Send percentage to the chProgress to notify TaskMan the progress of task is updated.
	// Marshal the latest correct state to a byte slice and return it when it's done or any error occurs.
	Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error)
}
*/

type Task interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Step() (int64, bool, error)
}

type ShowPercentTask interface {
	Total() int64
}
