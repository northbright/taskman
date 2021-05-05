package taskman

import (
	"context"
)

type Task interface {
	UniqueChecksum() []byte
	Run(ctx context.Context, state []byte, chProgress chan<- int) ([]byte, error)
}
