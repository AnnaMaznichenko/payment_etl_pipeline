package extractor

import (
	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewWalletExtractor(repo interfaces.PaymentRepository[dbo.PaymentWallet], pageSize int) interfaces.Extractor[dbo.PaymentWallet] {
	return &paymentExtractor[dbo.PaymentWallet]{
		repo:     repo,
		pageSize: pageSize,
		source:   source.Wallets,
	}
}
