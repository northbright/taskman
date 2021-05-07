package taskman_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/northbright/taskman"
)

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
