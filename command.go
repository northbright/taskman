package taskman

import (
//"encoding/json"
)

const (
	ADD     = "add"
	START   = "start"
	STOP    = "stop"
	REMOVE  = "remove"
	SUSPEND = "suspend"
	RESUME  = "resume"
)

var (
	InvalidCommandErr = "invalid command"
)

type Command interface {
	Type() string
}

type AddCommand struct {
	Data []byte `json:"data"`
}

func (c *AddCommand) Type() string {
	return ADD
}

type StartCommand struct {
	ID    string `json:"id"`
	State []byte `json:"state"`
}

func (c *StartCommand) Type() string {
	return START
}

type StopCommand struct {
	ID string `json:"id"`
}

func (c *StopCommand) Type() string {
	return STOP
}

type RemoveCommand struct {
	ID string `json:"id"`
}

func (c *RemoveCommand) Type() string {
	return REMOVE
}

type SuspendCommand struct {
	ID string `json:"id"`
}

func (c *SuspendCommand) Type() string {
	return SUSPEND
}

type ResumeCommand struct {
	ID string `json:"id"`
}

func (c *ResumeCommand) Type() string {
	return RESUME
}
