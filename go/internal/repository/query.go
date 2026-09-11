package repository

import (
	"gorm.io/gorm"

	"payment_etl_pipeline/internal/interfaces"
)

func applyQuery(db *gorm.DB, q interfaces.Query) *gorm.DB {
	if q.Condition.From != nil {
		db = db.Where("updated_at > ?", q.Condition.From)
	}
	if q.Condition.To != nil {
		db = db.Where("updated_at < ?", q.Condition.To)
	}

	if q.Pagination.Limit > 0 {
		db = db.Limit(q.Pagination.Limit)
	}
	if q.Pagination.Offset > 0 {
		db = db.Offset(q.Pagination.Offset)
	}

	db = db.Order("updated_at, id")

	return db
}
