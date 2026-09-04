package normalizer

import (
	"context"

	"payment_etl_pipeline/models"
)

const defaultStatus = string(models.StatusPending)

func normalizeEvent[T any](ctx context.Context, ch <-chan T, convert func(T) models.PaymentEvent) <-chan models.PaymentEvent {
	eventCh := make(chan models.PaymentEvent)

	go func() {
		defer close(eventCh)

		var event models.PaymentEvent

		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-ch:
				if !ok {
					return
				}
				event = convert(value)
			}

			if !sendWithContext(ctx, eventCh, event) {
				return
			}
		}

	}()

	return eventCh
}

func sendWithContext(ctx context.Context, ch chan<- models.PaymentEvent, value models.PaymentEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- value:
		return true
	}
}

func mapValue[K comparable](m map[K]string, key K, defaultVal string) string {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultVal
}
