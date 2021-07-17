package taskman

import (
	//"bytes"
	//"context"
	"errors"
	//"log"
	//"runtime"
	"sync"
	//"sync/atomic"
)

const (
	DefConcurrency = 16
)

type NewTaskFunc func(data []byte) (Task, error)

var (
	taskFuncs   = make(map[string]NewTaskFunc)
	taskFuncsMu = &sync.RWMutex{}

	NoSuchTaskNameErr = errors.New("no such task name")
	TaskNotFoundErr   = errors.New("task not found")
	TaskIsRunningErr  = errors.New("task is running")
	TaskIsStoppedErr  = errors.New("task is stopped")
)
