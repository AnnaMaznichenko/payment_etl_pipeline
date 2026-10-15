package normalizer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

func TestCardsNormalizer_Normalize_Fields(t *testing.T) {
	card := dbo.PaymentCard{
		TransactionID:    "tx-123",
		CardNumberMasked: "**** **** **** 1234",
		AmountKopecks:    150_000,
		CurrencyCode:     "RUB",
		PaymentStatus:    "succeeded",
		CreatedAt:        time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}

	in := make(chan dbo.PaymentCard, 1)
	in <- card
	close(in)

	n := NewCardsNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 1)
	assert.Equal(t, source.Cards.String(), got[0].SourceSystem)
	assert.Equal(t, card.TransactionID, got[0].ExternalID)
	assert.Equal(t, card.CardNumberMasked, got[0].UserIdentifier)
	assert.Equal(t, card.AmountKopecks, got[0].Amount)
	assert.Equal(t, card.CurrencyCode, got[0].Currency)
	assert.Equal(t, models.StatusSuccess.String(), got[0].Status)
	assert.Equal(t, card.CreatedAt, got[0].EventAt)
	assert.Equal(t, card.PaymentStatus, got[0].RawStatusOriginal)
}

func TestCardsNormalizer_StatusMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"succeeded maps to success", "succeeded", models.StatusSuccess.String()},
		{"cancelled maps to failed", "cancelled", models.StatusFailed.String()},
		{"unknown maps to pending", "unknown_status", defaultStatus},
		{"empty maps to pending", "", defaultStatus},
		{"pending maps to pending", "pending", defaultStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan dbo.PaymentCard, 1)
			in <- dbo.PaymentCard{
				TransactionID: "tx-1",
				PaymentStatus: tt.input,
			}
			close(in)

			n := NewCardsNormalizer()
			var got []models.PaymentEvent
			for ev := range n.Normalize(context.Background(), in) {
				got = append(got, ev)
			}

			require.Len(t, got, 1)
			assert.Equal(t, tt.expected, got[0].Status)
			assert.Equal(t, tt.input, got[0].RawStatusOriginal)
		})
	}
}

func TestCardsNormalizer_MultipleEvents(t *testing.T) {
	in := make(chan dbo.PaymentCard, 3)
	for i := 0; i < 3; i++ {
		in <- dbo.PaymentCard{
			TransactionID: fmt.Sprintf("tx-%d", i),
			PaymentStatus: "succeeded",
		}
	}
	close(in)

	n := NewCardsNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 3)
	assert.Equal(t, "tx-0", got[0].ExternalID)
	assert.Equal(t, "tx-1", got[1].ExternalID)
	assert.Equal(t, "tx-2", got[2].ExternalID)
}

func TestCardsNormalizer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewCardsNormalizer()
	in := make(chan dbo.PaymentCard)
	out := n.Normalize(ctx, in)
	cancel()

	select {
	case _, ok := <-out:
		assert.False(t, ok, "channel should be closed after cancel")
	case <-time.After(time.Second):
		t.Fatal("timeout: channel did not close after context cancellation")
	}
}

func TestCardsNormalizer_EmptyChannel(t *testing.T) {
	in := make(chan dbo.PaymentCard)
	close(in)

	var count int
	n := NewCardsNormalizer()
	for range n.Normalize(context.Background(), in) {
		count++
	}

	assert.Equal(t, 0, count)
}
