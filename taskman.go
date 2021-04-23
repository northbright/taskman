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
	mutex       *sync.Mutex
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
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for _, task := range tm.tasks {
		if bytes.Equal(task.UniqueChecksum(), t.UniqueChecksum()) {
			return "", ErrTaskExists
		}
	}

	id, _ := uuid.New()
	tm.tasks[id] = t

	return id, nil
}

func (tm *TaskMan) Run(id string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	t, ok := tm.tasks[id]
	if !ok {
		return ErrTaskNotFound
	}

	if _, ok := tm.cancelFuncs[id]; ok {
		return ErrTaskIsRunning
	}

	ctx, cancel := context.WithCancel(context.Background())
	tm.cancelFuncs[id] = cancel

	go func() {
		run(ctx, id, tm, t)
	}()

	return nil
}

func run(ctx context.Context, id string, tm *TaskMan, t Task) {
	defer func() {
		<-tm.sem
		//	tm._ch <-
	}()

	chProgress := make(chan int)

	go t.Run(ctx, chProgress)
	for {
		select {
		case <-ctx.Done():
			return
		case p := <-chProgress:
			m := newMessage(id, PROGRESS_UPDATED, p)
			tm.ch <- m
		}
	}
}

func (tm *TaskMan) Remove(id string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if _, ok := tm.tasks[id]; !ok {
	}
	return nil
}

/*
func (tm *TaskMan) Run(ids []string) error {

}
*/
