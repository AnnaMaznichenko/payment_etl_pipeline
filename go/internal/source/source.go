package source

type Source string

const (
	Cards      Source = "cards"
	Wallets    Source = "wallets"
	Crypto     Source = "crypto"
	Meta       Source = "meta"
	ClickHouse Source = "clickhouse"
)

func (s Source) String() string {
	return string(s)
}

func Slice() []Source {
	return []Source{Cards, Wallets, Crypto}
}
