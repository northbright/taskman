package taskman

import (
	//"bytes"
	//"context"
	"errors"
	//"log"
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
	ErrTaskNotFound  = errors.New("task not found")
	ErrTaskIsRunning = errors.New("task is running")
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
		return ErrTaskNotFound
	}

	tm.muMap.Lock()
	_, ok = tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		return ErrTaskIsRunning
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
		return ErrTaskNotFound
	}

	tm.muMap.Lock()
	_, ok = tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		return ErrTaskIsRunning
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
		return ErrTaskNotFound
	}

	tm.muMap.Lock()
	cancel, ok := tm.cancelFuncs[id]
	tm.muMap.Unlock()

	if ok {
		cancel()
	}

	// Get the exit signal channel for the id.
	tm.muMap.Lock()
	chExited, ok := tm.exitedChannels[id]
	tm.muMap.Unlock()

	if !ok {
		return nil
	}

	// Block until run() exit and close the exited channel.
	<-chExited

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

	errNoSuchTaskName = errors.New("no such task name")
)

type TaskMan struct {
	name    string
	newTask func(data []byte) Task
	tasks   map[uint64]Task
	tasksMu *sync.RWMutex
	maxID   uint64
	total   uint64
	chMsg   chan Message
}

func Register(name string, f func(data []byte) Task) {
	if _, ok := taskMans[name]; ok {
		panic("taskman: Register twice for: " + name)
	}

	taskMansMu.Lock()
	taskMans[name] = f
	taskMansMu.Unlock()
}

func New(name string) (*TaskMan, error) {
	taskMansMu.RLock()
	_, ok := taskMans[name]
	taskMansMu.RUnlock()
	if !ok {
		return nil, errNoSuchTaskName
	}

	tm := &TaskMan{
		name:    name,
		newTask: taskMans[name],
		tasks:   make(map[uint64]Task),
		tasksMu: &sync.RWMutex{},
		maxID:   0,
		total:   0,
		chMsg:   make(chan Message),
	}

	return tm, nil
}

func (tm *TaskMan) Add(data []byte) (uint64, error) {
	t := tm.newTask(data)

	id := atomic.AddUint64(&tm.maxID, 1)
	tm.tasksMu.Lock()
	tm.tasks[id] = t
	tm.tasksMu.Unlock()

	pt, ok := t.(ProgressiveTask)
	if ok {
		atomic.AddUint64(&tm.total, pt.Total())
	}

	return id, nil
}
