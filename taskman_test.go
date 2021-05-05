package taskman_test

import (
	"context"
	"log"
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

	for {
		select {
		case <-ctx.Done():
			log.Printf("Run is canceled")
			return nil, ctx.Err()
		default:
			if i <= 100 {
				log.Printf("Hello, %v", t.Name)
				chProgress <- i
				time.Sleep(time.Millisecond * 5)
				i++
			} else {
				return nil, nil
			}
		}
	}
}

func ExampleTaskMan_Run() {
	tm, ch := taskman.New(context.Background(), 2)

	go func() {
		names := []string{"Frank", "Luke"}
		ids := []string{}

		for _, name := range names {
			// Add task
			t := &MyTask{name}
			log.Printf("t address: %p", t)
			id, err := tm.Add(t)
			if err != nil {
				log.Printf("add task error: %v", err)
				return
			}
			ids = append(ids, id)

			// Run task
			if err = tm.Run(id, nil); err != nil {
				log.Printf("run task: %v error: %v", id, err)
				return
			}
		}

		time.Sleep(time.Millisecond * 30)

		// Stop first task
		if err := tm.Remove(ids[0]); err != nil {
			log.Printf("remove task: %v error: %v", ids[0], err)
			return
		}
		log.Printf("remove task: %v", ids[0])
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
