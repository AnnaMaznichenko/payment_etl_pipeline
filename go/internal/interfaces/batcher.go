package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

//go:generate mockery --name=Batcher --output=../mocks --outpkg=mocks --with-expecter

type Batcher interface {
	Batch(ctx context.Context, events <-chan models.PaymentEvent) <-chan []models.PaymentEvent
}
