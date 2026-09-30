package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

type Normalizer[T any] interface {
	Normalize(ctx context.Context, rawCh <-chan T) <-chan models.PaymentEvent
}
