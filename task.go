package taskman

import (
	"context"
	"encoding"
)

type Task interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	Init(ctx context.Context) (int64, error)
	Deinit(ctx context.Context) error
	// Return values:
	// int64: next step index
	// bool: if task is done
	// []byte: result when it's done
	// error
	Step(ctx context.Context, current int64) (int64, bool, []byte, error)
}
