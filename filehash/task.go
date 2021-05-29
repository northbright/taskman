package filehash

import (
	"context"
	"crypto"
	"encoding/json"

	"github.com/northbright/taskman"
)

type Task struct {
	File      string        `json:"file"`
	HashFuncs []crypto.Hash `json:"hash_funcs"`
}

func init() {
	taskman.Register("filehash", loadTask)
}

func loadTask(data []byte) taskman.Task {
	t := &Task{}
	json.Unmarshal(data, &t)
	return t
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
