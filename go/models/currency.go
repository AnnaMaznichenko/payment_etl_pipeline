package models

type Currency string

const (
	RUB Currency = "RUB"
	USD Currency = "USD"
	BTC Currency = "BTC"
)

const (
	RubToKopecks = 100         // 1 RUB = 100 копеек
	BtcToSatoshi = 100_000_000 // 1 BTC = 100 000 000 сатоши
)

func (c Currency) String() string {
	return string(c)
}
