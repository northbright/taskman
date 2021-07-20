package taskman

type Status int

const (
	ADDED = iota
	SCHEDULED
	STARTED
	STOPPED
	REMOVED
	SUSPENDED
	RESUMED
)
