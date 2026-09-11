package loader

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"payment_etl_pipeline/internal/db"
	"payment_etl_pipeline/internal/helpers"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
)

const (
	maxRetries = 3
	retryDelay = 1 * time.Second
)

type loader struct {
	conn clickhouse.Conn
}

func NewLoader(db *db.ClickHouseDB) interfaces.Loader {
	return &loader{
		conn: db.Conn,
	}
}

func (l loader) Load(ctx context.Context, batches <-chan []models.PaymentEvent) error {
	var totalBatches, totalEvents int64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case batch, ok := <-batches:
			if !ok {
				return nil
			}

			if len(batch) == 0 {
				continue
			}

			events := prepareEvents(batch)
			err := helpers.Retry(ctx, func(ctx context.Context) error {
				return insertBatch(ctx, l.conn, events)
			}, maxRetries, retryDelay)
			if err != nil {
				return err
			}

			totalBatches++
			totalEvents += int64(len(batch))
			log.Printf("[Loader] Batch #%d: %d events (total: %d)", totalBatches, len(batch), totalEvents)
		}
	}
}

func prepareEvents(batch []models.PaymentEvent) []models.ClickHousePaymentEvent {
	batchID := uuid.New()
	events := make([]models.ClickHousePaymentEvent, len(batch))
	for i, event := range batch {
		events[i] = models.ClickHousePaymentEvent{
			PaymentEvent: event,
			BatchID:      batchID,
		}
	}

	return events
}

func insertBatch(ctx context.Context, conn clickhouse.Conn, events []models.ClickHousePaymentEvent) error {
	batch, err := conn.PrepareBatch(ctx, `INSERT INTO unified_payments (source_system, external_id, user_identifier, amount, currency, status, event_at, raw_status_original, batch_id)`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	sent := false
	defer func() {
		if !sent {
			_ = batch.Abort()
		}
	}()

	for _, ev := range events {
		err := batch.Append(
			ev.SourceSystem,
			ev.ExternalID,
			ev.UserIdentifier,
			ev.Amount,
			ev.Currency,
			ev.Status,
			ev.EventAt,
			ev.RawStatusOriginal,
			ev.BatchID.String(),
		)
		if err != nil {
			return fmt.Errorf("append row: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("send batch: %w", err)
	}

	sent = true

	return nil
}
