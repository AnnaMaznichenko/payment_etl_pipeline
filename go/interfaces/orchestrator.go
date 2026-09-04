package interfaces

import (
	"context"
)

type Orchestrator interface {
	Run(ctx context.Context) error
}
