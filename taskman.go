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
	// Make map goroutine safe
	muMap *sync.Mutex
	// Make operation goroutine safe
	muOp         *sync.Mutex
	sem          chan struct{}
	ch           chan Message
	tasks        map[string]Task
	cancelFuncs  map[string]context.CancelFunc
	exitChannels map[string]chan struct{}
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
		&sync.Mutex{},
		make(chan struct{}, concurrency),
		make(chan Message),
		make(map[string]Task),
		make(map[string]context.CancelFunc),
		make(map[string]chan struct{}),
	}

	return tm, tm.ch
}

func (tm *TaskMan) postMessage(m Message) {
	go func() {
		tm.ch <- m
	}()
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

	ctx, cancel := context.WithCancel(context.Background())

	tm.muMap.Lock()
	tm.cancelFuncs[id] = cancel
	tm.exitChannels[id] = make(chan struct{})
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
		tm.postMessage(newMessage(id, EXITED, nil))
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
				tm.postMessage(newMessage(id, PROGRESS_UPDATED, p))
			}
		}
	}()

	tm.postMessage(newMessage(id, SCHEDULED, nil))
	// Block until other task is done.
	tm.sem <- struct{}{}
	tm.postMessage(newMessage(id, STARTED, nil))

	newState, err := t.Run(ctx, state, chProgress)
	if err != nil {
		// Ignore the errors of ctx.Err() after <- ctx.Done():
		if err != context.Canceled && err != context.DeadlineExceeded {
			tm.postMessage(newMessage(id, ERROR, err))
		}

		tm.postMessage(newMessage(id, STOPPED, newState))
	} else {
		tm.postMessage(newMessage(id, DONE, newState))
	}

	tm.muMap.Lock()
	delete(tm.cancelFuncs, id)
	if len(tm.cancelFuncs) == 0 {
		tm.postMessage(newMessage("", ALL_EXITED, nil))
	}
	close(tm.exitChannels[id])
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
	chExit, ok := tm.exitChannels[id]
	tm.muMap.Unlock()

	if !ok {
		return nil
	}

	// Block until run() exit and close the exit channel.
	_, ok = <-chExit

	return nil
}

func (tm *TaskMan) StopAll() error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	ids := []string{}

	// Copy task IDs.
	tm.muMap.Lock()
	for id, _ := range tm.tasks {
		ids = append(ids, id)
	}
	tm.muMap.Unlock()

	for _, id := range ids {
		if err := tm.stop(id); err != nil {
			return err
		}
	}

	return nil
}

func (tm *TaskMan) Delete(id string) error {
	tm.muOp.Lock()
	defer tm.muOp.Unlock()

	err := tm.stop(id)
	if err != nil {
		return err
	}

	tm.muMap.Lock()
	delete(tm.exitChannels, id)
	delete(tm.tasks, id)
	tm.muMap.Unlock()

	return nil
}
