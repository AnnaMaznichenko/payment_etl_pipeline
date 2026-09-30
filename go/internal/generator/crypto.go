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
	minAmountBtc = 0.0001
	maxAmountBtc = 0.5
)

type cryptoGenerator struct {
	repo     interfaces.PaymentRepository[dbo.PaymentCrypto]
	interval time.Duration
	mu       sync.Mutex
	txHashes []string
}

func NewCryptoGenerator(repo interfaces.PaymentRepository[dbo.PaymentCrypto], interval time.Duration) interfaces.SourceGenerator {
	return &cryptoGenerator{
		repo:     repo,
		interval: interval,
	}
}

func (g *cryptoGenerator) Name() source.Source {
	return source.Crypto
}

func (g *cryptoGenerator) Interval() time.Duration {
	return g.interval
}

func (g *cryptoGenerator) Generate(ctx context.Context) error {
	count := rand.IntN(maxNewPerTick) + 1
	cryptos := make([]dbo.PaymentCrypto, 0, count)

	g.mu.Lock()
	availableForUpdate := make([]string, len(g.txHashes))
	copy(availableForUpdate, g.txHashes)
	g.mu.Unlock()

	for i := 0; i < count; i++ {
		if len(availableForUpdate) > 0 && rand.Float64() < updateChance {
			// Обновляем статус
			idx := rand.IntN(len(availableForUpdate))
			txHash := availableForUpdate[idx]
			availableForUpdate = append(availableForUpdate[:idx], availableForUpdate[idx+1:]...)
			crypto := dbo.PaymentCrypto{
				TxHash:     txHash,
				StatusCode: randomFinalCryptoStatus(),
				UpdatedAt:  time.Now().UTC(),
			}
			cryptos = append(cryptos, crypto)
		} else {
			// Создаём новую запись
			crypto := newRandomCrypto()
			g.mu.Lock()
			g.txHashes = append(g.txHashes, crypto.TxHash)
			g.mu.Unlock()
			cryptos = append(cryptos, crypto)
		}
	}

	if err := g.repo.Upsert(ctx, cryptos); err != nil {
		return fmt.Errorf("upsert crypto: %w", err)
	}

	return nil
}

func newRandomCrypto() dbo.PaymentCrypto {
	return dbo.PaymentCrypto{
		TxHash:        randomHex(64, "0x"),
		WalletAddress: randomHex(40, "0x"),
		AmountBtc:     randomAmountBtc(),
		Confirmations: rand.IntN(20),
		StatusCode:    "pend",
	}
}

func randomAmountBtc() float64 {
	amount := minAmountBtc + rand.Float64()*(maxAmountBtc-minAmountBtc)
	return float64(int64(amount*1e8)) / 1e8
}

func randomFinalCryptoStatus() string {
	if rand.Float64() < 0.8 {
		return "done"
	}
	return "fail"
}
