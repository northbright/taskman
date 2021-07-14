package taskman

type CommandType uint

const (
	ADD = iota
	START
	STOP
	DELETE
	SUSPEND
	RESUME
	maxCmdTYPE
)

type Command struct {
	Type CommandType `json:"type"`
	ID   int         `json:"task_id"`
	Data []byte      `json:"data"`
}
