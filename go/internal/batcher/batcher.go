package batcher

import (
	"context"
	"time"

	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
)

type batcher struct {
	size    int
	timeout time.Duration
}

func NewBatcher(size int, timeout time.Duration) interfaces.Batcher {
	return &batcher{
		size:    size,
		timeout: timeout,
	}
}

func (b batcher) Batch(ctx context.Context, events <-chan models.PaymentEvent) <-chan []models.PaymentEvent {
	batch := make([]models.PaymentEvent, 0, b.size)
	out := make(chan []models.PaymentEvent)

	go func() {
		defer close(out)

		timer := time.NewTimer(b.timeout)
		sendAndClearBatch := func() {
			if len(batch) == 0 {
				resetTimer(timer, b.timeout)
				return
			}

			sendBatch(ctx, out, batch, timer, b.timeout)
			batch = batch[:0]
		}

		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				sendAndClearBatch()
			case event, ok := <-events:
				if !ok {
					sendAndClearBatch()
					return
				}

				if len(batch) == b.size {
					sendAndClearBatch()
				}

				batch = append(batch, event)
			}
		}
	}()

	return out
}

func sendBatch(ctx context.Context, out chan []models.PaymentEvent, batch []models.PaymentEvent, timer *time.Timer, timeout time.Duration) {
	batchCopy := make([]models.PaymentEvent, len(batch))
	copy(batchCopy, batch)

	select {
	case <-ctx.Done():
		return
	case out <- batchCopy:
	}

	resetTimer(timer, timeout)
}

func resetTimer(t *time.Timer, d time.Duration) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}
