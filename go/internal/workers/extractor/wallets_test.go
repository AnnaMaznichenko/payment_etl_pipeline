package extractor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
)

func TestWalletExtractor_Extract(t *testing.T) {
	t.Run("single page", testWalletExtractorSinglePage)
	t.Run("multiple pages", testWalletExtractorMultiplePages)
	t.Run("empty result", testWalletExtractorEmptyResult)
	t.Run("repository error", testWalletExtractorRepositoryError)
	t.Run("context cancellation", testWalletExtractorContextCancellation)
}

func testWalletExtractorSinglePage(t *testing.T) {
	repo, extractor := newWalletTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	wallets := []dbo.PaymentWallet{
		{OperationID: "op-1", UserPhone: "+79991234567", AmountRub: 1000.50},
		{OperationID: "op-2", UserPhone: "+79997654321", AmountRub: 2000.75},
	}

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return(wallets, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 10},
		}).
		Return([]dbo.PaymentWallet{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 2)
	assert.Equal(t, wallets, got)
	assert.Empty(t, errs)
}

func testWalletExtractorMultiplePages(t *testing.T) {
	repo, extractor := newWalletTest(t, 2)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	page1 := []dbo.PaymentWallet{
		{OperationID: "op-1", UserPhone: "+79991234567", AmountRub: 1000.50},
		{OperationID: "op-2", UserPhone: "+79997654321", AmountRub: 2000.75},
	}
	page2 := []dbo.PaymentWallet{
		{OperationID: "op-3", UserPhone: "+79991112233", AmountRub: 3000.00},
	}

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 2, Offset: 0},
		}).
		Return(page1, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 2, Offset: 2},
		}).
		Return(page2, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 2, Offset: 4},
		}).
		Return([]dbo.PaymentWallet{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 3)
	assert.Equal(t, page1[0].OperationID, got[0].OperationID)
	assert.Equal(t, page1[1].OperationID, got[1].OperationID)
	assert.Equal(t, page2[0].OperationID, got[2].OperationID)
	assert.Empty(t, errs)
}

func testWalletExtractorEmptyResult(t *testing.T) {
	repo, extractor := newWalletTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return([]dbo.PaymentWallet{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	assert.Empty(t, got)
	assert.Empty(t, errs)
}

func testWalletExtractorRepositoryError(t *testing.T) {
	repo, extractor := newWalletTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return(nil, errors.New("database error")).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	assert.Empty(t, got)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "find wallets")
	assert.Contains(t, errs[0].Error(), "database error")
}

func testWalletExtractorContextCancellation(t *testing.T) {
	repo, extractor := newWalletTest(t, 10)

	ctx, cancel := context.WithCancel(context.Background())
	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Return([]dbo.PaymentWallet{{OperationID: "op-1", UserPhone: "+79991234567", AmountRub: 1000.50}}, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ interfaces.Query) { cancel() }).
		Return([]dbo.PaymentWallet{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(ctx, from, to)
	got, errs := drain(dataCh, errCh)

	assert.NotEmpty(t, got)
	assert.Empty(t, errs, "should not emit errors on context cancellation")
	assert.Error(t, ctx.Err(), "ctx should be cancelled after test")
}
