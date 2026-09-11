package normalizer

import (
	"context"
	"math"
	"strconv"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

type walletsNormalizer struct {
	statusMap map[int]string
}

func NewWalletsNormalizer() interfaces.Normalizer[dbo.PaymentWallet] {
	return &walletsNormalizer{
		statusMap: map[int]string{
			0: models.StatusSuccess.String(),
			1: models.StatusFailed.String(),
		},
	}
}

func (n walletsNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentWallet) <-chan models.PaymentEvent {
	return normalizeEvent(ctx, rawCh, func(wallet dbo.PaymentWallet) models.PaymentEvent {
		return models.PaymentEvent{
			SourceSystem:      source.Wallets.String(),
			ExternalID:        wallet.OperationID,
			UserIdentifier:    wallet.UserPhone,
			Amount:            int64(math.Round(wallet.AmountRub * models.RubToKopecks)),
			Currency:          models.RUB.String(),
			Status:            mapValue(n.statusMap, wallet.State, defaultStatus),
			EventAt:           wallet.ProcessedDt,
			RawStatusOriginal: strconv.Itoa(wallet.State),
		}
	})
}
