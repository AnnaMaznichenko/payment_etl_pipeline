package extractor

import (
	"context"
	"time"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type CryptoExtractor struct {
	repo interfaces.CryptoRepository
}

func NewCCryptoExtractor(repo interfaces.CryptoRepository) *CryptoExtractor {
	return &CryptoExtractor{repo: repo}
}

func (e CryptoExtractor) Extract(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentCrypto, error) {

	return nil, nil
}
