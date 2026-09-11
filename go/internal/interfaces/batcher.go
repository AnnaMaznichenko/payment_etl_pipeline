package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

type Batcher interface {
	Batch(ctx context.Context, events <-chan models.PaymentEvent) <-chan []models.PaymentEvent
}
