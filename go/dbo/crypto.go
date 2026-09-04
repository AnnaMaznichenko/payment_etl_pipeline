package dbo

import "time"

type PaymentCrypto struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	TxHash        string    `gorm:"column:tx_hash;uniqueIndex;not null"`
	WalletAddress string    `gorm:"column:wallet_address;not null"`
	AmountBtc     float64   `gorm:"column:amount_btc;not null"`
	Confirmations int       `gorm:"column:confirmations;default:0"`
	StatusCode    string    `gorm:"column:status_code;default:'pend'"`
	BlockTime     time.Time `gorm:"column:block_time;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PaymentCrypto) TableName() string {
	return "payment_crypto"
}
