package helpers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"
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
	var err error
	for try := range maxRetries {
		err = fn(ctx)
		if err == nil {
			return nil
		}

		log.Printf("Try %d/%d failed: %v", try+1, maxRetries, err)

		if try < maxRetries-1 {
			if delay > 1 {
				delay = delay/2 + time.Duration(rand.Int63n(int64(delay/2)))
			}
			if PauseWithCtx(ctx, delay) {
				return ctx.Err()
			}
			delay *= 2
		}
	}

	return fmt.Errorf("failed after %d tries: %w", maxRetries, err)
}
