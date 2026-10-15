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

func TestCryptoNormalizer_Normalize_Fields(t *testing.T) {
	crypto := dbo.PaymentCrypto{
		TxHash:        "0xabc123def456",
		WalletAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
		AmountBtc:     0.5,
		StatusCode:    "done",
		BlockTime:     time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC),
	}

	in := make(chan dbo.PaymentCrypto, 1)
	in <- crypto
	close(in)

	n := NewCryptoNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 1)
	expectedAmount := int64(50_000_000) // 0.5 BTC → 50 000 000 satoshi
	assert.Equal(t, source.Crypto.String(), got[0].SourceSystem)
	assert.Equal(t, crypto.TxHash, got[0].ExternalID)
	assert.Equal(t, crypto.WalletAddress, got[0].UserIdentifier)
	assert.Equal(t, expectedAmount, got[0].Amount)
	assert.Equal(t, models.BTC.String(), got[0].Currency)
	assert.Equal(t, models.StatusSuccess.String(), got[0].Status)
	assert.Equal(t, crypto.BlockTime, got[0].EventAt)
	assert.Equal(t, crypto.StatusCode, got[0].RawStatusOriginal)
}

func TestCryptoNormalizer_StatusMapping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"done maps to success", "done", models.StatusSuccess.String()},
		{"fail maps to failed", "fail", models.StatusFailed.String()},
		{"unknown maps to pending", "unknown_status", defaultStatus},
		{"empty maps to pending", "", defaultStatus},
		{"pend maps to pending", "pend", defaultStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := make(chan dbo.PaymentCrypto, 1)
			in <- dbo.PaymentCrypto{
				TxHash:     "0x1",
				StatusCode: tt.input,
			}
			close(in)

			n := NewCryptoNormalizer()
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

func TestCryptoNormalizer_MultipleEvents(t *testing.T) {
	in := make(chan dbo.PaymentCrypto, 3)
	for i := 0; i < 3; i++ {
		in <- dbo.PaymentCrypto{
			TxHash:     fmt.Sprintf("0x%d", i),
			StatusCode: "done",
		}
	}
	close(in)

	n := NewCryptoNormalizer()
	var got []models.PaymentEvent
	for ev := range n.Normalize(context.Background(), in) {
		got = append(got, ev)
	}

	require.Len(t, got, 3)
	assert.Equal(t, "0x0", got[0].ExternalID)
	assert.Equal(t, "0x1", got[1].ExternalID)
	assert.Equal(t, "0x2", got[2].ExternalID)
}

func TestCryptoNormalizer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	n := NewCryptoNormalizer()
	in := make(chan dbo.PaymentCrypto)
	out := n.Normalize(ctx, in)
	cancel()

	select {
	case _, ok := <-out:
		assert.False(t, ok, "channel should be closed after cancel")
	case <-time.After(time.Second):
		t.Fatal("timeout: channel did not close after context cancellation")
	}
}

func TestCryptoNormalizer_EmptyChannel(t *testing.T) {
	in := make(chan dbo.PaymentCrypto)
	close(in)

	var count int
	n := NewCryptoNormalizer()
	for range n.Normalize(context.Background(), in) {
		count++
	}

	assert.Equal(t, 0, count)
}
