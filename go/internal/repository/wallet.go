package repository

import (
	"gorm.io/gorm"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewWalletRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentWallet] {
	return &paymentRepository[dbo.PaymentWallet]{
		db:           db,
		source:       source.Wallets,
		ignoreFields: walletIgnoredFields,
	}
}
