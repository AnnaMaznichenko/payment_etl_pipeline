package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"payment_etl_pipeline/internal/models"
)

func TestSendWithContext(t *testing.T) {
	t.Run("sends value when ctx is alive", func(t *testing.T) {
		ctx := context.Background()
		ch := make(chan models.PaymentEvent, 1)

		ok := sendWithContext(ctx, ch, models.PaymentEvent{})
		assert.True(t, ok)

		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatal("value was not sent")
		}
	})

	t.Run("returns false when ctx is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		ch := make(chan models.PaymentEvent)
		ok := sendWithContext(ctx, ch, models.PaymentEvent{})
		assert.False(t, ok)
	})
}

func TestMerge(t *testing.T) {
	t.Run("merges events from multiple channels", func(t *testing.T) {
		ctx := context.Background()

		ch1 := make(chan models.PaymentEvent, 3)
		ch2 := make(chan models.PaymentEvent, 2)
		ch3 := make(chan models.PaymentEvent, 1)

		for i := 0; i < 3; i++ {
			ch1 <- models.PaymentEvent{}
		}
		for i := 0; i < 2; i++ {
			ch2 <- models.PaymentEvent{}
		}
		ch3 <- models.PaymentEvent{}

		close(ch1)
		close(ch2)
		close(ch3)

		merged := merge(ctx, []<-chan models.PaymentEvent{ch1, ch2, ch3})

		var count int
		for range merged {
			count++
		}
		assert.Equal(t, 6, count)
	})

	t.Run("closes output when all inputs are closed", func(t *testing.T) {
		ctx := context.Background()

		ch := make(chan models.PaymentEvent)
		close(ch)

		merged := merge(ctx, []<-chan models.PaymentEvent{ch})

		_, ok := <-merged
		assert.False(t, ok, "merged channel should be closed")
	})

	t.Run("stops when ctx is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		ch1 := make(chan models.PaymentEvent)
		ch2 := make(chan models.PaymentEvent)

		merged := merge(ctx, []<-chan models.PaymentEvent{ch1, ch2})

		cancel()

		select {
		case _, ok := <-merged:
			assert.False(t, ok, "merged channel should be closed after ctx cancellation")
		case <-time.After(time.Second):
			t.Fatal("merge did not close output after ctx cancellation")
		}
	})

	t.Run("returns closed channel for empty input", func(t *testing.T) {
		ctx := context.Background()

		merged := merge(ctx, nil)

		select {
		case _, ok := <-merged:
			assert.False(t, ok)
		case <-time.After(time.Second):
			t.Fatal("merge did not close output for empty input")
		}
	})
}
