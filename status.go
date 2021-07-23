package taskman

type Status int

const (
	SCHEDULED = iota
	STARTED
	STOPPED
	SUSPENDED
	RESUMED
	DONE
)
