package interfaces

import (
	"context"

	"payment_etl_pipeline/models"
)

type Loader interface {
	Load(ctx context.Context, batches <-chan []models.PaymentEvent) error
}
