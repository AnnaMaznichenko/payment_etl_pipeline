package interfaces

import (
	"context"
	"time"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/source"
)

type Condition struct {
	From *time.Time
	To   *time.Time
}

type Pagination struct {
	Limit  int
	Offset int
}

type Query struct {
	Condition  Condition
	Pagination Pagination
}

type CardRepository interface {
	Find(ctx context.Context, query Query) ([]dbo.PaymentCard, error)
}

type CryptoRepository interface {
	Find(ctx context.Context, query Query) ([]dbo.PaymentCrypto, error)
}

type WalletRepository interface {
	Find(ctx context.Context, query Query) ([]dbo.PaymentWallet, error)
}

type CheckpointRepository interface {
	Get(ctx context.Context, sources []source.Source) (map[source.Source]time.Time, error)
	Update(ctx context.Context, updates map[source.Source]time.Time) error
}
