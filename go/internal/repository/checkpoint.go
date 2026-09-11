package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

const (
	colSourceName      = "source_name"
	colLastProcessedAt = "last_processed_at"
	colUpdatedAt       = "updated_at"
)

type checkpointRepository struct {
	db *gorm.DB
}

func NewCheckpointRepository(db *gorm.DB) interfaces.CheckpointRepository {
	return &checkpointRepository{
		db: db,
	}
}

func (r checkpointRepository) Get(ctx context.Context, sources []source.Source) (map[source.Source]time.Time, error) {
	sourceStrings := make([]string, len(sources))
	for i, s := range sources {
		sourceStrings[i] = s.String()
	}

	var checkpoints []dbo.ETLCheckpoint
	err := r.db.WithContext(ctx).
		Where("source_name IN ?", sourceStrings).
		Find(&checkpoints).Error
	if err != nil {
		return nil, fmt.Errorf("get checkpoints: %w", err)
	}

	result := make(map[source.Source]time.Time, len(sources))
	// нулевые значения для первого запроса (пустая таблица)
	for _, s := range sources {
		result[s] = time.Time{}
	}

	for _, cp := range checkpoints {
		s := source.Source(cp.SourceName)
		if cp.LastProcessedAt != nil {
			result[s] = *cp.LastProcessedAt
		}
	}

	return result, nil
}

func (r checkpointRepository) Update(ctx context.Context, updates map[source.Source]time.Time) error {
	if len(updates) == 0 {
		return nil
	}
	checkpoints := make([]dbo.ETLCheckpoint, 0, len(updates))
	for name, t := range updates {
		checkpoints = append(checkpoints, dbo.ETLCheckpoint{
			SourceName:      name.String(),
			LastProcessedAt: &t,
		})
	}

	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: colSourceName}},
			DoUpdates: clause.AssignmentColumns([]string{colLastProcessedAt, colUpdatedAt}),
		}).
		Create(&checkpoints).Error

	if err != nil {
		return fmt.Errorf("update checkpoints: %w", err)
	}

	return nil
}
