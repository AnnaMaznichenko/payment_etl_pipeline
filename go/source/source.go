package source

type Source string

const (
	Cards   Source = "cards"
	Wallets Source = "wallets"
	Crypto  Source = "crypto"
)

func (s Source) String() string {
	return string(s)
}

func Slice() []Source {
	return []Source{Cards, Wallets, Crypto}
}
