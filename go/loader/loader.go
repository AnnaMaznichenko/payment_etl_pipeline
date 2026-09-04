package loader

import (
	"context"

	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/models"
)

type Loader struct{}

func NewLoader() interfaces.Loader {
	return &Loader{}
}

func (l Loader) Load(ctx context.Context, batches <-chan []models.PaymentEvent) error {
	return nil
}
