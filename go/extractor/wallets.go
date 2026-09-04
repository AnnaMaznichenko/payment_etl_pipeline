package extractor

import (
	"context"
	"time"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type WalletExtractor struct {
	repo interfaces.WalletRepository
}

func NewWalletExtractor(repo interfaces.WalletRepository) *WalletExtractor {
	return &WalletExtractor{repo: repo}
}

func (e WalletExtractor) Extract(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentWallet, error) {

	return nil, nil
}
