package interfaces

import (
	"context"
	"time"
)

//go:generate mockery --name=Extractor --output=../mocks --outpkg=mocks --with-expecter

type Extractor[T any] interface {
	Extract(ctx context.Context, from, to time.Time) (<-chan T, <-chan error)
}
