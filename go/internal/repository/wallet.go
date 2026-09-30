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

var walletIgnoredFields = []string{"commission_rub"}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) interfaces.PaymentRepository[dbo.PaymentWallet] {
	return &walletRepository{db: db}
}

func (r *walletRepository) Find(ctx context.Context, query interfaces.Query) ([]dbo.PaymentWallet, error) {
	var data []dbo.PaymentWallet
	db := applyQuery(r.db.WithContext(ctx), query)

	if len(walletIgnoredFields) > 0 {
		db = db.Omit(walletIgnoredFields...)
	}

	err := db.Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", source.Wallets, err)
	}
	return data, nil
}

func (r *walletRepository) Upsert(ctx context.Context, wallets []dbo.PaymentWallet) error {
	if len(wallets) == 0 {
		return nil
	}

	deduplicated := deduplicate(wallets, func(w dbo.PaymentWallet) string {
		return w.OperationID
	})

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "operation_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"state", "updated_at"}),
		}).
		Create(&deduplicated).Error

	if err != nil {
		return fmt.Errorf("upsert %s: %w", source.Wallets, err)
	}

	return nil
}
