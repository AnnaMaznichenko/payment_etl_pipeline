package helpers

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPauseWithCtx_Timeout(t *testing.T) {
	start := time.Now()
	cancelled := PauseWithCtx(context.Background(), 50*time.Millisecond)
	elapsed := time.Since(start)

	assert.False(t, cancelled, "should return false when timeout completes")
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond, "should wait at least the specified duration")
}

func TestPauseWithCtx_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	cancelled := PauseWithCtx(ctx, time.Second)
	elapsed := time.Since(start)

	assert.True(t, cancelled, "should return true when context is cancelled")
	assert.Less(t, elapsed, 500*time.Millisecond, "should return immediately after cancellation, not wait for timeout")
}

func TestPauseWithCtx_AlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	cancelled := PauseWithCtx(ctx, time.Second)
	elapsed := time.Since(start)

	assert.True(t, cancelled, "should return true immediately for already cancelled context")
	assert.Less(t, elapsed, 50*time.Millisecond, "should not wait when context is already cancelled")
}

func TestRetry_SuccessOnFirstTry(t *testing.T) {
	var calls int32
	err := Retry(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	}, 3, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "should call fn exactly once on success")
}

func TestRetry_SuccessAfterRetries(t *testing.T) {
	var calls int32
	err := Retry(context.Background(), func(ctx context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}, 5, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "should retry until success")
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	var calls int32
	testErr := errors.New("persistent failure")
	err := Retry(context.Background(), func(ctx context.Context) error {
		atomic.AddInt32(&calls, 1)
		return testErr
	}, 3, time.Millisecond)

	require.Error(t, err)
	assert.ErrorIs(t, err, testErr, "should return the last error")
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls), "should make exactly maxRetries attempts")
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32

	err := Retry(ctx, func(ctx context.Context) error {
		if atomic.AddInt32(&calls, 1) == 2 {
			cancel()
		}
		return errors.New("fail")
	}, 10, time.Millisecond)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled, "should return context.Canceled when ctx is cancelled")
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "should make exactly 2 calls before cancellation")
}

func TestRetry_ReceivesContext(t *testing.T) {
	type ctxKey struct{}
	expectedValue := "test-value"
	ctx := context.WithValue(context.Background(), ctxKey{}, expectedValue)

	var receivedValue any
	err := Retry(ctx, func(ctx context.Context) error {
		receivedValue = ctx.Value(ctxKey{})
		return nil
	}, 1, time.Millisecond)

	require.NoError(t, err)
	assert.Equal(t, expectedValue, receivedValue, "fn should receive the original context with all its values")
}
