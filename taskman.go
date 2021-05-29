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

/*
type TaskMan struct {
	ctx         context.Context
	cancel      context.CancelFunc
	concurrency int
	// Make map goroutine safe
	muMap *sync.Mutex
	// Make operation goroutine safe
	muOp           *sync.Mutex
	sem            chan struct{}
	_ch            chan Message
	ch             chan Message
	tasks          map[string]Task
	cancelFuncs    map[string]context.CancelFunc
	exitedChannels map[string]chan struct{}
}

var (
	ErrTaskExists    = errors.New("task already exists")
	TaskNotFoundErr  = errors.New("task not found")
	TaskIsRunningErr = errors.New("task is running")
)

func New(concurrency int) (*TaskMan, <-chan Message) {
	ctx, cancel := context.WithCancel(context.Background())
	tm := &TaskMan{
		ctx,
		cancel,
		concurrency,
		&sync.Mutex{},
		&sync.Mutex{},
		make(chan struct{}, concurrency),
		make(chan Message),
		make(chan Message),
		make(map[string]Task),
		make(map[string]context.CancelFunc),
		make(map[string]chan struct{}),
	}

	go func() {
		for {
			select {
			case m := <-tm._ch:
				tm.ch <- m
			}
		}
	}()

	return tm, tm.ch
}

func (tm *TaskMan) Add(t Task) (string, error) {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	for _, task := range tm.tasks {
		if bytes.Equal(task.UniqueChecksum(), t.UniqueChecksum()) {
			return "", ErrTaskExists
		}
	}

	id, _ := uuid.New()
	tm.muMap.Lock()
	tm.tasks[id] = t
	tm.muMap.Unlock()

	return id, nil
}

func (tm *TaskMan) Update(id string, t Task) error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	tm.muMap.Lock()
	_, ok := tm.tasks[id]
	tm.muMap.Unlock()

	if !ok {
		return TaskNotFoundErr
	}

	tm.muMap.Lock()
	_, ok = tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		return TaskIsRunningErr
	}

	tm.muMap.Lock()
	tm.tasks[id] = t
	tm.muMap.Unlock()

	return nil
}

func (tm *TaskMan) Run(id string, state []byte) error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	tm.muMap.Lock()
	t, ok := tm.tasks[id]
	tm.muMap.Unlock()

	if !ok {
		return TaskNotFoundErr
	}

	tm.muMap.Lock()
	_, ok = tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		return TaskIsRunningErr
	}

	ctx, cancel := context.WithCancel(tm.ctx)

	tm.muMap.Lock()
	tm.cancelFuncs[id] = cancel
	tm.exitedChannels[id] = make(chan struct{})
	tm.muMap.Unlock()

	go func() {
		run(ctx, id, state, tm, t)
	}()

	return nil
}

func run(ctx context.Context, id string, state []byte, tm *TaskMan, t Task) {
	chProgress := make(chan int)

	defer func() {
		<-tm.sem
		tm._ch <- newMessage(id, EXITED, nil)
		close(chProgress)
	}()

	go func() {
		for {
			select {
			case p, ok := <-chProgress:
				if !ok {
					// Progress channel is closed when run() is about to exit.
					// End the loop to make goroutine exit.
					return
				}
				tm._ch <- newMessage(id, PROGRESS_UPDATED, p)
			}
		}
	}()

	tm._ch <- newMessage(id, SCHEDULED, nil)
	// Block until other task is done.
	tm.sem <- struct{}{}
	tm._ch <- newMessage(id, STARTED, nil)

	newState, err := t.Run(ctx, state, chProgress)
	if err != nil {
		// Ignore the errors of ctx.Err() after <- ctx.Done():
		if err != context.Canceled && err != context.DeadlineExceeded {
			tm._ch <- newMessage(id, ERROR, err)
		}

		tm._ch <- newMessage(id, STOPPED, newState)
	} else {
		tm._ch <- newMessage(id, DONE, newState)
	}

	tm.muMap.Lock()
	delete(tm.cancelFuncs, id)
	if len(tm.cancelFuncs) == 0 {
		tm._ch <- newMessage("", ALL_EXITED, nil)
	}
	close(tm.exitedChannels[id])
	tm.muMap.Unlock()
}

func (tm *TaskMan) Stop(id string) error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	return tm.stop(id)
}

func (tm *TaskMan) stop(id string) error {
	tm.muMap.Lock()
	_, ok := tm.tasks[id]
	tm.muMap.Unlock()

	if !ok {
		return TaskNotFoundErr
	}

	tm.muMap.Lock()
	cancel, ok := tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		cancel()
	}

	// Get the exit signal channel for the id.
	tm.muMap.Lock()
	exitedChed, ok := tm.exitedChannels[id]
	tm.muMap.Unlock()

	if !ok {
		return nil
	}

	// Block until run() exit and close the exited channel.
	<-exitedChed

	return nil
}

func (tm *TaskMan) StopAll() {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	// Close parent context's Done channel to stop all tasks.
	tm.cancel()

	// Block until all tasks exited.
	<-tm.ctx.Done()
}

func (tm *TaskMan) Delete(id string) error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	err := tm.stop(id)
	if err != nil {
		return err
	}

	tm.muMap.Lock()
	delete(tm.exitedChannels, id)
	delete(tm.tasks, id)
	tm.muMap.Unlock()

	return nil
}
*/

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

func (tm *TaskMan) MsgCh() <-chan Message {
	return tm.msgCh
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
			tm.msgCh <- newMessage(id, STOPPED, savedState)
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

func (tm *TaskMan) Stop(id int64) error {
	tm.taskDatasMu.RLock()
	defer tm.taskDatasMu.RUnlock()

	td, ok := tm.taskDatas[id]

	if !ok {
		return TaskNotFoundErr
	}

	td.cancelFuncMu.RLock()
	if td.cancelFunc != nil {
		td.cancelFunc()
	}
	td.cancelFuncMu.RUnlock()

	return nil
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
