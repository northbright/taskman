package taskman_test

import (
	"encoding/json"
	"log"
	"time"

	"github.com/northbright/taskman"
)

type State struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

type MyTask struct {
	State State `json:"state"`
}

func NewTask() *MyTask {
	return &MyTask{State: State{Current: 0, Total: 100}}
}

func (t *MyTask) Save() ([]byte, error) {
	return json.Marshal(t.State)
}

func (t *MyTask) Init(state []byte) (int64, int64, error) {
	if state != nil {
		json.Unmarshal(state, &t.State)
	}
	return t.State.Current, t.State.Total, nil
}

func (t *MyTask) Deinit() error {
	return nil
}

func (t *MyTask) Step() (int64, bool, []byte, error) {
	done := false
	var result []byte

	if t.State.Current < t.State.Total {
		log.Printf("Working on step: %v", t.State.Current)
		t.State.Current += 1
		time.Sleep(time.Millisecond * 10)
	} else {
		done = true
		result = []byte("Hello Wolrd!")
	}

	return t.State.Current, done, result, nil
}

func onStatusChanged(env interface{}, id int, status taskman.Status, state []byte, result []byte) {
	log.Printf("onStatusChanged: id: %v, status: %v, state: %s, result: %s",
		id, status, state, result)
}

func onProgressUpdated(env interface{}, id int, current, total, currentAll, totalAll int64, progress, progressAll float32) {
	log.Printf("onProgressUpdated: id: %v, current: %v, total: %v, currentAll: %v, totalAll: %v, progress: %v, progressAll: %v",
		id, current, total, currentAll, totalAll, progress, progressAll)
}

func onError(env interface{}, id int, err error) {
	log.Printf("onError: id: %v, err: %v", id, err)
}

func Example() {
	tm, _ := taskman.New(1, nil, onStatusChanged, onProgressUpdated, onError)

	t := NewTask()
	id, _ := tm.Add(t)
	log.Printf("Add task: id: %v", id)

	tm.Start(id, nil)

	<-time.After(time.Second * 5)
	// Output:
}
