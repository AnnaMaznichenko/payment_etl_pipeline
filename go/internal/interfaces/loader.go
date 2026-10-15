package interfaces

import (
	"context"

	"payment_etl_pipeline/internal/models"
)

//go:generate mockery --name=Loader --output=../mocks --outpkg=mocks --with-expecter

type Loader interface {
	Load(ctx context.Context, batches <-chan []models.PaymentEvent) error
}
