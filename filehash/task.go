package filehash

import (
	"context"
	"encoding/json"

	"github.com/northbright/taskman"
)

type Task struct {
	File      string   `json:"file"`
	HashFuncs []string `json:"hash_funcs"`
}

func init() {
	taskman.Register("filehash", loadTask)
}

func loadTask(data []byte) (taskman.Task, error) {
	t := &Task{}
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Task) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (t *Task) UnmarshalBinary(state []byte) error {
	return nil
}

func (t *Task) Init(ctx context.Context) error {
	return nil
}

func (t *Task) Deinit(ctx context.Context) error {
	return nil
}

func (t *Task) Step() (int64, bool, error) {
	return 0, false, nil
}

func (t *Task) Total() int64 {
	return 0
}
