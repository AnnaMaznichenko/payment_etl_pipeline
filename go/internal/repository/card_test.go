package repository

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
)

func TestCardRepository_Upsert(t *testing.T) {
	t.Run("insert new card", testCardUpsertInsert)
	t.Run("update existing card keeps other fields", testCardUpsertUpdate)
	t.Run("deduplicates in batch (last-write-wins)", testCardUpsertDeduplicates)
	t.Run("empty slice is no-op", testCardUpsertEmpty)
}

func testCardUpsertInsert(t *testing.T) {
	t.Cleanup(truncateCards)
	repo := NewCardRepository(testCardsDB)

	card := dbo.PaymentCard{
		TransactionID:    "tx-upsert-insert",
		CardNumberMasked: "**** **** **** 1111",
		AmountKopecks:    100_000,
		CurrencyCode:     "RUB",
		PaymentStatus:    "pending",
		GatewayResponse:  `{"status":"ok"}`,
	}

	require.NoError(t, repo.Upsert(context.Background(), []dbo.PaymentCard{card}))

	var got dbo.PaymentCard
	err := testCardsDB.Where("transaction_id = ?", card.TransactionID).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, card.TransactionID, got.TransactionID)
	assert.Equal(t, card.CardNumberMasked, got.CardNumberMasked)
	assert.Equal(t, card.AmountKopecks, got.AmountKopecks)
	assert.Equal(t, card.CurrencyCode, got.CurrencyCode)
	assert.Equal(t, card.PaymentStatus, got.PaymentStatus)
}

func testCardUpsertUpdate(t *testing.T) {
	t.Cleanup(truncateCards)
	ctx := context.Background()
	repo := NewCardRepository(testCardsDB)

	original := dbo.PaymentCard{
		TransactionID:    "tx-upsert-update",
		CardNumberMasked: "**** **** **** 2222",
		AmountKopecks:    200_000,
		CurrencyCode:     "RUB",
		PaymentStatus:    "pending",
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCard{original}))

	update := dbo.PaymentCard{
		TransactionID: original.TransactionID,
		PaymentStatus: "succeeded",
		UpdatedAt:     time.Now().UTC(),
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCard{update}))

	var got dbo.PaymentCard
	err := testCardsDB.Where("transaction_id = ?", original.TransactionID).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, update.PaymentStatus, got.PaymentStatus, "status should be updated")
	assert.Equal(t, original.AmountKopecks, got.AmountKopecks, "amount should NOT be overwritten")
	assert.Equal(t, original.CardNumberMasked, got.CardNumberMasked, "card number should NOT be overwritten")
	assert.Equal(t, original.CurrencyCode, got.CurrencyCode, "currency should NOT be overwritten")
}

func testCardUpsertDeduplicates(t *testing.T) {
	t.Cleanup(truncateCards)
	repo := NewCardRepository(testCardsDB)

	transactionID := "tx-upsert-dup"
	batch := []dbo.PaymentCard{
		{TransactionID: transactionID, PaymentStatus: "pending", AmountKopecks: 1},
		{TransactionID: transactionID, PaymentStatus: "succeeded", AmountKopecks: 2},
	}
	require.NoError(t, repo.Upsert(context.Background(), batch))

	var count int64
	require.NoError(t,
		testCardsDB.Model(&dbo.PaymentCard{}).
			Where("transaction_id = ?", transactionID).
			Count(&count).Error,
	)
	assert.Equal(t, int64(1), count, "batch should be deduplicated before insert")

	var got dbo.PaymentCard
	err := testCardsDB.Where("transaction_id = ?", transactionID).First(&got).Error
	require.NoError(t, err)
	assert.Equal(t, "succeeded", got.PaymentStatus, "should keep last occurrence")
	assert.Equal(t, int64(2), got.AmountKopecks, "should keep last occurrence")
}

func testCardUpsertEmpty(t *testing.T) {
	t.Cleanup(truncateCards)
	ctx := context.Background()
	repo := NewCardRepository(testCardsDB)

	require.NoError(t, repo.Upsert(ctx, nil))
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCard{}))
}

func TestCardRepository_Find(t *testing.T) {
	t.Run("filters by time range", testCardFindTimeRange)
	t.Run("applies pagination", testCardFindPagination)
	t.Run("omits gateway_response", testCardFindOmitGatewayResponse)
	t.Run("returns empty for no matches", testCardFindEmpty)
}

func testCardFindTimeRange(t *testing.T) {
	t.Cleanup(truncateCards)
	ctx := context.Background()
	repo := NewCardRepository(testCardsDB)

	now := time.Now().UTC().Truncate(time.Millisecond)
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)

	transactionID1 := "tx-find-old"
	transactionID2 := "tx-find-in"
	transactionID3 := "tx-find-new"
	cards := []dbo.PaymentCard{
		{TransactionID: transactionID1, UpdatedAt: now.Add(-2 * time.Hour)},
		{TransactionID: transactionID2, UpdatedAt: now},
		{TransactionID: transactionID3, UpdatedAt: now.Add(2 * time.Hour)},
	}
	require.NoError(t, repo.Upsert(ctx, cards))

	got, err := repo.Find(ctx, interfaces.Query{
		Condition: interfaces.Condition{From: &before, To: &after},
	})
	require.NoError(t, err)

	ids := extractCardIDs(got)
	assert.Contains(t, ids, transactionID2)
	assert.NotContains(t, ids, transactionID1)
	assert.NotContains(t, ids, transactionID3)
}

func testCardFindPagination(t *testing.T) {
	t.Cleanup(truncateCards)
	ctx := context.Background()
	repo := NewCardRepository(testCardsDB)

	transactionID0 := "tx-find-page-0"
	transactionID1 := "tx-find-page-1"
	transactionID2 := "tx-find-page-2"
	transactionID3 := "tx-find-page-3"
	now := time.Now().UTC()
	cards := make([]dbo.PaymentCard, 5)
	for i := range cards {
		cards[i] = dbo.PaymentCard{
			TransactionID: fmt.Sprintf("tx-find-page-%d", i),
			UpdatedAt:     now.Add(time.Duration(i) * time.Second),
		}
	}
	require.NoError(t, repo.Upsert(ctx, cards))

	from := now.Add(-time.Minute)
	to := now.Add(time.Hour)

	page1, err := repo.Find(ctx, interfaces.Query{
		Condition:  interfaces.Condition{From: &from, To: &to},
		Pagination: interfaces.Pagination{Limit: 2, Offset: 0},
	})
	require.NoError(t, err)
	require.Len(t, page1, 2)

	page2, err := repo.Find(ctx, interfaces.Query{
		Condition:  interfaces.Condition{From: &from, To: &to},
		Pagination: interfaces.Pagination{Limit: 2, Offset: 2},
	})
	require.NoError(t, err)
	require.Len(t, page2, 2)

	assert.Equal(t, []string{transactionID0, transactionID1}, extractCardIDs(page1))
	assert.Equal(t, []string{transactionID2, transactionID3}, extractCardIDs(page2))
}

func testCardFindOmitGatewayResponse(t *testing.T) {
	t.Cleanup(truncateCards)
	ctx := context.Background()
	repo := NewCardRepository(testCardsDB)

	transactionID := "tx-find-omit"
	card := dbo.PaymentCard{
		TransactionID:   transactionID,
		GatewayResponse: `{"secret":"data"}`,
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCard{card}))

	got, err := repo.Find(ctx, interfaces.Query{
		Pagination: interfaces.Pagination{Limit: 1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	var found bool
	for _, c := range got {
		if c.TransactionID == transactionID {
			found = true
			assert.Empty(t, c.GatewayResponse, "gateway_response should be omitted by Find")
		}
	}
	assert.True(t, found, "card should be found")
}

func testCardFindEmpty(t *testing.T) {
	t.Cleanup(truncateCards)
	repo := NewCardRepository(testCardsDB)

	future := time.Now().UTC().Add(100 * time.Hour)
	farFuture := future.Add(time.Hour)

	got, err := repo.Find(context.Background(), interfaces.Query{
		Condition: interfaces.Condition{From: &future, To: &farFuture},
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func extractCardIDs(cards []dbo.PaymentCard) []string {
	ids := make([]string, len(cards))
	for i, c := range cards {
		ids[i] = c.TransactionID
	}

	return ids
}

func truncateCards() {
	if err := testCardsDB.Exec("TRUNCATE payment_cards RESTART IDENTITY CASCADE").Error; err != nil {
		log.Printf("truncate cards: %v", err)
	}
}
