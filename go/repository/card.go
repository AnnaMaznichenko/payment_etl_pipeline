package repository

import (
	"context"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type CardRepository struct{}

func NewCardRepository() interfaces.CardRepository {
	return &CardRepository{}
}

func (r CardRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentCard, error) {

	return nil, nil
}
