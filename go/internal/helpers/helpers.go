package helpers

import (
	"context"
	"time"

	"github.com/avast/retry-go/v4"
)

func PauseWithCtx(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return true
	case <-time.After(d):
		return false
	}
}

func Retry(ctx context.Context, fn func(ctx context.Context) error, maxRetries int, delay time.Duration) error {
	return retry.Do(
		func() error { return fn(ctx) },
		retry.Attempts(uint(maxRetries)),
		retry.Delay(delay),
		retry.DelayType(retry.FullJitterBackoffDelay),
		retry.Context(ctx),
		retry.LastErrorOnly(true),
	)
}
