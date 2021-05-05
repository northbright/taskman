package taskman

import (
	"bytes"
	"context"
	"errors"
	//"log"
	"sync"

	"github.com/northbright/uuid"
)

type TaskMan struct {
	ctx         context.Context
	concurrency int
	mu          *sync.Mutex
	sem         chan struct{}
	ch          chan Message
	tasks       map[string]Task
	cancelFuncs map[string]context.CancelFunc
}

var (
	ErrTaskExists    = errors.New("task already exists")
	ErrTaskNotFound  = errors.New("task not found")
	ErrTaskIsRunning = errors.New("task is running")
)

func New(ctx context.Context, concurrency int) (*TaskMan, <-chan Message) {
	tm := &TaskMan{
		ctx,
		concurrency,
		&sync.Mutex{},
		make(chan struct{}, concurrency),
		make(chan Message),
		make(map[string]Task),
		make(map[string]context.CancelFunc),
	}

	return tm, tm.ch
}

func (tm *TaskMan) Add(t Task) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, task := range tm.tasks {
		if bytes.Equal(task.UniqueChecksum(), t.UniqueChecksum()) {
			return "", ErrTaskExists
		}
	}

	id, _ := uuid.New()
	tm.tasks[id] = t

	return id, nil
}

func (tm *TaskMan) Run(id string, state []byte) error {
	tm.mu.Lock()
	t, ok := tm.tasks[id]
	tm.mu.Unlock()

	if !ok {
		return ErrTaskNotFound
	}

	tm.mu.Lock()
	_, ok = tm.cancelFuncs[id]
	tm.mu.Unlock()

	if ok {
		return ErrTaskIsRunning
	}

	ctx, cancel := context.WithCancel(context.Background())

	tm.mu.Lock()
	tm.cancelFuncs[id] = cancel
	tm.mu.Unlock()

	go func() {
		run(ctx, id, state, tm, t)
	}()

	return nil
}

func run(ctx context.Context, id string, state []byte, tm *TaskMan, t Task) {
	chProgress := make(chan int)

	defer func() {
		<-tm.sem
		tm.ch <- newMessage(id, EXITED, nil)
		close(chProgress)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case p, ok := <-chProgress:
				if !ok {
					// Progress channel is closed when run() is about to exit.
					// End the loop to make goroutine exit.
					return
				}
				tm.ch <- newMessage(id, PROGRESS_UPDATED, p)
			}
		}
	}()

	tm.ch <- newMessage(id, SCHEDULED, nil)
	// Block until other task is done.
	tm.sem <- struct{}{}
	tm.ch <- newMessage(id, STARTED, nil)

	newState, err := t.Run(ctx, state, chProgress)
	if err != nil {
		// Ignore the errors of ctx.Err() after <- ctx.Done():
		if err != context.Canceled && err != context.DeadlineExceeded {
			tm.ch <- newMessage(id, ERROR, err)
		}

		tm.ch <- newMessage(id, STOPPED, newState)
	} else {
		tm.ch <- newMessage(id, DONE, newState)
	}

	tm.mu.Lock()
	delete(tm.cancelFuncs, id)
	tm.mu.Unlock()
}

func (tm *TaskMan) Remove(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, ok := tm.tasks[id]; !ok {
		return ErrTaskNotFound
	}

	if cancel, ok := tm.cancelFuncs[id]; ok {
		cancel()
		return nil
	}

	return nil
}
