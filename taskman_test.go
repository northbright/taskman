package taskman_test

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/northbright/taskman"
)

type MyTask struct {
	Name string
}

func (t *MyTask) UniqueChecksum() []byte {
	return []byte(t.Name)
}

func (t *MyTask) Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error) {
	i := 0

	// Load state if need.
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
			if i <= 100 {
				log.Printf("Hello, %v", t.Name)

				// Send percentage to progress channel.
				chProgress <- i

				// Emulate heavy work by sleeping.
				time.Sleep(time.Millisecond * 5)
				i++
			} else {
				return nil, nil
			}
		}
	}
}

func ExampleTaskMan() {
	tm, ch := taskman.New(context.Background(), 2)

	go func() {
		names := []string{"Frank", "Luke", "Jacky", "Nango"}
		ids := []string{}

		for _, name := range names {
			// Create a new task.
			t := &MyTask{name}

			// Add task.
			id, err := tm.Add(t)
			if err != nil {
				log.Printf("add task error: %v", err)
				return
			}
			ids = append(ids, id)

			// Run task.
			if err = tm.Run(id, nil); err != nil {
				log.Printf("run task: %v error: %v", id, err)
				return
			}
		}

		// Stop first task after a timeout.
		time.Sleep(time.Millisecond * 30)

		if err := tm.Stop(ids[0]); err != nil {
			log.Printf("stop task: %v error: %v", ids[0], err)
			return
		}
		log.Printf("stop task: %v successfully", ids[0])
	}()

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
				// Update the task.
				id := m.TaskID
				t := &MyTask{"Capt"}
				if err := tm.Update(id, t); err != nil {
					log.Printf("update task: %v error: %v", id, err)
					return
				}
				log.Printf("update task: %v successfully", id)

				// Resume task with state.
				state, _ := m.Data.([]byte)
				if err := tm.Run(id, state); err != nil {
					log.Printf("Resume task: %v error: %v", id, err)
					return
				}
				log.Printf("Resume task: %v successfully", m.TaskID)

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
