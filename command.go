package taskman

type Operation int

const (
	ADD Operation = iota
	REMOVE
	START
	STOP
	SUSPEND
	RESUME
)

type OnCommandOK func(env interface{}, cmd Command)
type OnCommandError func(env interface{}, cmd Command, err error)

type Command struct {
	ID      int
	OP      Operation
	Data    []byte
	OnOK    OnCommandOK
	OnError OnCommandError
}
