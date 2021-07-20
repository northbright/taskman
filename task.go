package taskman

import (
	"encoding"
)

type Task interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Init() (int64, error)
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
