package normalizer

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

func TestWalletsNormalizer_Normalize_Fields(t *testing.T) {
	wallet := dbo.PaymentWallet{
		OperationID: "op-456",
		UserPhone:   "+79001234567",
		AmountRub:   100.50,
		State:       0,
		ProcessedDt: time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}

	in := make(chan dbo.PaymentWallet, 1)
	in <- wallet
	close(in)

	n := NewWalletsNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 1)
	expectedAmount := int64(10050) // 100.50 RUB → 10050 копеек
	expectedRawStatus := "0"
	assert.Equal(t, source.Wallets.String(), got[0].SourceSystem)
	assert.Equal(t, wallet.OperationID, got[0].ExternalID)
	assert.Equal(t, wallet.UserPhone, got[0].UserIdentifier)
	assert.Equal(t, expectedAmount, got[0].Amount)
	assert.Equal(t, models.RUB.String(), got[0].Currency)
	assert.Equal(t, models.StatusSuccess.String(), got[0].Status)
	assert.Equal(t, wallet.ProcessedDt, got[0].EventAt)
	assert.Equal(t, expectedRawStatus, got[0].RawStatusOriginal)
}

func TestWalletsNormalizer_StatusMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"state 0 maps to success", 0, models.StatusSuccess.String()},
		{"state 1 maps to failed", 1, models.StatusFailed.String()},
		{"state -1 maps to pending", -1, defaultStatus},
		{"state 2 maps to pending", 2, defaultStatus},
		{"state 99 maps to pending", 99, defaultStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan dbo.PaymentWallet, 1)
			in <- dbo.PaymentWallet{
				OperationID: "op-1",
				State:       tt.input,
			}
			close(in)

			n := NewWalletsNormalizer()
			var got []models.PaymentEvent
			for ev := range n.Normalize(context.Background(), in) {
				got = append(got, ev)
			}

			require.Len(t, got, 1)
			assert.Equal(t, tt.expected, got[0].Status)
			assert.Equal(t, strconv.Itoa(tt.input), got[0].RawStatusOriginal)
		})
	}
}

func TestWalletsNormalizer_MultipleEvents(t *testing.T) {
	in := make(chan dbo.PaymentWallet, 3)
	for i := 0; i < 3; i++ {
		in <- dbo.PaymentWallet{
			OperationID: fmt.Sprintf("op-%d", i),
			State:       0,
		}
	}
	close(in)

	n := NewWalletsNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 3)
	assert.Equal(t, "op-0", got[0].ExternalID)
	assert.Equal(t, "op-1", got[1].ExternalID)
	assert.Equal(t, "op-2", got[2].ExternalID)
}

func TestWalletsNormalizer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewWalletsNormalizer()
	in := make(chan dbo.PaymentWallet)
	out := n.Normalize(ctx, in)
	cancel()

	select {
	case _, ok := <-out:
		assert.False(t, ok, "channel should be closed after cancel")
	case <-time.After(time.Second):
		t.Fatal("timeout: channel did not close after context cancellation")
	}
}

func TestWalletsNormalizer_EmptyChannel(t *testing.T) {
	in := make(chan dbo.PaymentWallet)
	close(in)

	var count int
	n := NewWalletsNormalizer()
	for range n.Normalize(context.Background(), in) {
		count++
	}

	assert.Equal(t, 0, count)
}
