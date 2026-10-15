package interfaces

import (
	"context"
	"time"

	"payment_etl_pipeline/internal/source"
)

type Condition struct {
	From *time.Time
	To   *time.Time
}

type Pagination struct {
	Limit  int
	Offset int
}

type Query struct {
	Condition  Condition
	Pagination Pagination
}

//go:generate mockery --name=PaymentRepository --output=../mocks --outpkg=mocks --with-expecter

type PaymentRepository[T any] interface {
	Find(ctx context.Context, query Query) ([]T, error)
	Upsert(ctx context.Context, items []T) error
}

//go:generate mockery --name=CheckpointRepository --output=../mocks --outpkg=mocks --with-expecter

type CheckpointRepository interface {
	Get(ctx context.Context, sources []source.Source) (map[source.Source]time.Time, error)
	Update(ctx context.Context, updates map[source.Source]time.Time) error
}
