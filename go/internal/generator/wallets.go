package generator

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

const (
	minAmountRub   = 100.0
	maxAmountRub   = 50_000.0
	commissionRate = 0.01
)

type walletsGenerator struct {
	repo     interfaces.PaymentRepository[dbo.PaymentWallet]
	interval time.Duration
	mu       sync.Mutex
	ids      []string
}

func NewWalletsGenerator(repo interfaces.PaymentRepository[dbo.PaymentWallet], interval time.Duration) interfaces.SourceGenerator {
	return &walletsGenerator{
		repo:     repo,
		interval: interval,
	}
}

func (g *walletsGenerator) Name() source.Source {
	return source.Wallets
}

func (g *walletsGenerator) Interval() time.Duration {
	return g.interval
}

func (g *walletsGenerator) Generate(ctx context.Context) error {
	count := rand.IntN(maxNewPerTick) + 1
	wallets := make([]dbo.PaymentWallet, 0, count)

	g.mu.Lock()
	availableForUpdate := make([]string, len(g.ids))
	copy(availableForUpdate, g.ids)
	g.mu.Unlock()

	for i := 0; i < count; i++ {
		if len(availableForUpdate) > 0 && rand.Float64() < updateChance {
			// Обновляем статус
			idx := rand.IntN(len(availableForUpdate))
			id := availableForUpdate[idx]
			availableForUpdate = append(availableForUpdate[:idx], availableForUpdate[idx+1:]...)
			wallet := dbo.PaymentWallet{
				OperationID: id,
				State:       randomFinalWalletStatus(),
				UpdatedAt:   time.Now().UTC(),
			}
			wallets = append(wallets, wallet)
		} else {
			// Создаём новую запись
			wallet := newRandomWallet()
			g.mu.Lock()
			g.ids = append(g.ids, wallet.OperationID)
			g.mu.Unlock()
			wallets = append(wallets, wallet)
		}
	}

	if err := g.repo.Upsert(ctx, wallets); err != nil {
		return fmt.Errorf("upsert wallets: %w", err)
	}

	return nil
}

func newRandomWallet() dbo.PaymentWallet {
	amount := randomAmountRub()
	return dbo.PaymentWallet{
		OperationID:   randomHex(12, "op-"),
		UserPhone:     randomPhone(),
		AmountRub:     amount,
		CommissionRub: round2(amount * commissionRate),
		State:         0,
	}
}

func randomPhone() string {
	return fmt.Sprintf("+79%09d", rand.IntN(1_000_000_000))
}

func randomAmountRub() float64 {
	amount := minAmountRub + rand.Float64()*(maxAmountRub-minAmountRub)
	return round2(amount)
}

func randomFinalWalletStatus() int {
	if rand.Float64() < 0.8 {
		return 1
	}
	return 2
}
