package interfaces

import (
	"context"
	"time"

	"payment_etl_pipeline/internal/source"
)

type Generator interface {
	Run(ctx context.Context) error
}

type SourceGenerator interface {
	Name() source.Source
	Interval() time.Duration
	Generate(ctx context.Context) error
}
