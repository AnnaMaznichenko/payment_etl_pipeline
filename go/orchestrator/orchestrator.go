package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/models"
	"payment_etl_pipeline/source"
)

type orchestrator struct {
	cardExtractor    interfaces.Extractor[dbo.PaymentCard]
	cryptoExtractor  interfaces.Extractor[dbo.PaymentCrypto]
	walletExtractor  interfaces.Extractor[dbo.PaymentWallet]
	cardNormalizer   interfaces.Normalizer[dbo.PaymentCard]
	cryptoNormalizer interfaces.Normalizer[dbo.PaymentCrypto]
	walletNormalizer interfaces.Normalizer[dbo.PaymentWallet]
	batcher          interfaces.Batcher
	loader           interfaces.Loader
	repo             interfaces.CheckpointRepository
	pullInterval     time.Duration
}

func NewOrchestrator(
	cardExtractor interfaces.Extractor[dbo.PaymentCard],
	cryptoExtractor interfaces.Extractor[dbo.PaymentCrypto],
	walletExtractor interfaces.Extractor[dbo.PaymentWallet],
	cardNormalizer interfaces.Normalizer[dbo.PaymentCard],
	cryptoNormalizer interfaces.Normalizer[dbo.PaymentCrypto],
	walletNormalizer interfaces.Normalizer[dbo.PaymentWallet],
	batcher interfaces.Batcher,
	loader interfaces.Loader,
	repo interfaces.CheckpointRepository,
	pullInterval time.Duration,
) interfaces.Orchestrator {
	return &orchestrator{
		cardExtractor:    cardExtractor,
		cryptoExtractor:  cryptoExtractor,
		walletExtractor:  walletExtractor,
		cardNormalizer:   cardNormalizer,
		cryptoNormalizer: cryptoNormalizer,
		walletNormalizer: walletNormalizer,
		batcher:          batcher,
		loader:           loader,
		repo:             repo,
		pullInterval:     pullInterval,
	}
}

func (o orchestrator) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			processingStartTime := time.Now()

			lastProcessingAt, err := o.repo.Get(ctx, source.Slice())
			if err != nil {
				return fmt.Errorf("repo: %w", err)
			}

			cardsRawCh, err := extract(ctx, o.cardExtractor, lastProcessingAt[source.Cards], processingStartTime, source.Cards)
			if err != nil {
				return err
			}

			cryptoRawCh, err := extract(ctx, o.cryptoExtractor, lastProcessingAt[source.Crypto], processingStartTime, source.Crypto)
			if err != nil {
				return err
			}

			walletsRawCh, err := extract(ctx, o.walletExtractor, lastProcessingAt[source.Wallets], processingStartTime, source.Wallets)
			if err != nil {
				return err
			}

			merged := merge(ctx, []<-chan models.PaymentEvent{
				o.cardNormalizer.Normalize(ctx, cardsRawCh),
				o.cryptoNormalizer.Normalize(ctx, cryptoRawCh),
				o.walletNormalizer.Normalize(ctx, walletsRawCh),
			})

			chWithBatch := o.batcher.Batch(ctx, merged)
			err = o.loader.Load(ctx, chWithBatch)
			if err != nil {
				return fmt.Errorf("load error: %w", err)
			}

			err = o.repo.Update(ctx, map[source.Source]time.Time{
				source.Cards:   processingStartTime,
				source.Crypto:  processingStartTime,
				source.Wallets: processingStartTime,
			})
			if err != nil {
				return fmt.Errorf("update checkpoints: %w", err)
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(o.pullInterval):
			}
		}
	}
}

func extract[T any](
	ctx context.Context,
	extractor interfaces.Extractor[T],
	last time.Time,
	processingStartTime time.Time,
	name source.Source,
) (<-chan T, error) {
	ch, err := extractor.Extract(ctx, last, processingStartTime)
	if err != nil {
		return nil, fmt.Errorf("%s extract: %w", name, err)
	}

	return ch, nil
}

func merge(ctx context.Context, eventsCh []<-chan models.PaymentEvent) <-chan models.PaymentEvent {
	var wg sync.WaitGroup
	result := make(chan models.PaymentEvent)

	for _, event := range eventsCh {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case resultBuff, ok := <-event:
					if !ok {
						return
					}
					if !sendWithContext(ctx, result, resultBuff) {
						return
					}
				}
			}
		}()
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
