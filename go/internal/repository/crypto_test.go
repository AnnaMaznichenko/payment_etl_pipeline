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

func TestCryptoRepository_Upsert(t *testing.T) {
	t.Run("insert new crypto", testCryptoUpsertInsert)
	t.Run("update existing crypto keeps other fields", testCryptoUpsertUpdate)
	t.Run("deduplicates in batch (last-write-wins)", testCryptoUpsertDeduplicates)
	t.Run("empty slice is no-op", testCryptoUpsertEmpty)
}

func testCryptoUpsertInsert(t *testing.T) {
	t.Cleanup(truncateCrypto)
	repo := NewCryptoRepository(testCryptoDB)

	crypto := dbo.PaymentCrypto{
		TxHash:        "0xabc123insert",
		WalletAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		AmountBtc:     0.5,
		Confirmations: 3,
		StatusCode:    "pend",
	}

	require.NoError(t, repo.Upsert(context.Background(), []dbo.PaymentCrypto{crypto}))

	var got dbo.PaymentCrypto
	err := testCryptoDB.Where("tx_hash = ?", crypto.TxHash).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, crypto.TxHash, got.TxHash)
	assert.Equal(t, crypto.WalletAddress, got.WalletAddress)
	assert.Equal(t, crypto.AmountBtc, got.AmountBtc)
	assert.Equal(t, crypto.Confirmations, got.Confirmations)
	assert.Equal(t, crypto.StatusCode, got.StatusCode)
}

func testCryptoUpsertUpdate(t *testing.T) {
	t.Cleanup(truncateCrypto)
	ctx := context.Background()
	repo := NewCryptoRepository(testCryptoDB)

	original := dbo.PaymentCrypto{
		TxHash:        "0xabc123update",
		WalletAddress: "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
		AmountBtc:     1.0,
		Confirmations: 1,
		StatusCode:    "pend",
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCrypto{original}))

	update := dbo.PaymentCrypto{
		TxHash:     original.TxHash,
		StatusCode: "conf",
		UpdatedAt:  time.Now().UTC(),
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCrypto{update}))

	var got dbo.PaymentCrypto
	err := testCryptoDB.Where("tx_hash = ?", original.TxHash).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, update.StatusCode, got.StatusCode, "status_code should be updated")
	assert.Equal(t, original.AmountBtc, got.AmountBtc, "amount_btc should NOT be overwritten")
	assert.Equal(t, original.WalletAddress, got.WalletAddress, "wallet_address should NOT be overwritten")
	assert.Equal(t, original.Confirmations, got.Confirmations, "confirmations should NOT be overwritten")
}

func testCryptoUpsertDeduplicates(t *testing.T) {
	t.Cleanup(truncateCrypto)
	repo := NewCryptoRepository(testCryptoDB)

	txHash := "0xabc123dup"
	status := "conf"
	amount := 0.2
	batch := []dbo.PaymentCrypto{
		{TxHash: txHash, StatusCode: "pend", AmountBtc: 0.1},
		{TxHash: txHash, StatusCode: status, AmountBtc: amount},
	}
	require.NoError(t, repo.Upsert(context.Background(), batch))

	var count int64
	require.NoError(t,
		testCryptoDB.Model(&dbo.PaymentCrypto{}).
			Where("tx_hash = ?", txHash).
			Count(&count).Error,
	)
	assert.Equal(t, int64(1), count, "batch should be deduplicated before insert")

	var got dbo.PaymentCrypto
	err := testCryptoDB.Where("tx_hash = ?", txHash).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, status, got.StatusCode, "should keep last occurrence")
	assert.Equal(t, amount, got.AmountBtc, "should keep last occurrence")
}

func testCryptoUpsertEmpty(t *testing.T) {
	t.Cleanup(truncateCrypto)
	ctx := context.Background()
	repo := NewCryptoRepository(testCryptoDB)

	require.NoError(t, repo.Upsert(ctx, nil))
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCrypto{}))
}

func TestCryptoRepository_Find(t *testing.T) {
	t.Run("filters by time range", testCryptoFindTimeRange)
	t.Run("applies pagination", testCryptoFindPagination)
	t.Run("omits confirmations", testCryptoFindOmitConfirmations)
	t.Run("returns empty for no matches", testCryptoFindEmpty)
}

func testCryptoFindTimeRange(t *testing.T) {
	t.Cleanup(truncateCrypto)
	ctx := context.Background()
	repo := NewCryptoRepository(testCryptoDB)

	now := time.Now().UTC().Truncate(time.Millisecond)
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)

	txHash1 := "0xfindold"
	txHash2 := "0xfindin"
	txHash3 := "0xfindnew"
	cryptos := []dbo.PaymentCrypto{
		{TxHash: txHash1, WalletAddress: "addr1", AmountBtc: 0.1, UpdatedAt: now.Add(-2 * time.Hour)},
		{TxHash: txHash2, WalletAddress: "addr2", AmountBtc: 0.2, UpdatedAt: now},
		{TxHash: txHash3, WalletAddress: "addr3", AmountBtc: 0.3, UpdatedAt: now.Add(2 * time.Hour)},
	}
	require.NoError(t, repo.Upsert(ctx, cryptos))

	got, err := repo.Find(ctx, interfaces.Query{
		Condition: interfaces.Condition{From: &before, To: &after},
	})
	require.NoError(t, err)

	ids := extractCryptoIDs(got)
	assert.Contains(t, ids, txHash2)
	assert.NotContains(t, ids, txHash1)
	assert.NotContains(t, ids, txHash3)
}

func testCryptoFindPagination(t *testing.T) {
	t.Cleanup(truncateCrypto)
	ctx := context.Background()
	repo := NewCryptoRepository(testCryptoDB)

	txHash0 := "0xpage0"
	txHash1 := "0xpage1"
	txHash2 := "0xpage2"
	txHash3 := "0xpage3"
	now := time.Now().UTC()
	cryptos := make([]dbo.PaymentCrypto, 5)
	for i := range cryptos {
		cryptos[i] = dbo.PaymentCrypto{
			TxHash:        fmt.Sprintf("0xpage%d", i),
			WalletAddress: "addr",
			AmountBtc:     0.1,
			UpdatedAt:     now.Add(time.Duration(i) * time.Second),
		}
	}
	require.NoError(t, repo.Upsert(ctx, cryptos))

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

	assert.Equal(t, []string{txHash0, txHash1}, extractCryptoIDs(page1))
	assert.Equal(t, []string{txHash2, txHash3}, extractCryptoIDs(page2))
}

func testCryptoFindOmitConfirmations(t *testing.T) {
	t.Cleanup(truncateCrypto)
	ctx := context.Background()
	repo := NewCryptoRepository(testCryptoDB)

	txHash := "0xfindomit"
	crypto := dbo.PaymentCrypto{
		TxHash:        txHash,
		WalletAddress: "addr",
		AmountBtc:     0.5,
		Confirmations: 6,
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentCrypto{crypto}))

	got, err := repo.Find(ctx, interfaces.Query{
		Pagination: interfaces.Pagination{Limit: 1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	var found bool
	for _, c := range got {
		if c.TxHash == txHash {
			found = true
			assert.Empty(t, c.Confirmations, "confirmations should be omitted by Find")
		}
	}
	assert.True(t, found, "crypto should be found")
}

func testCryptoFindEmpty(t *testing.T) {
	t.Cleanup(truncateCrypto)
	repo := NewCryptoRepository(testCryptoDB)

	future := time.Now().UTC().Add(100 * time.Hour)
	farFuture := future.Add(time.Hour)

	got, err := repo.Find(context.Background(), interfaces.Query{
		Condition: interfaces.Condition{From: &future, To: &farFuture},
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func extractCryptoIDs(cryptos []dbo.PaymentCrypto) []string {
	ids := make([]string, len(cryptos))
	for i, c := range cryptos {
		ids[i] = c.TxHash
	}

	return ids
}

func truncateCrypto() {
	if err := testCryptoDB.Exec("TRUNCATE payment_crypto RESTART IDENTITY CASCADE").Error; err != nil {
		log.Printf("truncate crypto: %v", err)
	}
}
