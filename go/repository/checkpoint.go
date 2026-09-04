package repository

import (
	"context"
	"time"

	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/source"
)

type CheckpointRepository struct{}

func NewCheckpointRepository() interfaces.CheckpointRepository {
	return &CheckpointRepository{}
}

func (r CheckpointRepository) Get(ctx context.Context, sources []source.Source) (map[source.Source]time.Time, error) {
	return nil, nil
}

func (r CheckpointRepository) Update(ctx context.Context, updates map[source.Source]time.Time) error {
	return nil
}
