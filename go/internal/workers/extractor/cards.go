package extractor

import (
	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

func NewCardExtractor(repo interfaces.PaymentRepository[dbo.PaymentCard], pageSize int) interfaces.Extractor[dbo.PaymentCard] {
	return &paymentExtractor[dbo.PaymentCard]{
		repo:     repo,
		pageSize: pageSize,
		source:   source.Cards,
	}
}
