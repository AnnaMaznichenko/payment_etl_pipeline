package loader

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/models"
)

var eventTime = time.Now().UTC().Truncate(time.Second)

func TestLoader_Load(t *testing.T) {
	tests := []struct {
		name    string
		batches [][]models.PaymentEvent
		wantLen int64
	}{
		{
			name: "single batch",
			batches: [][]models.PaymentEvent{
				{
					{
						SourceSystem:      "cards",
						ExternalID:        "card_001",
						UserIdentifier:    "user_123",
						Amount:            1500,
						Currency:          "RUB",
						Status:            "completed",
						EventAt:           eventTime,
						RawStatusOriginal: "SUCCESS",
					},
					{
						SourceSystem:      "cards",
						ExternalID:        "card_002",
						UserIdentifier:    "user_456",
						Amount:            2500,
						Currency:          "RUB",
						Status:            "pending",
						EventAt:           eventTime,
						RawStatusOriginal: "PENDING",
					},
				},
			},
			wantLen: 2,
		},
		{
			name: "multiple batches",
			batches: [][]models.PaymentEvent{
				{
					{
						SourceSystem:      "crypto",
						ExternalID:        "crypto_001",
						UserIdentifier:    "user_789",
						Amount:            50000,
						Currency:          "BTC",
						Status:            "confirmed",
						EventAt:           eventTime,
						RawStatusOriginal: "CONFIRMED",
					},
				},
				{
					{
						SourceSystem:      "wallets",
						ExternalID:        "wallet_001",
						UserIdentifier:    "user_101",
						Amount:            3000,
						Currency:          "RUB",
						Status:            "success",
						EventAt:           eventTime,
						RawStatusOriginal: "OK",
					},
					{
						SourceSystem:      "wallets",
						ExternalID:        "wallet_002",
						UserIdentifier:    "user_102",
						Amount:            4500,
						Currency:          "RUB",
						Status:            "success",
						EventAt:           eventTime,
						RawStatusOriginal: "OK",
					},
				},
				{
					{
						SourceSystem:      "cards",
						ExternalID:        "card_003",
						UserIdentifier:    "user_103",
						Amount:            7500,
						Currency:          "USD",
						Status:            "completed",
						EventAt:           eventTime,
						RawStatusOriginal: "DONE",
					},
				},
			},
			wantLen: 4,
		},
		{
			name: "empty batch skipped",
			batches: [][]models.PaymentEvent{
				{
					{
						SourceSystem:      "cards",
						ExternalID:        "card_004",
						UserIdentifier:    "user_104",
						Amount:            1000,
						Currency:          "RUB",
						Status:            "completed",
						EventAt:           eventTime,
						RawStatusOriginal: "SUCCESS",
					},
				},
				{},
				{
					{
						SourceSystem:      "crypto",
						ExternalID:        "crypto_002",
						UserIdentifier:    "user_105",
						Amount:            25000,
						Currency:          "ETH",
						Status:            "confirmed",
						EventAt:           eventTime,
						RawStatusOriginal: "CONFIRMED",
					},
				},
			},
			wantLen: 2,
		},
		{
			name:    "empty channel",
			batches: [][]models.PaymentEvent{},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			truncatePayments(t, testConn)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			l := NewLoader(testConn)
			ch := make(chan []models.PaymentEvent)

			go func() {
				defer close(ch)
				for _, batch := range tt.batches {
					ch <- batch
				}
			}()

			err := l.Load(ctx, ch)
			require.NoError(t, err)

			var count uint64
			row := testConn.QueryRow(ctx, "SELECT count() FROM "+tableName)
			require.NoError(t, row.Scan(&count))
			assert.Equal(t, tt.wantLen, int64(count))
		})
	}
}

func TestLoader_Load_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	l := NewLoader(testConn)
	ch := make(chan []models.PaymentEvent)

	err := l.Load(ctx, ch)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLoader_Load_DataIntegrity(t *testing.T) {
	truncatePayments(t, testConn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	l := NewLoader(testConn)
	ch := make(chan []models.PaymentEvent)

	batch := []models.PaymentEvent{
		{
			SourceSystem:      "cards",
			ExternalID:        "card_integrity_001",
			UserIdentifier:    "user_integrity",
			Amount:            9999,
			Currency:          "RUB",
			Status:            "verified",
			EventAt:           eventTime,
			RawStatusOriginal: "VERIFIED",
		},
	}

	go func() {
		defer close(ch)
		ch <- batch
	}()

	err := l.Load(ctx, ch)
	require.NoError(t, err)

	rows, err := testConn.Query(ctx, `
		SELECT source_system, external_id, user_identifier, amount, currency, status, event_at, raw_status_original
		FROM default.unified_payments
		WHERE external_id = 'card_integrity_001'
	`)
	require.NoError(t, err)
	defer rows.Close()

	require.True(t, rows.Next(), "expected at least one row")

	var (
		sourceSystem, externalID, userIdentifier string
		amount                                   float64
		currency, status, rawStatusOriginal      string
		eventAt                                  time.Time
	)

	err = rows.Scan(&sourceSystem, &externalID, &userIdentifier, &amount, &currency, &status, &eventAt, &rawStatusOriginal)
	require.NoError(t, err)

	assert.Equal(t, "cards", sourceSystem)
	assert.Equal(t, "card_integrity_001", externalID)
	assert.Equal(t, "user_integrity", userIdentifier)
	assert.Equal(t, float64(9999), amount)
	assert.Equal(t, "RUB", currency)
	assert.Equal(t, "verified", status)
	assert.Equal(t, eventTime.UTC(), eventAt.UTC())
	assert.Equal(t, "VERIFIED", rawStatusOriginal)
}
