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

func TestCardExtractor_Extract(t *testing.T) {
	t.Run("single page", testCardExtractorSinglePage)
	t.Run("multiple pages", testCardExtractorMultiplePages)
	t.Run("empty result", testCardExtractorEmptyResult)
	t.Run("repository error", testCardExtractorRepositoryError)
	t.Run("context cancellation", testCardExtractorContextCancellation)
}

func testCardExtractorSinglePage(t *testing.T) {
	repo, extractor := newCardTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	cards := []dbo.PaymentCard{
		{TransactionID: "tx-1", AmountKopecks: 1000},
		{TransactionID: "tx-2", AmountKopecks: 2000},
	}

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return(cards, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 10},
		}).
		Return([]dbo.PaymentCard{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 2)
	assert.Equal(t, cards, got)
	assert.Empty(t, errs)
}

func testCardExtractorMultiplePages(t *testing.T) {
	repo, extractor := newCardTest(t, 2)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	page1 := []dbo.PaymentCard{
		{TransactionID: "tx-1", AmountKopecks: 1000},
		{TransactionID: "tx-2", AmountKopecks: 2000},
	}
	page2 := []dbo.PaymentCard{
		{TransactionID: "tx-3", AmountKopecks: 3000},
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
		Return([]dbo.PaymentCard{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	require.Len(t, got, 3)
	assert.Equal(t, page1[0].TransactionID, got[0].TransactionID)
	assert.Equal(t, page1[1].TransactionID, got[1].TransactionID)
	assert.Equal(t, page2[0].TransactionID, got[2].TransactionID)
	assert.Empty(t, errs)
}

func testCardExtractorEmptyResult(t *testing.T) {
	repo, extractor := newCardTest(t, 10)

	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, interfaces.Query{
			Condition:  interfaces.Condition{From: &from, To: &to},
			Pagination: interfaces.Pagination{Limit: 10, Offset: 0},
		}).
		Return([]dbo.PaymentCard{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(context.Background(), from, to)
	got, errs := drain(dataCh, errCh)

	assert.Empty(t, got)
	assert.Empty(t, errs)
}

func testCardExtractorRepositoryError(t *testing.T) {
	repo, extractor := newCardTest(t, 10)

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
	assert.Contains(t, errs[0].Error(), "find cards")
	assert.Contains(t, errs[0].Error(), "database error")
}

func testCardExtractorContextCancellation(t *testing.T) {
	repo, extractor := newCardTest(t, 10)

	ctx, cancel := context.WithCancel(context.Background())
	from := time.Now().Add(-time.Hour)
	to := time.Now()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Return([]dbo.PaymentCard{{TransactionID: "tx-1", AmountKopecks: 1000}}, nil).
		Once()

	repo.EXPECT().
		Find(mock.Anything, mock.Anything).
		Run(func(_ context.Context, _ interfaces.Query) { cancel() }).
		Return([]dbo.PaymentCard{}, nil).
		Once()

	dataCh, errCh := extractor.Extract(ctx, from, to)
	got, errs := drain(dataCh, errCh)

	assert.NotEmpty(t, got)
	assert.Empty(t, errs, "should not emit errors on context cancellation")
	assert.Error(t, ctx.Err(), "ctx should be cancelled after test")
}
