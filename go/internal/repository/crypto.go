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

var cryptoIgnoredFields = []string{"confirmations"}

type cryptoRepository struct {
	db *gorm.DB
}

func NewCryptoRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentCrypto] {
	return &cryptoRepository{db: db}
}

func (r *cryptoRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentCrypto, error) {
	var data []dbo.PaymentCrypto
	db := applyQuery(r.db.WithContext(ctx), query)

	if len(cryptoIgnoredFields) > 0 {
		db = db.Omit(cryptoIgnoredFields...)
	}

	err := db.Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", source.Crypto, err)
	}
	return data, nil
}

func (r *cryptoRepository) Upsert(ctx context.Context, crypto []dbo.PaymentCrypto) error {
	if len(crypto) == 0 {
		return nil
	}

	deduplicated := deduplicate(crypto, func(c dbo.PaymentCrypto) string {
		return c.TxHash
	})

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tx_hash"}},
			DoUpdates: clause.AssignmentColumns([]string{"status_code", "updated_at"}),
		}).
		Create(&deduplicated).Error

	if err != nil {
		return fmt.Errorf("upsert %s: %w", source.Crypto, err)
	}

	return nil
}
