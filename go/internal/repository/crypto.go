package repository

import (
	"gorm.io/gorm"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewCryptoRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentCrypto] {
	return &paymentRepository[dbo.PaymentCrypto]{
		db:           db,
		source:       source.Crypto,
		ignoreFields: cryptoIgnoredFields,
	}
}
