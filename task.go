package taskman

import (
	"context"
	"encoding"
)

type Task interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	UniqueChecksum() []byte
	Run(ctx context.Context, chProgress chan<- int) error
}
