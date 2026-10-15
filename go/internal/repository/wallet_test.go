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

func TestWalletRepository_Upsert(t *testing.T) {
	t.Run("insert new wallet", testWalletUpsertInsert)
	t.Run("update existing wallet keeps other fields", testWalletUpsertUpdate)
	t.Run("deduplicates in batch (last-write-wins)", testWalletUpsertDeduplicates)
	t.Run("empty slice is no-op", testWalletUpsertEmpty)
}

func testWalletUpsertInsert(t *testing.T) {
	t.Cleanup(truncateWallets)
	repo := NewWalletRepository(testWalletsDB)

	wallet := dbo.PaymentWallet{
		OperationID:   "op-upsert-insert",
		UserPhone:     "+79991234567",
		AmountRub:     1000.50,
		CommissionRub: 50.25,
		State:         1,
	}

	require.NoError(t, repo.Upsert(context.Background(), []dbo.PaymentWallet{wallet}))

	var got dbo.PaymentWallet
	err := testWalletsDB.Where("operation_id = ?", wallet.OperationID).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, wallet.OperationID, got.OperationID)
	assert.Equal(t, wallet.UserPhone, got.UserPhone)
	assert.Equal(t, wallet.AmountRub, got.AmountRub)
	assert.Equal(t, wallet.CommissionRub, got.CommissionRub)
	assert.Equal(t, wallet.State, got.State)
}

func testWalletUpsertUpdate(t *testing.T) {
	t.Cleanup(truncateWallets)
	ctx := context.Background()
	repo := NewWalletRepository(testWalletsDB)

	original := dbo.PaymentWallet{
		OperationID:   "op-upsert-update",
		UserPhone:     "+79991234567",
		AmountRub:     2000.00,
		CommissionRub: 100.00,
		State:         1,
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentWallet{original}))

	update := dbo.PaymentWallet{
		OperationID: original.OperationID,
		State:       2,
		UpdatedAt:   time.Now().UTC(),
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentWallet{update}))

	var got dbo.PaymentWallet
	err := testWalletsDB.Where("operation_id = ?", original.OperationID).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, update.State, got.State, "state should be updated")
	assert.Equal(t, original.AmountRub, got.AmountRub, "amount_rub should NOT be overwritten")
	assert.Equal(t, original.UserPhone, got.UserPhone, "user_phone should NOT be overwritten")
	assert.Equal(t, original.CommissionRub, got.CommissionRub, "commission_rub should NOT be overwritten")
}

func testWalletUpsertDeduplicates(t *testing.T) {
	t.Cleanup(truncateWallets)
	repo := NewWalletRepository(testWalletsDB)

	operationID := "op-upsert-dup"
	state := 2
	amount := 1500.75
	batch := []dbo.PaymentWallet{
		{OperationID: operationID, AmountRub: 1000.00, State: 1},
		{OperationID: operationID, AmountRub: amount, State: state},
	}
	require.NoError(t, repo.Upsert(context.Background(), batch))

	var count int64
	require.NoError(t,
		testWalletsDB.Model(&dbo.PaymentWallet{}).
			Where("operation_id = ?", operationID).
			Count(&count).Error,
	)
	assert.Equal(t, int64(1), count, "batch should be deduplicated before insert")

	var got dbo.PaymentWallet
	err := testWalletsDB.Where("operation_id = ?", operationID).First(&got).Error
	require.NoError(t, err)

	assert.Equal(t, state, got.State, "should keep last occurrence")
	assert.Equal(t, amount, got.AmountRub, "should keep last occurrence")
}

func testWalletUpsertEmpty(t *testing.T) {
	t.Cleanup(truncateWallets)
	ctx := context.Background()
	repo := NewWalletRepository(testWalletsDB)

	require.NoError(t, repo.Upsert(ctx, nil))
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentWallet{}))
}

func TestWalletRepository_Find(t *testing.T) {
	t.Run("filters by time range", testWalletFindTimeRange)
	t.Run("applies pagination", testWalletFindPagination)
	t.Run("omits commission_rub", testWalletFindOmitCommission)
	t.Run("returns empty for no matches", testWalletFindEmpty)
}

func testWalletFindTimeRange(t *testing.T) {
	t.Cleanup(truncateWallets)
	ctx := context.Background()
	repo := NewWalletRepository(testWalletsDB)

	now := time.Now().UTC().Truncate(time.Millisecond)
	before := now.Add(-time.Hour)
	after := now.Add(time.Hour)

	operationID1 := "op-find-old"
	operationID2 := "op-find-in"
	operationID3 := "op-find-new"
	wallets := []dbo.PaymentWallet{
		{OperationID: operationID1, AmountRub: 100.00, State: 1, UpdatedAt: now.Add(-2 * time.Hour)},
		{OperationID: operationID2, AmountRub: 200.00, State: 1, UpdatedAt: now},
		{OperationID: operationID3, AmountRub: 300.00, State: 1, UpdatedAt: now.Add(2 * time.Hour)},
	}
	require.NoError(t, repo.Upsert(ctx, wallets))

	got, err := repo.Find(ctx, interfaces.Query{
		Condition: interfaces.Condition{From: &before, To: &after},
	})
	require.NoError(t, err)

	ids := extractWalletIDs(got)
	assert.Contains(t, ids, operationID2)
	assert.NotContains(t, ids, operationID1)
	assert.NotContains(t, ids, operationID3)
}

func testWalletFindPagination(t *testing.T) {
	t.Cleanup(truncateWallets)
	ctx := context.Background()
	repo := NewWalletRepository(testWalletsDB)

	operationID0 := "op-page-0"
	operationID1 := "op-page-1"
	operationID2 := "op-page-2"
	operationID3 := "op-page-3"
	now := time.Now().UTC()
	wallets := make([]dbo.PaymentWallet, 5)
	for i := range wallets {
		wallets[i] = dbo.PaymentWallet{
			OperationID: fmt.Sprintf("op-page-%d", i),
			UserPhone:   "+79991234567",
			AmountRub:   100.00,
			State:       1,
			UpdatedAt:   now.Add(time.Duration(i) * time.Second),
		}
	}
	require.NoError(t, repo.Upsert(ctx, wallets))

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

	assert.Equal(t, []string{operationID0, operationID1}, extractWalletIDs(page1))
	assert.Equal(t, []string{operationID2, operationID3}, extractWalletIDs(page2))
}

func testWalletFindOmitCommission(t *testing.T) {
	t.Cleanup(truncateWallets)
	ctx := context.Background()
	repo := NewWalletRepository(testWalletsDB)

	operationID := "op-find-omit"
	wallet := dbo.PaymentWallet{
		OperationID:   operationID,
		UserPhone:     "+79991234567",
		AmountRub:     500.00,
		CommissionRub: 25.50,
		State:         1,
	}
	require.NoError(t, repo.Upsert(ctx, []dbo.PaymentWallet{wallet}))

	got, err := repo.Find(ctx, interfaces.Query{
		Pagination: interfaces.Pagination{Limit: 1},
	})
	require.NoError(t, err)
	require.NotEmpty(t, got)

	var found bool
	for _, w := range got {
		if w.OperationID == operationID {
			found = true
			assert.Empty(t, w.CommissionRub, "commission_rub should be omitted by Find")
		}
	}
	assert.True(t, found, "wallet should be found")
}

func testWalletFindEmpty(t *testing.T) {
	t.Cleanup(truncateWallets)
	repo := NewWalletRepository(testWalletsDB)

	future := time.Now().UTC().Add(100 * time.Hour)
	farFuture := future.Add(time.Hour)

	got, err := repo.Find(context.Background(), interfaces.Query{
		Condition: interfaces.Condition{From: &future, To: &farFuture},
	})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func extractWalletIDs(wallets []dbo.PaymentWallet) []string {
	ids := make([]string, len(wallets))
	for i, w := range wallets {
		ids[i] = w.OperationID
	}

	return ids
}

func truncateWallets() {
	if err := testWalletsDB.Exec("TRUNCATE payment_wallets RESTART IDENTITY CASCADE").Error; err != nil {
		log.Printf("truncate wallets: %v", err)
	}
}
