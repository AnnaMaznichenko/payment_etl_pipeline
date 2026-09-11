package normalizer

import (
	"context"

	"payment_etl_pipeline/internal/dbo"
	"payment_etl_pipeline/internal/interfaces"
	"payment_etl_pipeline/internal/models"
	"payment_etl_pipeline/internal/source"
)

type cardsNormalizer struct {
	statusMap map[string]string
}

func NewCardsNormalizer() interfaces.Normalizer[dbo.PaymentCard] {
	return &cardsNormalizer{
		statusMap: map[string]string{
			"succeeded": models.StatusSuccess.String(),
			"cancelled": models.StatusFailed.String(),
		},
	}
}

func (n cardsNormalizer) Normalize(ctx context.Context, rawCh <-chan dbo.PaymentCard) <-chan models.PaymentEvent {
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
