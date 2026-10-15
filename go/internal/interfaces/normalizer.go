package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

//go:generate mockery --name=Normalizer --output=../mocks --outpkg=mocks --with-expecter

type Normalizer[T any] interface {
	Normalize(ctx context.Context, rawCh <-chan T) <-chan models.PaymentEvent
}
