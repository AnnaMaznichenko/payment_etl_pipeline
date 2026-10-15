package batcher

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"payment_etl_pipeline/internal/models"
)

func TestBatcher_FullSize(t *testing.T) {
	ctx := context.Background()
	ids := []string{"A", "B", "C"}
	events := make(chan models.PaymentEvent, len(ids))

	for _, id := range ids {
		events <- models.PaymentEvent{ExternalID: id}
	}
	close(events)

	b := NewBatcher(3, time.Second)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	require.Len(t, batches, 1, "should send exactly one batch")
	assert.Len(t, batches[0], 3, "batch should have 3 events")
	assert.Equal(t, "A", batches[0][0].ExternalID)
	assert.Equal(t, "B", batches[0][1].ExternalID)
	assert.Equal(t, "C", batches[0][2].ExternalID)
}

func TestBatcher_PartialBatchOnClose(t *testing.T) {
	ctx := context.Background()
	events := make(chan models.PaymentEvent, 2)

	events <- models.PaymentEvent{ExternalID: "A"}
	events <- models.PaymentEvent{ExternalID: "B"}
	close(events)

	b := NewBatcher(5, time.Second)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	require.Len(t, batches, 1, "should send one batch on channel close")
	assert.Len(t, batches[0], 2, "batch should have 2 events")
	assert.Equal(t, "A", batches[0][0].ExternalID)
	assert.Equal(t, "B", batches[0][1].ExternalID)
}

func TestBatcher_MultipleBatches(t *testing.T) {
	ctx := context.Background()
	ids := []string{"A", "B", "C", "D", "E", "F", "G"}
	events := make(chan models.PaymentEvent, len(ids))

	// Expected: batch1=[A,B,C], batch2=[D,E,F], batch3=[G]
	for _, id := range ids {
		events <- models.PaymentEvent{ExternalID: id}
	}
	close(events)

	b := NewBatcher(3, 50*time.Millisecond)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	require.Len(t, batches, 3, "should send 3 batches")

	// First batch:  full
	assert.Len(t, batches[0], 3)
	assert.Equal(t, []string{"A", "B", "C"}, []string{
		batches[0][0].ExternalID, batches[0][1].ExternalID, batches[0][2].ExternalID,
	})

	// Second batch: full
	assert.Len(t, batches[1], 3)
	assert.Equal(t, []string{"D", "E", "F"}, []string{
		batches[1][0].ExternalID, batches[1][1].ExternalID, batches[1][2].ExternalID,
	})

	// Third batch: incomplete (flushed by timeout)
	assert.Len(t, batches[2], 1)
	assert.Equal(t, "G", batches[2][0].ExternalID)
}

func TestBatcher_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan models.PaymentEvent, 2)

	events <- models.PaymentEvent{ExternalID: "A"}
	events <- models.PaymentEvent{ExternalID: "B"}

	// Cancel the context before closing the channel
	cancel()
	close(events)

	b := NewBatcher(5, time.Second)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	// On context cancellation the batch is not sent (graceful shutdown)
	assert.Empty(t, batches, "should not send batches after context cancellation")
}

func TestBatcher_EmptyBatchOnTimeout(t *testing.T) {
	ctx := context.Background()
	events := make(chan models.PaymentEvent)
	close(events)

	b := NewBatcher(5, 50*time.Millisecond)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	assert.Empty(t, batches, "should not send empty batch")
}

func TestBatcher_BatchCopy(t *testing.T) {
	ctx := context.Background()
	ids := []string{"A", "B", "C", "D", "E", "F"}
	events := make(chan models.PaymentEvent, len(ids))

	for _, id := range ids {
		events <- models.PaymentEvent{ExternalID: id}
	}
	close(events)

	b := NewBatcher(3, 50*time.Millisecond)
	out := b.Batch(ctx, events)

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	require.Len(t, batches, 2)

	// The first batch must remain [A,B,C] even after the second is sent
	assert.Equal(t, "A", batches[0][0].ExternalID)
	assert.Equal(t, "B", batches[0][1].ExternalID)
	assert.Equal(t, "C", batches[0][2].ExternalID)

	// The second batch must be [D,E,F]
	assert.Equal(t, "D", batches[1][0].ExternalID)
	assert.Equal(t, "E", batches[1][1].ExternalID)
	assert.Equal(t, "F", batches[1][2].ExternalID)
}

func TestBatcher_SendBatchContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := make(chan models.PaymentEvent, 3)

	events <- models.PaymentEvent{ExternalID: "A"}
	events <- models.PaymentEvent{ExternalID: "B"}
	events <- models.PaymentEvent{ExternalID: "C"}

	b := NewBatcher(3, time.Second)
	out := b.Batch(ctx, events)

	time.Sleep(20 * time.Millisecond)
	cancel()

	var batches [][]models.PaymentEvent
	for batch := range out {
		batches = append(batches, batch)
	}

	// On context cancellation no batch is sent
	assert.Empty(t, batches, "should not send batch when context is cancelled during send")
}

func TestBatcher_TimerFiresBeforeChannelClose(t *testing.T) {
	ctx := context.Background()
	events := make(chan models.PaymentEvent, 10)

	events <- models.PaymentEvent{ExternalID: "A"}
	events <- models.PaymentEvent{ExternalID: "B"}

	b := NewBatcher(5, 100*time.Millisecond)
	out := b.Batch(ctx, events)

	select {
	case batch := <-out:
		require.Len(t, batch, 2, "should send partial batch on timer")
		assert.Equal(t, "A", batch[0].ExternalID)
		assert.Equal(t, "B", batch[1].ExternalID)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for batch")
	}

	close(events)

	// Drain the remaining batches
	for batch := range out {
		_ = batch
	}
}
