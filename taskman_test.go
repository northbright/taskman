package taskman_test

import (
	"context"
	"log"
	"time"

	"github.com/northbright/taskman"
)

type MyTask struct{}

func (t *MyTask) UniqueChecksum() []byte {
	return []byte{1}
}

func (t *MyTask) Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error) {

	for i := 0; i <= 100; i++ {
		chProgress <- i
		time.Sleep(time.Millisecond * 5)
	}

	return nil, nil
}

func ExampleTaskMan_Run() {

	/*
			type MyTask struct {}

		        func (t *MyTask) UnmarshalBinary([]byte) error {
		                return nil
		        }

			func (t *MyTask) MarshalBinary() ([]byte, error) {
				return nil, nil
			}
	*/

	tm, ch := taskman.New(context.Background(), 1)

	go func() {
		t := &MyTask{}
		id, err := tm.Add(t)
		if err != nil {
			return
		}
		log.Printf("id: %v", id)
		tm.Run(id, nil)
	}()

	for {
		select {
		case m := <-ch:
			switch m.Type {
			case taskman.ERROR:
				log.Printf("task %v error: %v", m.TaskID, m.Data.(string))
			case taskman.SCHEDULED:
				log.Printf("task %v scheduled", m.TaskID)
			case taskman.STARTED:
				log.Printf("task: %v started", m.TaskID)
			case taskman.STOPPED:
				log.Printf("task: %v stopped", m.TaskID)
			case taskman.DONE:
				log.Printf("task: %v done", m.TaskID)
			case taskman.EXITED:
				log.Printf("task: %v exited", m.TaskID)
				return
			case taskman.PROGRESS_UPDATED:
				p, _ := m.Data.(int)
				log.Printf("task: %v, progress: %v", m.TaskID, p)
			}
		}
	}

	// Output:
}
