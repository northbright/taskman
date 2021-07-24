package taskman

import ()

// Task is the common interface implemented by all real-world tasks.
type Task interface {
	// Save serializes the state to byte slice.
	Save() ([]byte, error)
	// Load deserializes the saved state.
	// Param:
	// state: saved state by calling Save().
	// int64: number of current step(index).
	// int64: number of total steps. It's used to compute progress.
	Load(state []byte) (int64, int64, error)
	// Init does the initialization before the first step starts.
	Init() error
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
