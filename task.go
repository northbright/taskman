package taskman

import (
	"encoding"
)

// Task is the common interface implemented by all real-world tasks.
type Task interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	// Init initialize the task.
	// Return:
	// int64: number of current completed steps. It's used to compute progress.
	// int64: number of total steps. It's used to compute progress.
	// error: any error during the initialization.
	Init() (int64, int64, error)
	// Deinit deinitializes after all steps are done.
	Deinit() error
	// Step does the real work.
	// A large task can be devided into many small pieces(steps).
	// Return:
	// int64: number of current completed steps.
	// bool: if task is done. The task goroutine will exit when it's true.
	// []byte: result when it's done otherwise it's ignored.
	// error: any error in the step will stop the task.
	Step() (int64, bool, []byte, error)
}
