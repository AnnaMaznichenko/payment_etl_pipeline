package normalizer

import (
	"context"
	"math"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/models"
	"payment_etl_pipeline/source"
)

type CryptoNormalizer struct {
	statusMap map[string]string
}

func NewCryptoNormalizer() interfaces.Normalizer[dbo.PaymentCrypto] {
	return &CryptoNormalizer{
		statusMap: map[string]string{
			"done": models.StatusSuccess.String(),
			"fail": models.StatusFailed.String(),
		},
	}
}

func (n CryptoNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentCrypto) <-chan models.PaymentEvent {
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
