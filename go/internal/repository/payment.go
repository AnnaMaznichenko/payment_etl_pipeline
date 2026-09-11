package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

var (
	cardIgnoredFields   = []string{"gateway_response"}
	cryptoIgnoredFields = []string{"confirmations"}
	walletIgnoredFields = []string{"commission_rub"}
)

type paymentRepository[T any] struct {
	db           *gorm.DB
	source       source.Source
	ignoreFields []string
}

func (r paymentRepository[T]) Find(ctx context.Context, query interfaces.Query) ([]T, error) {
	var data []T
	db := applyQuery(r.db.WithContext(ctx), query)

	if len(r.ignoreFields) > 0 {
		db = db.Omit(r.ignoreFields...)
	}

	err := db.Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("find %s: %w", r.source, err)
	}
	return data, nil
}
