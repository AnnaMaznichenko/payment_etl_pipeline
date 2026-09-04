package normalizer

import (
	"context"

	"payment_etl_pipeline/dbo"
	"payment_etl_pipeline/interfaces"
	"payment_etl_pipeline/models"
	"payment_etl_pipeline/source"
)

type CardsNormalizer struct {
	statusMap map[string]string
}

func NewCardsNormalizer() interfaces.Normalizer[dbo.PaymentCard] {
	return &CardsNormalizer{
		statusMap: map[string]string{
			"succeeded": models.StatusSuccess.String(),
			"cancelled": models.StatusFailed.String(),
		},
	}
}

func (n CardsNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentCard) <-chan models.PaymentEvent {
	return normalizeEvent(ctx, rawCh, func(card dbo.PaymentCard) models.PaymentEvent {
		return models.PaymentEvent{
			SourceSystem:      source.Cards.String(),
			ExternalID:        card.TransactionID,
			UserIdentifier:    card.CardNumberMasked,
			Amount:            card.AmountKopecks,
			Currency:          card.CurrencyCode,
			Status:            mapValue(n.statusMap, card.PaymentStatus, defaultStatus),
			EventAt:           card.CreatedAt,
			RawStatusOriginal: card.PaymentStatus,
		}
	})
}
