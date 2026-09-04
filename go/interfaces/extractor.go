package interfaces

import (
	"context"
	"time"
)

type Extractor[T any] interface {
	Extract(ctx context.Context, from, to time.Time) (<-chan T, error)
}
