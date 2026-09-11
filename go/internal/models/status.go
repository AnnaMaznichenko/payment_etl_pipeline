package models

type NormalizedStatus string

const (
	StatusSuccess NormalizedStatus = "success"
	StatusFailed  NormalizedStatus = "failed"
	StatusPending NormalizedStatus = "pending"
)

func (s NormalizedStatus) String() string {
	return string(s)
}
