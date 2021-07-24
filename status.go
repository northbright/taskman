package taskman

type Status int

const (
	CREATED Status = iota
	SCHEDULED
	STARTED
	STOPPED
	SUSPENDED
	DONE
)

func ValidStatusToChange(prevStatus, statusToChange Status) bool {
	switch statusToChange {
	case CREATED, DONE:
		return false
	case SCHEDULED:
		if prevStatus != CREATED && prevStatus != STOPPED {
			return false
		}
	case STARTED:
		if prevStatus != SCHEDULED && prevStatus != SUSPENDED {
			return false
		}
	case STOPPED:
		if prevStatus != SCHEDULED && prevStatus != STARTED && prevStatus != SUSPENDED {
			return false
		}
	case SUSPENDED:
		if prevStatus != STARTED {
			return false
		}
	}

	return true
}
