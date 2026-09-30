package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

type Loader interface {
	Load(ctx context.Context, batches <-chan []models.PaymentEvent) error
}
