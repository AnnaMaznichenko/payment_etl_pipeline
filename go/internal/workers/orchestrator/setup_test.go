package orchestrator

import (
	"testing"
	"time"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/mocks"
	"payment_etl_pipeline/internal/models"
)

type testOrchestrator struct {
	cardExt    *mocks.Extractor[dbo.PaymentCard]
	cryptoExt  *mocks.Extractor[dbo.PaymentCrypto]
	walletsExt *mocks.Extractor[dbo.PaymentWallet]

	cardNorm    *mocks.Normalizer[dbo.PaymentCard]
	cryptoNorm  *mocks.Normalizer[dbo.PaymentCrypto]
	walletsNorm *mocks.Normalizer[dbo.PaymentWallet]

	batcher        *mocks.Batcher
	loader         *mocks.Loader
	checkpointRepo *mocks.CheckpointRepository

	orchestrator interfaces.Orchestrator
}

func newTestOrchestrator(t *testing.T, pollInterval time.Duration) *testOrchestrator {
	t.Helper()

	cardExt := mocks.NewExtractor[dbo.PaymentCard](t)
	cryptoExt := mocks.NewExtractor[dbo.PaymentCrypto](t)
	walletsExt := mocks.NewExtractor[dbo.PaymentWallet](t)

	cardNorm := mocks.NewNormalizer[dbo.PaymentCard](t)
	cryptoNorm := mocks.NewNormalizer[dbo.PaymentCrypto](t)
	walletsNorm := mocks.NewNormalizer[dbo.PaymentWallet](t)

	batcher := mocks.NewBatcher(t)
	loader := mocks.NewLoader(t)
	checkpointRepo := mocks.NewCheckpointRepository(t)

	o := NewOrchestrator(
		Extractors{
			Cards:   cardExt,
			Crypto:  cryptoExt,
			Wallets: walletsExt,
		},
		Normalizers{
			Cards:   cardNorm,
			Crypto:  cryptoNorm,
			Wallets: walletsNorm,
		},
		batcher,
		loader,
		checkpointRepo,
		pollInterval,
	)

	return &testOrchestrator{
		cardExt:        cardExt,
		cryptoExt:      cryptoExt,
		walletsExt:     walletsExt,
		cardNorm:       cardNorm,
		cryptoNorm:     cryptoNorm,
		walletsNorm:    walletsNorm,
		batcher:        batcher,
		loader:         loader,
		checkpointRepo: checkpointRepo,
		orchestrator:   o,
	}
}

func closedChannels[T any]() (<-chan T, <-chan error) {
	ch := make(chan T)
	errCh := make(chan error)
	close(ch)
	close(errCh)
	return ch, errCh
}

func closedEventChannel() <-chan models.PaymentEvent {
	ch := make(chan models.PaymentEvent)
	close(ch)
	return ch
}
