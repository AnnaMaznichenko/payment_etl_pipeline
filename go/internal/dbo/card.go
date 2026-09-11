package dbo

import "time"

type PaymentCard struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement"`
	TransactionID    string    `gorm:"column:transaction_id;uniqueIndex;not null"`
	CardNumberMasked string    `gorm:"column:card_number_masked"`
	AmountKopecks    int64     `gorm:"column:amount_kopecks;not null"`
	CurrencyCode     string    `gorm:"column:currency_code;default:'RUB'"`
	PaymentStatus    string    `gorm:"column:payment_status"`
	GatewayResponse  string    `gorm:"column:gateway_response"`
	CreatedAt        time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PaymentCard) TableName() string {
	return "payment_cards"
}
