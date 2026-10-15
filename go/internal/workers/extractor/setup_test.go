package extractor

import (
	"testing"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/mocks"
)

func newCardTest(t *testing.T, pageSize int) (
	*mocks.PaymentRepository[dbo.PaymentCard],
	interfaces.Extractor[dbo.PaymentCard],
) {
	t.Helper()
	repo := mocks.NewPaymentRepository[dbo.PaymentCard](t)

	return repo, NewCardExtractor(repo, pageSize)
}

func newWalletTest(t *testing.T, pageSize int) (
	*mocks.PaymentRepository[dbo.PaymentWallet],
	interfaces.Extractor[dbo.PaymentWallet],
) {
	t.Helper()
	repo := mocks.NewPaymentRepository[dbo.PaymentWallet](t)

	return repo, NewWalletExtractor(repo, pageSize)
}

func newCryptoTest(t *testing.T, pageSize int) (
	*mocks.PaymentRepository[dbo.PaymentCrypto],
	interfaces.Extractor[dbo.PaymentCrypto],
) {
	t.Helper()
	repo := mocks.NewPaymentRepository[dbo.PaymentCrypto](t)

	return repo, NewCryptoExtractor(repo, pageSize)
}

func drain[T any](dataCh <-chan T, errCh <-chan error) ([]T, []error) {
	var items []T
	for item := range dataCh {
		items = append(items, item)
	}

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return items, errs
}
