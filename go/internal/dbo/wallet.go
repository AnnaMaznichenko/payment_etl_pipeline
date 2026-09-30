package dbo

import "time"

type PaymentWallet struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	OperationID   string    `gorm:"column:operation_id;type:varchar(50);uniqueIndex;not null"`
	UserPhone     string    `gorm:"column:user_phone;type:varchar(15);not null"`
	AmountRub     float64   `gorm:"column:amount_rub;type:decimal(12,2);not null"`
	CommissionRub float64   `gorm:"column:commission_rub;type:decimal(10,2);default:0"`
	State         int       `gorm:"column:state;type:smallint;not null"` // 0,1,2
	ProcessedDt   time.Time `gorm:"column:processed_dt;autoCreateTime"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (PaymentWallet) TableName() string {
	return "payment_wallets"
}
