package taskman

import (
	"encoding"
)

// Task is the common interface implemented by all real-world tasks.
type Task interface {
	// Should implement encoding.BinaryMarshaler interface:
	// MarshalBinary() (data []byte, err error)
	// Used to serialize task state
	encoding.BinaryMarshaler
	// Should implement encoding.BinaryUnmarshaler interface:
	// UnmarshalBinary(data []byte) error
	// Used to deserialize task state.
	encoding.BinaryUnmarshaler
	// Init does the initialization before the first step starts.
	// Return:
	// int64: number of total steps. It's used to compute progress.
	Init() (int64, error)
	// Deinit deinitializes after all steps are done.
	Deinit() error
	// Step does the real work.
	// A large task can be devided into many small pieces(steps).
	// Param:
	// current: current step(index) to do.
	// Return:
	// int64: next step(index) to do.
	// bool: if task is done. The task goroutine will exit when it's true.
	// []byte: result when it's done otherwise it's ignored.
	// error: any error in the step will stop the task.
	Step(current int64) (int64, bool, []byte, error)
}
