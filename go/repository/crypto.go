package repository

import (
	"context"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type CryptoRepository struct{}

func NewCryptoRepository() interfaces.CryptoRepository {
	return &CryptoRepository{}
}

func (r CryptoRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentCrypto, error) {

	return nil, nil
}
