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
	Name  string
	State State `json:"state"`
}

func NewTask(name string) *MyTask {
	return &MyTask{Name: name, State: State{Current: 0, Total: 100}}
}

func (t *MyTask) MarshalBinary() ([]byte, error) {
	return json.Marshal(t.State)
}

func (t *MyTask) UnmarshalBinary(state []byte) error {
	return json.Unmarshal(state, &t.State)
}

func (t *MyTask) Init() (int64, int64, error) {
	return t.State.Current, t.State.Total, nil
}

func (t *MyTask) Deinit() error {
	return nil
}

func (t *MyTask) Step() (int64, bool, []byte, error) {
	done := false
	var result []byte

	if t.State.Current < t.State.Total {
		t.State.Current += 1
		log.Printf("%s working... %v", t.Name, t.State.Current)
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

	t1 := NewTask("Task 1")
	id, _ := tm.Add(t1)
	log.Printf("Add task: id: %v", id)

	tm.Start(id, nil)

	<-time.After(time.Millisecond * 100)

	t2 := NewTask("Task 2")
	id, _ = tm.Add(t2)
	log.Printf("Add task: id: %v", id)

	tm.Start(id, nil)

	<-time.After(time.Second * 5)
	// Output:
}
