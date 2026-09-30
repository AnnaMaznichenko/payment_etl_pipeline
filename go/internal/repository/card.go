package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

var cardIgnoredFields = []string{"gateway_response"}

type cardRepository struct {
	db *gorm.DB
}

func NewCardRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentCard] {
	return &cardRepository{db: db}
}

func (r *cardRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentCard, error) {
	var data []dbo.PaymentCard
	db := applyQuery(r.db.WithContext(ctx), query)

	if len(cardIgnoredFields) > 0 {
		db = db.Omit(cardIgnoredFields...)
	}

	err := db.Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", source.Cards, err)
	}
	return data, nil
}

func (r *cardRepository) Upsert(ctx context.Context, cards []dbo.PaymentCard) error {
	if len(cards) == 0 {
		return nil
	}

	deduplicated := deduplicate(cards, func(c dbo.PaymentCard) string {
		return c.TransactionID
	})

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "transaction_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"payment_status", "updated_at"}),
		}).
		Create(&deduplicated).Error

	if err != nil {
		return fmt.Errorf("upsert %s: %w", source.Cards, err)
	}

	return nil
}
