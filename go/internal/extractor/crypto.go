package extractor

import (
	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewCryptoExtractor(repo interfaces.PaymentRepository[dbo.PaymentCrypto], pageSize int) interfaces.Extractor[dbo.PaymentCrypto] {
	return &paymentExtractor[dbo.PaymentCrypto]{
		repo:     repo,
		pageSize: pageSize,
		source:   source.Crypto,
	}
}
