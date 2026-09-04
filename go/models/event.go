package models

import "time"

type PaymentEvent struct {
	SourceSystem      string    `json:"source_system"`
	ExternalID        string    `json:"external_id"`
	UserIdentifier    string    `json:"user_identifier"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	Status            string    `json:"status"`
	EventAt           time.Time `json:"event_at"`
	RawStatusOriginal string    `json:"raw_status_original"`
}

type ClickHousePaymentEvent struct {
	PaymentEvent
	BatchID uint64 `json:"batch_id"`
}
