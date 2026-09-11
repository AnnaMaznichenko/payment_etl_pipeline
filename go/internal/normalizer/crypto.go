package normalizer

import (
	"context"
	"math"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

type cryptoNormalizer struct {
	statusMap map[string]string
}

func NewCryptoNormalizer() interfaces.Normalizer[dbo.PaymentCrypto] {
	return &cryptoNormalizer{
		statusMap: map[string]string{
			"done": models.StatusSuccess.String(),
			"fail": models.StatusFailed.String(),
		},
	}
}

func (n cryptoNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentCrypto) <-chan models.PaymentEvent {
	return normalizeEvent(ctx, rawCh, func(crypto dbo.PaymentCrypto) models.PaymentEvent {
		return models.PaymentEvent{
			SourceSystem:      source.Crypto.String(),
			ExternalID:        crypto.TxHash,
			UserIdentifier:    crypto.WalletAddress,
			Amount:            int64(math.Round(crypto.AmountBtc * models.BtcToSatoshi)),
			Currency:          models.BTC.String(),
			Status:            mapValue(n.statusMap, crypto.StatusCode, defaultStatus),
			EventAt:           crypto.BlockTime,
			RawStatusOriginal: crypto.StatusCode,
		}
	})
}
