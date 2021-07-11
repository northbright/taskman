package taskman_test

import (
	"context"
	//"fmt"
	"log"
	//"strconv"
	"encoding/json"
	"time"

	"github.com/northbright/taskman"
)

/*
// MyTask implements taskman.Task interface:
// UniqueChecksum() []byte
// Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error)
type MyTask struct {
	Name string
}

// UniqueChecksum returns the unique checksum to identify the task in the taskman.
func (t *MyTask) UniqueChecksum() []byte {
	// Return the byte slice of the name as checksum.
	return []byte(t.Name)
}

// Run does the real work.
func (t *MyTask) Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error) {
	i := 1

	// Load state to resume the task.
	// Convert state to percentage.
	if state != nil {
		i, _ = strconv.Atoi(string(state))
		log.Printf("load state: i = %v", i)
	}

	for {
		select {
		case <-ctx.Done():
			log.Printf("Run is canceled")

			// Return the progress percentage as state
			return []byte(fmt.Sprintf("%d", i)), ctx.Err()
		default:
			if i < 100 {
				log.Printf("Hello, %v", t.Name)
				i++
				// Send percentage to progress channel.
				chProgress <- i

				// Emulate heavy work by sleeping.
				time.Sleep(time.Millisecond * 20)
			} else {
				return nil, nil
			}
		}
	}
}

func ExampleTaskMan() {
	// Get the task state from message loop.
	chState := make(chan []byte)

	// Create a task manager which may run 2 task(s) at the same time.
	tm, ch := taskman.New(2)

	ids := []string{}

	go func() {
		// Prepare 2 tasks.
		names := []string{"Frank", "Luke"}

		for _, name := range names {
			// Create a new task.
			t := &MyTask{name}

			// Add the task.
			id, err := tm.Add(t)
			if err != nil {
				log.Printf("add task error: %v", err)
				return
			}
			ids = append(ids, id)

			// Run the task.
			if err = tm.Run(id, nil); err != nil {
				log.Printf("run task: %v error: %v", id, err)
				return
			}
		}

		timeout1 := time.After(time.Millisecond * 300)
		timeout2 := time.After(time.Millisecond * 800)

		id := ids[0]
		for {
			select {
			case <-timeout1:
				// Stop 1st task after timeout.
				if err := tm.Stop(id); err != nil {
					log.Printf("stop task: %v error: %v", id, err)
					return
				}
				log.Printf("stop task: %v successfully", id)

			case state := <-chState:
				// Resume 1st task after read STOPPED message from message channel.
				// Get the state from the STOPPED message.
				if err := tm.Run(id, state); err != nil {
					log.Printf("Resume task: %v error: %v", id, err)
					return
				}
				log.Printf("Resume task: %v successfully", id)

			case <-timeout2:
				// Stop all tasks after timeout.
				tm.StopAll()
				log.Printf("stop all tasks")
				return
			}
		}
	}()

	// Message handler
	for {
		select {
		case m := <-ch:
			switch m.Type {
			case taskman.ERROR:
				log.Printf("task: %v error: %v", m.TaskID, m.Data.(string))
			case taskman.SCHEDULED:
				log.Printf("task: %v scheduled", m.TaskID)
			case taskman.STARTED:
				log.Printf("task: %v started", m.TaskID)
			case taskman.STOPPED:
				log.Printf("task: %v stopped", m.TaskID)
				// Get the task's state and post it to another goroutine.
				state, _ := m.Data.([]byte)
				go func() {
					chState <- state
				}()

			case taskman.DONE:
				log.Printf("task: %v done", m.TaskID)
			case taskman.EXITED:
				log.Printf("task: %v exited", m.TaskID)
			case taskman.ALL_EXITED:
				log.Printf("all tasks exited")
				return
			case taskman.PROGRESS_UPDATED:
				p, _ := m.Data.(int)
				log.Printf("task: %v, progress: %v", m.TaskID, p)
			}
		}
	}

	// Output:
}
*/

type MyTask struct {
	name    string
	total   int64
	Current int64 `json:"current"`
}

func (t *MyTask) MarshalBinary() ([]byte, error) {
	return json.Marshal(t)
}

func (t *MyTask) UnmarshalBinary(state []byte) error {
	return json.Unmarshal(state, t)
}

func (t *MyTask) Init(ctx context.Context) error {
	// Add delay to emulate initialization.
	<-time.After(time.Millisecond * 30)
	log.Printf("%v initialized", t.name)
	return nil
}

func (t *MyTask) Deinit(ctx context.Context) error {
	// Add delay to emulate deinitialization.
	<-time.After(time.Millisecond * 20)
	log.Printf("%v deinitialized", t.name)
	return nil
}

func (t *MyTask) Step() (int64, bool, error) {
	if t.Current < t.total {
		t.Current++
	}

	time.Sleep(time.Millisecond * 10)

	done := false
	if t.Current == t.total {
		done = true
	}

	return t.Current, done, nil
}

func (t *MyTask) Total() int64 {
	return t.total
}

func init() {
	taskman.Register("MyTask", func(data []byte) (taskman.Task, error) {
		return &MyTask{
			name:    string(data),
			total:   100,
			Current: 0,
		}, nil
	})
}

func ExampleTaskMan() {
	var err error
	// Set concurrency.
	concurrency := 1
	// Create a new task manager.
	tm, ch, _ := taskman.New("MyTask", concurrency)
	// Create a channel to receive saved state.
	stateCh := make(chan []byte)

	// Start a new goroutine to handle the task messages.
	go func() {
		for {
			select {
			case m := <-ch:
				switch m.Type {
				case taskman.ERROR:
					log.Printf("task: %v error: %v", m.TaskID, m.Data.(string))
				case taskman.SCHEDULED:
					log.Printf("task: %v scheduled", m.TaskID)
				case taskman.STARTED:
					log.Printf("task: %v started", m.TaskID)
				case taskman.STOPPED:
					log.Printf("task: %v stopped", m.TaskID)
					state, _ := m.Data.([]byte)
					log.Printf("saved state: %s", string(state))
					stateCh <- state
				case taskman.DELETED:
					log.Printf("task: %v deleted", m.TaskID)
					state, _ := m.Data.([]byte)
					log.Printf("saved state: %s", string(state))

				case taskman.RESTORED:
					log.Printf("task: %v restored", m.TaskID)
					state, _ := m.Data.([]byte)
					log.Printf("restored state: %s", string(state))

				case taskman.SUSPENDED:
					log.Printf("task %v suspended", m.TaskID)
				case taskman.RESUMED:
					log.Printf("task %v resumed", m.TaskID)
				case taskman.DONE:
					log.Printf("task: %v done", m.TaskID)
					state, _ := m.Data.([]byte)
					log.Printf("final saved state: %s", string(state))
				case taskman.EXITED:
					log.Printf("task: %v exited", m.TaskID)
				case taskman.ALL_EXITED:
					log.Printf("all tasks exited")
				case taskman.PROGRESS_UPDATED:
					p, _ := m.Data.(int)
					log.Printf("task: %v, progress: %v", m.TaskID, p)
				}

			}
		}
	}()

	// Add first task to task manager and get the task ID.
	id1, _ := tm.Add([]byte("My first task"))

	// Add second task.
	id2, _ := tm.Add([]byte("My second task"))

	// Start the first task.
	if err = tm.Start(id1, nil); err != nil {
		log.Printf("start task %v error: %v", id1, err)
		return
	}

	// Start the second task.
	<-time.After(time.Millisecond * 20)
	if err = tm.Start(id2, nil); err != nil {
		log.Printf("start task %v error: %v", id2, err)
		return
	}

	// Start the first task twice.
	<-time.After(time.Millisecond * 10)
	if err = tm.Start(id1, nil); err != nil {
		log.Printf("start task %v again error: %v", id1, err)
	}

	// Suspend the first task.
	<-time.After(time.Millisecond * 200)
	if err = tm.Suspend(id1); err != nil {
		log.Printf("suspend task %v error: %v", id1, err)
		return
	}

	// Resume the first task.
	<-time.After(time.Millisecond * 1000)
	if err = tm.Resume(id1); err != nil {
		log.Printf("resume task %v error: %v", id1, err)
		return
	}

	// Stop the first task.
	<-time.After(time.Millisecond * 100)
	if err = tm.Stop(id1); err != nil {
		log.Printf("stop task %v error: %v", id1, err)
		return
	}

	// Restore the first task at once after stop it.
	// It loads the saved state.
	state := <-stateCh
	if err = tm.Start(id1, state); err != nil {
		log.Printf("restore task %v error: %v", id1, err)
		return
	}

	// Delete the first task
	<-time.After(time.Millisecond * 100)
	if err = tm.Delete(id1); err != nil {
		log.Printf("delete task %v error: %v", id1, err)
		return
	}

	<-time.After(time.Second * 5)

	// Output:
}
