package extractor

import (
	"context"
	"time"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
)

type CardExtractor struct {
	repo interfaces.CardRepository
}

func NewCardExtractor(repo interfaces.CardRepository) *CardExtractor {
	return &CardExtractor{repo: repo}
}

func (e CardExtractor) Extract(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentCard, error) {
	// Здесь может быть бесконечный цикл с пагинацией
	// cards, err := e.repo.Find(ctx, interfaces.Query{
	// 	Condition: interfaces.Condition{
	// 		From: &from,
	// 		To: &to,
	// 	},
	// 	Pagination: interfaces.Pagination{
	// 		Limit: 1000,
	// 		Offset: 1,
	// 	},
	// })
	//отправка в канал

	return nil, nil
}
