package generator

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

const (
	minAmountKopecks = 1_000
	maxAmountKopecks = 500_000
)

var cardCurrencies = []string{"RUB", "USD"}

type cardsGenerator struct {
	repo     interfaces.PaymentRepository[dbo.PaymentCard]
	interval time.Duration
	mu       sync.Mutex
	ids      []string
}

func NewCardsGenerator(repo interfaces.PaymentRepository[dbo.PaymentCard], interval time.Duration) interfaces.SourceGenerator {
	return &cardsGenerator{
		repo:     repo,
		interval: interval,
	}
}

func (g *cardsGenerator) Name() source.Source {
	return source.Cards
}

func (g *cardsGenerator) Interval() time.Duration {
	return g.interval
}

func (g *cardsGenerator) Generate(ctx context.Context) error {
	count := rand.IntN(maxNewPerTick) + 1
	cards := make([]dbo.PaymentCard, 0, count)

	g.mu.Lock()
	availableForUpdate := make([]string, len(g.ids))
	copy(availableForUpdate, g.ids)
	g.mu.Unlock()

	for i := 0; i < count; i++ {
		if len(availableForUpdate) > 0 && rand.Float64() < updateChance {
			// Обновляем статус
			idx := rand.IntN(len(availableForUpdate))
			txID := availableForUpdate[idx]
			availableForUpdate = append(availableForUpdate[:idx], availableForUpdate[idx+1:]...)
			card := dbo.PaymentCard{
				TransactionID: txID,
				PaymentStatus: randomFinalCardStatus(),
				UpdatedAt:     time.Now().UTC(),
			}
			cards = append(cards, card)
		} else {
			// Создаём новую запись
			card := newRandomCard()
			g.mu.Lock()
			g.ids = append(g.ids, card.TransactionID)
			g.mu.Unlock()
			cards = append(cards, card)
		}
	}

	if err := g.repo.Upsert(ctx, cards); err != nil {
		return fmt.Errorf("upsert cards: %w", err)
	}

	return nil
}

func newRandomCard() dbo.PaymentCard {
	return dbo.PaymentCard{
		TransactionID:    uuid.NewString(),
		CardNumberMasked: maskCardNumber(),
		AmountKopecks:    int64(rand.IntN(maxAmountKopecks-minAmountKopecks) + minAmountKopecks),
		CurrencyCode:     cardCurrencies[rand.IntN(len(cardCurrencies))],
		PaymentStatus:    "pending",
		GatewayResponse:  `{"status":"ok","fraud_check":"passed"}`,
	}
}

func maskCardNumber() string {
	last4 := rand.IntN(10000)
	return fmt.Sprintf("**** **** **** %04d", last4)
}

func randomFinalCardStatus() string {
	if rand.Float64() < 0.8 {
		return "succeeded"
	}
	return "cancelled"
}
