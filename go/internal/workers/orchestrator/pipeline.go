package orchestrator

import (
	"context"
	"log"
	"sync"
	"time"

	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

const (
	maxCheckpointRetries = 3
	retryDelay           = 1 * time.Second
)

type sourceResult struct {
	eventsCh <-chan models.PaymentEvent
	errCh    <-chan error
	name     source.Source
}

func extractAndNormalize[T any](
	ctx context.Context,
	extractor interfaces.Extractor[T],
	normalizer interfaces.Normalizer[T],
	last time.Time,
	processingStartTime time.Time,
	name source.Source,
) sourceResult {
	dataCh, errCh := extractor.Extract(ctx, last, processingStartTime)
	eventsCh := normalizer.Normalize(ctx, dataCh)

	return sourceResult{
		eventsCh: eventsCh,
		errCh:    errCh,
		name:     name,
	}
}

func sourceHasErrors(name source.Source, errCh <-chan error) bool {
	hasErr := false
	for err := range errCh {
		if err != nil {
			log.Printf("Extractor %s error: %v", name, err)
			hasErr = true
		}
	}

	return hasErr
}

func merge(ctx context.Context, eventsCh []<-chan models.PaymentEvent) <-chan models.PaymentEvent {
	var wg sync.WaitGroup
	result := make(chan models.PaymentEvent)

	for i, event := range eventsCh {
		wg.Add(1)
		go func(ch <-chan models.PaymentEvent, idx int) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("merge goroutine for source #%d panicked: %v", idx, r)
				}
			}()

			for {
				select {
				case <-ctx.Done():
					return
				case resultBuff, ok := <-ch:
					if !ok {
						return
					}
					if !sendWithContext(ctx, result, resultBuff) {
						return
					}
				}
			}
		}(event, i)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return result
}

func sendWithContext(ctx context.Context, ch chan<- models.PaymentEvent, value models.PaymentEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- value:
		return true
	}
}
