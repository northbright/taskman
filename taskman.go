package taskman

import (
	//"bytes"
	//"context"
	"errors"
	//"log"
	"runtime"
	"sync"
	"sync/atomic"
)

const (
	DefaultConcurrency = 1
)

type OnStatusChanged func(env interface{}, id int, status Status, state []byte, result []byte)

type OnProgressUpdated func(env interface{}, id int, current, total, currentAll, totalAll int64, progress, progressAll float32)

type OnError func(env interface{}, id int, err error)

var (
	NoSuchTaskNameErr     = errors.New("no such task name")
	TaskNotFoundErr       = errors.New("task not found")
	TaskIsRunningErr      = errors.New("task is running")
	TaskIsStoppedErr      = errors.New("task is stopped")
	InvalidCommand        = errors.New("invalid command")
	InvadliStatusToChange = errors.New("invalid status to change")
)

type taskData struct {
	task        Task
	removed     bool
	removedMu   *sync.RWMutex
	status      Status
	statusMu    *sync.RWMutex
	scheduledCh chan struct{}
	toStopCh    chan struct{}
	stoppedCh   chan struct{}
	toSuspendCh chan bool
}

func (td *taskData) setStatus(status Status) {
	td.statusMu.Lock()
	td.status = status
	td.statusMu.Unlock()
}

func (td *taskData) getStatus() Status {
	td.statusMu.RLock()
	status := td.status
	td.statusMu.RUnlock()

	return status
}

func (td *taskData) SetRemoved(removed bool) {
	td.removedMu.Lock()
	td.removed = removed
	td.removedMu.Unlock()
}

func (td *taskData) getRemoved() bool {
	td.removedMu.RLock()
	removed := td.removed
	td.removedMu.RUnlock()

	return removed
}

type TaskMan struct {
	concurrency       int
	runningTasksNum   int32
	env               interface{}
	onStatusChanged   OnStatusChanged
	onProgressUpdated OnProgressUpdated
	onError           OnError
	maxID             int32
	sem               chan struct{}
	statusCh          chan Status
	taskDatas         map[int]*taskData
	taskDatasMu       *sync.RWMutex
	current           int64
	total             int64
}

func New(concurrency int, env interface{}, onStatusChanged OnStatusChanged, onProgressUpdated OnProgressUpdated, onError OnError) (*TaskMan, error) {
	if concurrency <= 0 {
		concurrency = DefaultConcurrency
	}

	tm := &TaskMan{
		concurrency:       concurrency,
		runningTasksNum:   0,
		env:               env,
		onStatusChanged:   onStatusChanged,
		onProgressUpdated: onProgressUpdated,
		onError:           onError,
		maxID:             0,
		sem:               make(chan struct{}, concurrency),
		statusCh:          make(chan Status),
		taskDatas:         make(map[int]*taskData),
		taskDatasMu:       &sync.RWMutex{},
		current:           0,
		total:             0,
	}

	return tm, nil
}

func (tm *TaskMan) Add(t Task) (int, error) {
	tm.taskDatasMu.Lock()
	td := &taskData{
		task:        t,
		removed:     false,
		removedMu:   &sync.RWMutex{},
		status:      CREATED,
		statusMu:    &sync.RWMutex{},
		scheduledCh: make(chan struct{}),
		toStopCh:    make(chan struct{}),
		stoppedCh:   make(chan struct{}),
		toSuspendCh: make(chan bool),
	}

	id := int(atomic.AddInt32(&tm.maxID, 1))

	tm.taskDatas[id] = td
	tm.taskDatasMu.Unlock()

	return id, nil
}

func computeProgress(current, total int64) float32 {
	if total > 0 {
		return float32(float64(current) / (float64(total) / float64(100)))
	}
	return 0
}

func (tm *TaskMan) run(id int, state []byte, td *taskData) {
	var (
		err         error
		suspended   bool
		current     int64
		oldCurrent  int64
		total       int64
		progress    float32
		oldProgress float32
		done        bool
		savedState  []byte
		result      []byte
	)

	defer func() {
		<-tm.sem
	}()

	td.setStatus(SCHEDULED)
	tm.onStatusChanged(tm.env, id, SCHEDULED, nil, nil)
	td.scheduledCh <- struct{}{}

	tm.sem <- struct{}{}

	// Init the task.
	if current, total, err = td.task.Init(state); err != nil {
		tm.onError(tm.env, id, err)
		return
	}

	atomic.AddInt64(&tm.total, total)
	atomic.AddInt32(&tm.runningTasksNum, 1)

	for {
		select {
		case <-td.toStopCh:
			// Update current, total for all tasks when task stopped.
			atomic.AddInt64(&tm.current, -current)
			atomic.AddInt64(&tm.total, -total)

			// Save state when task stopped.
			if savedState, err = td.task.Save(); err != nil {
				tm.onError(tm.env, id, err)
			}

			td.setStatus(STOPPED)

			tm.onStatusChanged(tm.env, id, STOPPED, savedState, nil)
			td.stoppedCh <- struct{}{}
			return

		case suspended = <-td.toSuspendCh:
			if suspended {
				td.setStatus(SUSPENDED)
			} else {
				td.setStatus(STARTED)
			}
			tm.onStatusChanged(tm.env, id, td.status, nil, nil)

		default:
			runtime.Gosched()

			if suspended {
				continue
			}

			current, done, result, err = td.task.Step()
			if err != nil {
				tm.onError(tm.env, id, err)
				return
			}

			atomic.AddInt64(&tm.current, current-oldCurrent)
			oldCurrent = current

			if total > 0 {
				progress = computeProgress(current, total)
				if progress != oldProgress {
					oldProgress = progress
					progressAll := computeProgress(tm.current, tm.total)
					tm.onProgressUpdated(tm.env, id, current, total,
						tm.current, tm.total, progress, progressAll)
				}
			}

			if done {
				// Save final state.
				if savedState, err = td.task.Save(); err != nil {
					tm.onError(tm.env, id, err)
					return
				}

				// Deinit task.
				if err = td.task.Deinit(); err != nil {
					tm.onError(tm.env, id, err)
					return
				}

				td.setStatus(DONE)
				tm.onStatusChanged(tm.env, id, DONE, savedState, result)

				return
			}

		}
	}
}

func (tm *TaskMan) get(id int) (*taskData, error) {
	tm.taskDatasMu.RLock()
	defer tm.taskDatasMu.RUnlock()

	td, ok := tm.taskDatas[id]
	if !ok {
		return nil, TaskNotFoundErr
	}

	if td.getRemoved() {
		return nil, TaskNotFoundErr
	}

	return td, nil
}

func (tm *TaskMan) Start(id int, state []byte) error {
	td, err := tm.get(id)
	if err != nil {
		return err
	}

	prevStatus := td.getStatus()
	if !ValidStatusToChange(prevStatus, SCHEDULED) {
		return InvadliStatusToChange
	}

	go func() {
		tm.run(id, state, td)
	}()

	<-td.scheduledCh
	return nil
}
