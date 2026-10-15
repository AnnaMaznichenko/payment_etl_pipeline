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

func TestCryptoExtractor_Extract(t *testing.T) {
	t.Run("single page", testCryptoExtractorSinglePage)
	t.Run("multiple pages", testCryptoExtractorMultiplePages)
	t.Run("empty result", testCryptoExtractorEmptyResult)
	t.Run("repository error", testCryptoExtractorRepositoryError)
	t.Run("context cancellation", testCryptoExtractorContextCancellation)
}

func testCryptoExtractorSinglePage(t *testing.T) {
	repo, extractor := newCryptoTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	crypto := []dbo.PaymentCrypto{
		{TxHash: "0xabc123", WalletAddress: "1A2B3C", AmountBtc: 0.5},
		{TxHash: "0xdef456", WalletAddress: "4D5E6F", AmountBtc: 1.0},
	}

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return(crypto, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 10},
		}).
		Return([]dbo.PaymentCrypto{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 2)
	assert.Equal(t, crypto, got)
	assert.Empty(t, errs)
}

func testCryptoExtractorMultiplePages(t *testing.T) {
	repo, extractor := newCryptoTest(t, 2)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	page1 := []dbo.PaymentCrypto{
		{TxHash: "0xabc123", WalletAddress: "1A2B3C", AmountBtc: 0.5},
		{TxHash: "0xdef456", WalletAddress: "4D5E6F", AmountBtc: 1.0},
	}
	page2 := []dbo.PaymentCrypto{
		{TxHash: "0xghi789", WalletAddress: "7G8H9I", AmountBtc: 2.5},
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
		Return([]dbo.PaymentCrypto{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 3)
	assert.Equal(t, page1[0].TxHash, got[0].TxHash)
	assert.Equal(t, page1[1].TxHash, got[1].TxHash)
	assert.Equal(t, page2[0].TxHash, got[2].TxHash)
	assert.Empty(t, errs)
}

func testCryptoExtractorEmptyResult(t *testing.T) {
	repo, extractor := newCryptoTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return([]dbo.PaymentCrypto{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	assert.Empty(t, got)
	assert.Empty(t, errs)
}

func testCryptoExtractorRepositoryError(t *testing.T) {
	repo, extractor := newCryptoTest(t, 10)

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
	assert.Contains(t, errs[0].Error(), "find crypto")
	assert.Contains(t, errs[0].Error(), "database error")
}

func testCryptoExtractorContextCancellation(t *testing.T) {
	repo, extractor := newCryptoTest(t, 10)

	ctx, cancel := context.WithCancel(context.Background())
	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Return([]dbo.PaymentCrypto{{TxHash: "0xabc123", WalletAddress: "1A2B3C", AmountBtc: 0.5}}, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ interfaces.Query) { cancel() }).
		Return([]dbo.PaymentCrypto{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(ctx, from, to)
	got, errs := drain(dataCh, errCh)

	assert.NotEmpty(t, got)
	assert.Empty(t, errs, "should not emit errors on context cancellation")
	assert.Error(t, ctx.Err(), "ctx should be cancelled after test")
}
