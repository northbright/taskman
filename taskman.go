package taskman

import (
	//"bytes"
	"context"
	"errors"
	//"log"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	DefConcurrency = 16
)

type NewTaskFunc func(data []byte) (Task, error)

var (
	taskMans   = make(map[string]NewTaskFunc)
	taskMansMu = &sync.RWMutex{}

	NoSuchTaskNameErr = errors.New("no such task name")
	TaskNotFoundErr   = errors.New("task not found")
	TaskIsRunningErr  = errors.New("task is running")
	TaskIsStoppedErr  = errors.New("task is stopped")
)

type TaskData struct {
	task         Task
	ctx          context.Context
	cancelFunc   context.CancelFunc
	cancelFuncMu *sync.RWMutex
	suspended    bool
	suspendedMu  *sync.RWMutex
	showPercent  bool
	total        int64
	isDeleted    bool
	isDeletedMu  *sync.RWMutex
}

type TaskMan struct {
	ctx             context.Context
	name            string
	concurrency     int
	runningTasksNum int32
	newTask         NewTaskFunc
	taskDatas       map[int64]*TaskData
	taskDatasMu     *sync.RWMutex
	maxID           int64
	total           int64
	sem             chan struct{}
	msgCh           chan Message
}

func Register(name string, f NewTaskFunc) {
	if _, ok := taskMans[name]; ok {
		panic("taskman: Register twice for: " + name)
	}

	taskMansMu.Lock()
	taskMans[name] = f
	taskMansMu.Unlock()
}

func New(name string, concurrency int) (*TaskMan, <-chan Message, error) {
	taskMansMu.RLock()
	_, ok := taskMans[name]
	taskMansMu.RUnlock()
	if !ok {
		return nil, nil, NoSuchTaskNameErr
	}

	if concurrency <= 0 {
		concurrency = DefConcurrency
	}

	tm := &TaskMan{
		ctx:             context.Background(),
		name:            name,
		concurrency:     concurrency,
		runningTasksNum: 0,
		newTask:         taskMans[name],
		taskDatas:       make(map[int64]*TaskData),
		taskDatasMu:     &sync.RWMutex{},
		maxID:           0,
		total:           0,
		sem:             make(chan struct{}, concurrency),
		msgCh:           make(chan Message),
	}

	return tm, tm.msgCh, nil
}

func (tm *TaskMan) Add(data []byte) (int64, error) {
	t, err := tm.newTask(data)
	if err != nil {
		return -1, err
	}

	var total int64
	spt, ok := t.(ShowPercentTask)
	if ok {
		total = spt.Total()
		atomic.AddInt64(&tm.total, total)
	}

	id := atomic.AddInt64(&tm.maxID, 1)
	tm.taskDatasMu.Lock()
	td := &TaskData{
		task:         t,
		ctx:          nil,
		cancelFunc:   nil,
		cancelFuncMu: &sync.RWMutex{},
		suspended:    true,
		suspendedMu:  &sync.RWMutex{},
		showPercent:  ok,
		total:        total,
		isDeleted:    false,
		isDeletedMu:  &sync.RWMutex{},
	}
	tm.taskDatas[id] = td
	tm.taskDatasMu.Unlock()

	return id, nil
}

func (tm *TaskMan) Start(id int64, state []byte) error {
	tm.taskDatasMu.Lock()
	defer tm.taskDatasMu.Unlock()

	td, ok := tm.taskDatas[id]
	if !ok {
		return TaskNotFoundErr
	}

	if td.cancelFunc != nil {
		return TaskIsRunningErr
	}

	ctx, cancelFunc := context.WithCancel(tm.ctx)

	td.cancelFuncMu.Lock()
	td.cancelFunc = cancelFunc
	td.cancelFuncMu.Unlock()

	td.suspendedMu.Lock()
	td.suspended = false
	td.suspendedMu.Unlock()

	go func() {
		tm.run(ctx, id, td, state)
	}()

	return nil
}

func (tm *TaskMan) run(ctx context.Context, id int64, td *TaskData, state []byte) {
	var (
		err        error
		savedState []byte
		suspended  bool
		percent    float64
	)

	tm.msgCh <- newMessage(id, SCHEDULED, nil)

	tm.sem <- struct{}{}

	defer func() {
		<-tm.sem

		td.cancelFuncMu.Lock()
		td.cancelFunc = nil
		td.cancelFuncMu.Unlock()

		switch err {
		case context.Canceled, context.DeadlineExceeded:
			// Check if task is also deleted.
			td.isDeletedMu.RLock()
			isDeleted := td.isDeleted
			td.isDeletedMu.RUnlock()

			if isDeleted {
				// Remove the task from task data map
				tm.taskDatasMu.Lock()
				delete(tm.taskDatas, id)
				tm.taskDatasMu.Unlock()

				tm.msgCh <- newMessage(id, DELETED, savedState)
			} else {
				tm.msgCh <- newMessage(id, STOPPED, savedState)
			}

		case nil:
			tm.msgCh <- newMessage(id, DONE, savedState)
		default:
			tm.msgCh <- newMessage(id, ERROR, err.Error())
		}

		tm.msgCh <- newMessage(id, EXITED, nil)

		if atomic.AddInt32(&tm.runningTasksNum, -1) <= 0 {
			tm.msgCh <- newMessage(id, ALL_EXITED, nil)
		}
	}()

	if state != nil {
		if err = td.task.UnmarshalBinary(state); err != nil {
			return
		}

		// Init the task after state restored.
		if err = td.task.Init(ctx); err != nil {
			return
		}

		tm.msgCh <- newMessage(id, RESTORED, state)
	} else {
		// Init the task.
		if err = td.task.Init(ctx); err != nil {
			return
		}

		tm.msgCh <- newMessage(id, STARTED, nil)
	}

	atomic.AddInt32(&tm.runningTasksNum, 1)

	for {
		select {
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

					// Deinit task.
					if err = td.task.Deinit(ctx); err != nil {
						return
					}

					return
				}
			}
		}
	}
}

func computePercent(current, total int64) float64 {
	if total > 0 {
		return float64(current) / (float64(total) / float64(100))
	}
	return 0
}

func (tm *TaskMan) stop(id int64, isDeleted bool) error {
	tm.taskDatasMu.RLock()
	defer tm.taskDatasMu.RUnlock()

	td, ok := tm.taskDatas[id]

	if !ok {
		return TaskNotFoundErr
	}

	// Mark the task as to be isDeleted.
	if isDeleted {
		td.isDeletedMu.Lock()
		td.isDeleted = true
		td.isDeletedMu.Unlock()
	}

	td.cancelFuncMu.RLock()
	if td.cancelFunc != nil {
		td.cancelFunc()
	}
	td.cancelFuncMu.RUnlock()

	return nil
}

func (tm *TaskMan) Stop(id int64) error {
	return tm.stop(id, false)
}

func (tm *TaskMan) Delete(id int64) error {
	return tm.stop(id, true)
}

func (tm *TaskMan) setSuspendStatus(id int64, suspended bool) error {
	tm.taskDatasMu.RLock()
	defer tm.taskDatasMu.RUnlock()

	td, ok := tm.taskDatas[id]

	if !ok {
		return TaskNotFoundErr
	}

	td.cancelFuncMu.RLock()
	defer td.cancelFuncMu.RUnlock()
	if td.cancelFunc == nil {
		return TaskIsStoppedErr
	}

	td.setSuspendStatus(suspended)

	return nil
}

func (tm *TaskMan) Suspend(id int64) error {
	return tm.setSuspendStatus(id, true)
}

func (tm *TaskMan) Resume(id int64) error {
	return tm.setSuspendStatus(id, false)
}

func (td *TaskData) setSuspendStatus(suspended bool) {
	td.suspendedMu.Lock()
	if td.suspended != suspended {
		td.suspended = suspended
	}
	td.suspendedMu.Unlock()
}
