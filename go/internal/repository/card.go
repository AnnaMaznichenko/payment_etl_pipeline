package repository

import (
	"gorm.io/gorm"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewCardRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentCard] {
	return &paymentRepository[dbo.PaymentCard]{
		db:           db,
		source:       source.Cards,
		ignoreFields: cardIgnoredFields,
	}
}
