package extractor

import (
	"context"
	"fmt"
	"time"

	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/source"
)

type paymentExtractor[T any] struct {
	repo     interfaces.PaymentRepository[T]
	pageSize int
	source   source.Source
}

func (e paymentExtractor[T]) Extract(ctx context.Context, from, to time.Time) (<-chan T, <-chan error) {
	dataCh := make(chan T)
	errCh := make(chan error, 1)

	go func() {
		defer close(dataCh)
		defer close(errCh)

		offset := 0
		for {
			select {
			case <-ctx.Done():
				return
			default:
				items, err := e.repo.Find(ctx, interfaces.Query{
					Condition: interfaces.Condition{
						From: &from,
						To:   &to,
					},
					Pagination: interfaces.Pagination{
						Limit:  e.pageSize,
						Offset: offset,
					},
				})
				if err != nil {
					select {
					case <-ctx.Done():
					case errCh <- fmt.Errorf("find %s: %w", e.source, err):
					}

					return
				}

				if len(items) == 0 {
					return
				}

				for _, item := range items {
					select {
					case <-ctx.Done():
						return
					case dataCh <- item:
					}
				}

				offset += e.pageSize
			}
		}
	}()

	return dataCh, errCh
}
