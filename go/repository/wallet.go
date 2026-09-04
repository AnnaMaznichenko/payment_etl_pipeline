package repository

import (
	"context"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type WalletRepository struct{}

func NewWalletRepository() interfaces.WalletRepository {
	return &WalletRepository{}
}

func (r WalletRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentWallet, error) {

	return nil, nil
}
