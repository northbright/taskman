package filehash_test

import (
	"log"
	"time"

	"github.com/northbright/taskman"
	_ "github.com/northbright/taskman/filehash"
)

func Example() {
	data := []byte(`{"file":"task.go","hash_funcs":["md5","sha1"]}`)

	tm, ch, _ := taskman.New("filehash", 1)

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
				case taskman.RESULT_GENERATED:
					log.Printf("task: %v result generated", m.TaskID)
					result, _ := m.Data.([]byte)
					log.Printf("result: %s", string(result))
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

	id, _ := tm.Add(data)
	tm.Start(id, nil)

	<-time.After(time.Second * 5)

	// Output:
}
