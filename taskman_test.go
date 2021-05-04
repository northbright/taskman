package taskman_test

import (
	"context"
	"log"
	"time"

	"github.com/northbright/taskman"
)

type MyTask struct{}

func (t *MyTask) UnmarshalBinary([]byte) error {
	return nil
}

func (t *MyTask) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (t *MyTask) UniqueChecksum() []byte {
	return []byte{1}
}

func (t *MyTask) Run(ctx context.Context, chProgress chan<- int) error {

	for i := 0; i <= 100; i++ {
		chProgress <- i
		time.Sleep(time.Millisecond * 5)
	}

	return nil
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
		tm.Run(id)
	}()

	for {
		select {
		case m := <-ch:
			switch m.Type {
			case taskman.PROGRESS_UPDATED:
				p, _ := m.Data.(int)
				log.Printf("task: %v, progress: %v", m.TaskID, p)
			}
		}
	}

	// Output:
}
