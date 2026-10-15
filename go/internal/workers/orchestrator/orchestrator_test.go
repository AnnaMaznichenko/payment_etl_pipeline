package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

const testPollInterval = time.Millisecond

func TestOrchestrator_Run(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		beforeRun := time.Now().UTC()
		checkpoints := map[source.Source]time.Time{
			source.Cards:   beforeRun.Add(-3 * time.Hour),
			source.Crypto:  beforeRun.Add(-2 * time.Hour),
			source.Wallets: beforeRun.Add(-1 * time.Hour),
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil).
			Once()

		to.cardExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Cards])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentCard]()).
			Once()
		to.cryptoExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Crypto])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentCrypto]()).
			Once()
		to.walletsExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Wallets])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentWallet]()).
			Once()

		to.cardNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.cryptoNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.walletsNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel()).
			Once()

		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		to.checkpointRepo.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(updates map[source.Source]time.Time) bool {
				_, hasCards := updates[source.Cards]
				_, hasCrypto := updates[source.Crypto]
				_, hasWallets := updates[source.Wallets]
				return len(updates) == 3 && hasCards && hasCrypto && hasWallets
			})).
			RunAndReturn(func(ctx context.Context, updates map[source.Source]time.Time) error {
				// cancel() so that Run exits after the first cycle instead of looping forever
				cancel()
				return nil
			}).
			Once()

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("context cancellation", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("checkpoint get error", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		getErr := errors.New("db connection failed")
		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(nil, getErr).
			Once()

		err := to.orchestrator.Run(ctx)
		require.Error(t, err)
		assert.ErrorIs(t, err, getErr)
	})

	t.Run("loader error continues cycle", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		checkpoints := map[source.Source]time.Time{
			source.Cards:   time.Time{},
			source.Crypto:  time.Time{},
			source.Wallets: time.Time{},
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil)

		setupExtractAndNormalize(t, to)

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel())

		callCount := 0
		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, batches <-chan []models.PaymentEvent) error {
				callCount++
				if callCount == 1 {
					return errors.New("clickhouse unavailable")
				}
				cancel()
				return nil
			})

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, callCount, 2, "orchestrator must continue after loader error")
	})

	t.Run("ctx cancelled after load skips update", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		checkpoints := map[source.Source]time.Time{
			source.Cards:   time.Time{},
			source.Crypto:  time.Time{},
			source.Wallets: time.Time{},
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil).
			Once()

		setupExtractAndNormalize(t, to)

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel()).
			Once()

		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, batches <-chan []models.PaymentEvent) error {
				cancel()
				return nil
			}).
			Once()

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
		to.checkpointRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("partial source failure update only successful", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		checkpoints := map[source.Source]time.Time{
			source.Cards:   time.Time{},
			source.Crypto:  time.Time{},
			source.Wallets: time.Time{},
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil).
			Once()

		to.cardExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentCard, <-chan error) {
				ch, _ := closedChannels[dbo.PaymentCard]()
				return ch, errorChannel(errors.New("cards db down"))
			}).
			Once()
		to.cryptoExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			Return(closedChannels[dbo.PaymentCrypto]()).
			Once()
		to.walletsExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			Return(closedChannels[dbo.PaymentWallet]()).
			Once()

		to.cardNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.cryptoNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.walletsNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel()).
			Once()

		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		// Update is expected only for Crypto and Wallets (Cards failed)
		to.checkpointRepo.EXPECT().
			Update(mock.Anything, mock.MatchedBy(func(updates map[source.Source]time.Time) bool {
				_, hasCards := updates[source.Cards]
				_, hasCrypto := updates[source.Crypto]
				_, hasWallets := updates[source.Wallets]
				return !hasCards && hasCrypto && hasWallets
			})).
			RunAndReturn(func(ctx context.Context, updates map[source.Source]time.Time) error {
				cancel()
				return nil
			}).
			Once()

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("all sources failed no update", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		t.Cleanup(cancel)

		checkpoints := map[source.Source]time.Time{
			source.Cards:   time.Time{},
			source.Crypto:  time.Time{},
			source.Wallets: time.Time{},
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil)

		to.cardExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentCard, <-chan error) {
				ch, _ := closedChannels[dbo.PaymentCard]()
				return ch, errorChannel(errors.New("cards db down"))
			})
		to.cryptoExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentCrypto, <-chan error) {
				ch, _ := closedChannels[dbo.PaymentCrypto]()
				return ch, errorChannel(errors.New("crypto db down"))
			})
		to.walletsExt.EXPECT().
			Extract(mock.Anything, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, from, to time.Time) (<-chan dbo.PaymentWallet, <-chan error) {
				ch, _ := closedChannels[dbo.PaymentWallet]()
				return ch, errorChannel(errors.New("wallets db down"))
			})

		to.cardNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel())
		to.cryptoNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel())
		to.walletsNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel())

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel())

		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			Return(nil)

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
		to.checkpointRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	})

	t.Run("update error continues cycle", func(t *testing.T) {
		to := newTestOrchestrator(t, testPollInterval)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		t.Cleanup(cancel)

		beforeRun := time.Now().UTC()
		checkpoints := map[source.Source]time.Time{
			source.Cards:   beforeRun.Add(-3 * time.Hour),
			source.Crypto:  beforeRun.Add(-2 * time.Hour),
			source.Wallets: beforeRun.Add(-1 * time.Hour),
		}

		to.checkpointRepo.EXPECT().
			Get(mock.Anything, source.Slice()).
			Return(checkpoints, nil)

		to.cardExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Cards])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentCard]()).
			Once()
		to.cryptoExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Crypto])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentCrypto]()).
			Once()
		to.walletsExt.EXPECT().
			Extract(mock.Anything,
				mock.MatchedBy(func(from time.Time) bool {
					return from.Equal(checkpoints[source.Wallets])
				}),
				mock.MatchedBy(func(to time.Time) bool {
					return !to.Before(beforeRun)
				}),
			).
			Return(closedChannels[dbo.PaymentWallet]()).
			Once()

		to.cardNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.cryptoNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()
		to.walletsNorm.EXPECT().
			Normalize(mock.Anything, mock.Anything).
			Return(closedEventChannel()).
			Once()

		to.batcher.EXPECT().
			Batch(mock.Anything, mock.Anything).
			Return(closedBatchChannel()).
			Once()

		to.loader.EXPECT().
			Load(mock.Anything, mock.Anything).
			Return(nil).
			Once()

		updateErr := errors.New("db write failed")
		updateCalls := 0
		to.checkpointRepo.EXPECT().
			Update(mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, _ map[source.Source]time.Time) error {
				updateCalls++
				cancel()
				return updateErr
			})

		err := to.orchestrator.Run(ctx)
		assert.NoError(t, err)
		assert.Equal(t, 1, updateCalls, "Update must be attempted once before ctx cancellation stops retries")
	})
}

// errorChannel creates a channel with one error and closes it
func errorChannel(err error) <-chan error {
	ch := make(chan error, 1)
	ch <- err
	close(ch)
	return ch
}

// closedBatchChannel returns a closed batch channel
func closedBatchChannel() <-chan []models.PaymentEvent {
	ch := make(chan []models.PaymentEvent)
	close(ch)
	return ch
}

// setupExtractAndNormalize configures all 6 Extract+Normalize mocks for tests that need multiple cycles
func setupExtractAndNormalize(t *testing.T, to *testOrchestrator) {
	t.Helper()

	to.cardExt.EXPECT().
		Extract(mock.Anything, mock.Anything, mock.Anything).
		Return(closedChannels[dbo.PaymentCard]())
	to.cryptoExt.EXPECT().
		Extract(mock.Anything, mock.Anything, mock.Anything).
		Return(closedChannels[dbo.PaymentCrypto]())
	to.walletsExt.EXPECT().
		Extract(mock.Anything, mock.Anything, mock.Anything).
		Return(closedChannels[dbo.PaymentWallet]())

	to.cardNorm.EXPECT().
		Normalize(mock.Anything, mock.Anything).
		Return(closedEventChannel())
	to.cryptoNorm.EXPECT().
		Normalize(mock.Anything, mock.Anything).
		Return(closedEventChannel())
	to.walletsNorm.EXPECT().
		Normalize(mock.Anything, mock.Anything).
		Return(closedEventChannel())
}
