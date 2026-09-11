package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/helpers"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

type Extractors struct {
	Cards   interfaces.Extractor[dbo.PaymentCard]
	Crypto  interfaces.Extractor[dbo.PaymentCrypto]
	Wallets interfaces.Extractor[dbo.PaymentWallet]
}

type Normalizers struct {
	Cards   interfaces.Normalizer[dbo.PaymentCard]
	Crypto  interfaces.Normalizer[dbo.PaymentCrypto]
	Wallets interfaces.Normalizer[dbo.PaymentWallet]
}

type orchestrator struct {
	extractors     Extractors
	normalizers    Normalizers
	batcher        interfaces.Batcher
	loader         interfaces.Loader
	checkpointRepo interfaces.CheckpointRepository
	pollInterval   time.Duration
}

func NewOrchestrator(
	extractors Extractors,
	normalizers Normalizers,
	batcher interfaces.Batcher,
	loader interfaces.Loader,
	checkpointRepo interfaces.CheckpointRepository,
	pollInterval time.Duration,
) interfaces.Orchestrator {
	return &orchestrator{
		extractors:     extractors,
		normalizers:    normalizers,
		batcher:        batcher,
		loader:         loader,
		checkpointRepo: checkpointRepo,
		pollInterval:   pollInterval,
	}
}

func (o orchestrator) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			processingStartTime := time.Now()

			lastProcessingAt, err := o.checkpointRepo.Get(ctx, source.Slice())
			if err != nil {
				return fmt.Errorf("repo: %w", err)
			}

			results := []sourceResult{
				extractAndNormalize(ctx, o.extractors.Cards, o.normalizers.Cards, lastProcessingAt[source.Cards], processingStartTime, source.Cards),
				extractAndNormalize(ctx, o.extractors.Crypto, o.normalizers.Crypto, lastProcessingAt[source.Crypto], processingStartTime, source.Crypto),
				extractAndNormalize(ctx, o.extractors.Wallets, o.normalizers.Wallets, lastProcessingAt[source.Wallets], processingStartTime, source.Wallets),
			}

			channels := make([]<-chan models.PaymentEvent, 0, len(results))
			for _, r := range results {
				channels = append(channels, r.eventsCh)
			}

			merged := merge(ctx, channels)

			chWithBatch := o.batcher.Batch(ctx, merged)
			err = o.loader.Load(ctx, chWithBatch)
			if err != nil {
				log.Printf("Load error: %v", err)
				if helpers.PauseWithCtx(ctx, o.pollInterval) {
					return nil
				}
				continue
			}

			updates := make(map[source.Source]time.Time)
			for _, r := range results {
				if sourceHasErrors(r.name, r.errCh) {
					log.Printf("Source %s skipped: had extraction errors", r.name)
					continue
				}
				updates[r.name] = processingStartTime
			}

			if len(updates) == 0 {
				log.Println("All sources had errors, no checkpoints to update")
				if helpers.PauseWithCtx(ctx, o.pollInterval) {
					return nil
				}
				continue
			}

			err = helpers.Retry(ctx, func(ctx context.Context) error {
				return o.checkpointRepo.Update(ctx, updates)
			}, maxCheckpointRetries, retryDelay)
			if err != nil {
				log.Printf("update checkpoints: %v", err)
				if helpers.PauseWithCtx(ctx, o.pollInterval) {
					return nil
				}
				continue
			}

			log.Printf("Cycle completed in %v, updated %d source(s)", time.Since(processingStartTime), len(updates))

			if helpers.PauseWithCtx(ctx, o.pollInterval) {
				return nil
			}
		}
	}
}
