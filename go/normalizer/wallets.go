package normalizer

import (
	"context"
	"math"
	"strconv"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/models"
	"payment_etl_pipeline/source"
)

type WalletsNormalizer struct {
	statusMap map[int]string
}

func NewWalletsNormalizer() interfaces.Normalizer[dbo.PaymentWallet] {
	return &WalletsNormalizer{
		statusMap: map[int]string{
			0: models.StatusSuccess.String(),
			1: models.StatusFailed.String(),
		},
	}
}

func (n WalletsNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentWallet) <-chan models.PaymentEvent {
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
