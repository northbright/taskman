package taskman

import (
	//"bytes"
	//"context"
	"errors"
	"log"
	"unsafe"
	//"runtime"
	"sync"
	//"sync/atomic"
)

const (
	DefaultConcurrency = 1
)

type NewTaskFunc func(data []byte) (Task, error)

type Callback func(env unsafe.Pointer, msg Message)

var (
	taskFuncs   = make(map[string]NewTaskFunc)
	taskFuncsMu = &sync.RWMutex{}

	NoSuchTaskNameErr = errors.New("no such task name")
	TaskNotFoundErr   = errors.New("task not found")
	TaskIsRunningErr  = errors.New("task is running")
	TaskIsStoppedErr  = errors.New("task is stopped")
	InvalidCommand    = errors.New("invalid command")
)

type taskData struct {
	t      Task
	status Status
}

type TaskMan struct {
	concurrency int
	newTask     NewTaskFunc
	env         unsafe.Pointer
	cb          Callback
	maxID       int
	sem         chan struct{}
	cmdCh       chan Command
	cmdResultCh chan CommandResult
	statusCh    chan string
	msgCh       chan Message
	taskDatas   map[int]taskData
	total       int64
}

func Register(name string, f NewTaskFunc) {
	if _, ok := taskFuncs[name]; ok {
		panic("taskman: Register twice for: " + name)
	}

	taskFuncsMu.Lock()
	taskFuncs[name] = f
	taskFuncsMu.Unlock()
}

func New(name string, concurrency int, env unsafe.Pointer, cb Callback) (*TaskMan, error) {
	taskFuncsMu.RLock()
	_, ok := taskFuncs[name]
	taskFuncsMu.RUnlock()
	if !ok {
		return nil, NoSuchTaskNameErr
	}

	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	tm := &TaskMan{
		concurrency: concurrency,
		newTask:     taskFuncs[name],
		env:         env,
		cb:          cb,
		maxID:       0,
		sem:         make(chan struct{}, concurrency),
		cmdCh:       make(chan Command),
		cmdResultCh: make(chan CommandResult),
		msgCh:       make(chan Message),
		taskDatas:   make(map[int]taskData),
		total:       0,
	}

	go func() {
		tm.handler()
	}()

	return tm, nil
}

func (tm *TaskMan) handler() {
	for {
		select {
		case cmd := <-tm.cmdCh:
			var r CommandResult = CommandResult{
				Cmd:     cmd,
				Success: true,
				ErrMsg:  "",
			}

			log.Printf("cmd: %v", cmd)
			switch cmd.Type() {
			case STOP:
				stopCmd, ok := cmd.(*StopCommand)
				if !ok {
					r.Success = false
					r.ErrMsg = InvalidCommand.Error()
				}

				status := tm.taskDatas[stopCmd.ID].status
				if status != SCHEDULED && status != STARTED && status != SUSPENDED && status != RESUMED {
					r.Success = false
					r.ErrMsg = TaskIsStoppedErr.Error()
				}

				tm.cmdResultCh <- r
			}

		case msg := <-tm.msgCh:
			log.Printf("msg %v", msg)
		}
	}
}

func computePercent(current, total int64) float64 {
	if total > 0 {
		return float64(current) / (float64(total) / float64(100))
	}
	return 0
}

func (tm *TaskMan) run(id int, t *Task, state []byte) {
	/*
		var (
			err        error
			savedState []byte
			result     []byte
			suspended  bool
			percent    float64
		)

		tm._msgCh <- &ScheduledMessage{ID: id}
		tm.sem <- struct{}{}

		for {
			select {
			case cmd := <-tm._cmdCh:

			case <-ctx.Done():
				if savedState, err = td.task.MarshalBinary(); err != nil {
					return
				}
				err = ctx.Err()
				return

			default:
				runtime.Gosched()

				td.suspendedMu.RLock()
				if suspended != td.suspended {
					suspended = td.suspended
					if suspended {
						tm.msgCh <- newMessage(id, SUSPENDED, nil)
					} else {
						tm.msgCh <- newMessage(id, RESUMED, nil)
					}
				}
				td.suspendedMu.RUnlock()

				if !suspended {
					current, done, err := td.task.Step()
					if err != nil {
						return
					}

					if td.showPercent {
						if td.total > 0 {
							newPercent := computePercent(current, td.total)
							if newPercent != percent {
								percent = newPercent

								tm.msgCh <- newMessage(id, PROGRESS_UPDATED, int(percent))
							}
						}
					}

					if done {
						// Save final state.
						if savedState, err = td.task.MarshalBinary(); err != nil {
							return
						}

						// Generate result.
						if result, err = td.task.Result(); err != nil {
							return
						}

						// Deinit task.
						if err = td.task.Deinit(ctx); err != nil {
							return
						}

						return
					}
				}
			}
		}
	*/
}
