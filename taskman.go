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

var (
	taskMans   = make(map[string]func(data []byte) Task)
	taskMansMu = &sync.RWMutex{}

	NoSuchTaskNameErr = errors.New("no such task name")
	TaskNotFoundErr   = errors.New("task not found")
	TaskIsRunningErr  = errors.New("task is running")
)

type TaskData struct {
	task        Task
	ctx         context.Context
	cancelFunc  context.CancelFunc
	pausedCh    chan bool
	exitedCh    chan struct{}
	showPercent bool
	total       int64
}

type TaskMan struct {
	ctx         context.Context
	name        string
	concurrency int
	newTask     func(data []byte) Task
	taskDatas   map[int64]*TaskData
	taskDatasMu *sync.RWMutex
	maxID       int64
	total       int64
	sem         chan struct{}
	msgCh       chan Message
}

func Register(name string, f func(data []byte) Task) {
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

	tm := &TaskMan{
		ctx:         context.Background(),
		name:        name,
		concurrency: concurrency,
		newTask:     taskMans[name],
		taskDatas:   make(map[int64]*TaskData),
		taskDatasMu: &sync.RWMutex{},
		maxID:       0,
		total:       0,
		sem:         make(chan struct{}, concurrency),
		msgCh:       make(chan Message),
	}

	return tm, tm.msgCh, nil
}

func (tm *TaskMan) MsgCh() <-chan Message {
	return tm.msgCh
}

func (tm *TaskMan) Add(data []byte) (int64, error) {
	t := tm.newTask(data)

	var total int64
	spt, ok := t.(ShowPercentTask)
	if ok {
		total = spt.Total()
		atomic.AddInt64(&tm.total, total)
	}

	id := atomic.AddInt64(&tm.maxID, 1)
	tm.taskDatasMu.Lock()
	td := &TaskData{
		task:        t,
		ctx:         nil,
		cancelFunc:  nil,
		pausedCh:    nil,
		exitedCh:    nil,
		showPercent: ok,
		total:       total,
	}
	tm.taskDatas[id] = td
	tm.taskDatasMu.Unlock()

	return id, nil
}

func (tm *TaskMan) Start(id int64) error {
	tm.taskDatasMu.RLock()
	td, ok := tm.taskDatas[id]
	tm.taskDatasMu.RUnlock()

	if !ok {
		return TaskNotFoundErr
	}

	if td.cancelFunc != nil {
		return TaskIsRunningErr
	}

	ctx, cancelFunc := context.WithCancel(tm.ctx)

	tm.taskDatasMu.Lock()
	tm.taskDatas[id].cancelFunc = cancelFunc
	tm.taskDatas[id].pausedCh = make(chan bool)
	tm.taskDatas[id].exitedCh = make(chan struct{})
	tm.taskDatasMu.Unlock()

	go func() {
		tm.run(ctx, id, td)
	}()
	return nil
}

func (tm *TaskMan) run(ctx context.Context, id int64, td *TaskData) {
	tm.msgCh <- newMessage(id, SCHEDULED, nil)

	tm.sem <- struct{}{}
	defer func() {
		<-tm.sem
		tm.taskDatasMu.Lock()
		tm.taskDatas[id].cancelFunc = nil
		tm.taskDatasMu.Unlock()
		tm.msgCh <- newMessage(id, EXITED, nil)
	}()

	var (
		paused  bool
		percent float64
	)

	tm.msgCh <- newMessage(id, STARTED, nil)

	for {
		select {
		case <-ctx.Done():
			state, err := td.task.MarshalBinary()
			if err != nil {
				tm.msgCh <- newMessage(id, ERROR, err)
				return
			}

			tm.msgCh <- newMessage(id, STOPPED, state)
			return
		case paused = <-td.pausedCh:
		default:
			runtime.Gosched()

			if !paused {
				current, done, err := td.task.Step()
				if err != nil {
					tm.msgCh <- newMessage(id, ERROR, err)
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
					state, err := td.task.MarshalBinary()
					if err != nil {
						tm.msgCh <- newMessage(id, ERROR, err)
						return
					}

					tm.msgCh <- newMessage(id, DONE, state)
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
	td, ok := tm.taskDatas[id]
	tm.taskDatasMu.RUnlock()

	if !ok {
		return TaskNotFoundErr
	}

	if td.cancelFunc != nil {
		td.cancelFunc()
	}

	return nil
}
