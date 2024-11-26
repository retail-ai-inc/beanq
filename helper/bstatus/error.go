package bstatus

type Error interface {
	error
	BQError() string
}

type BqError string

func (e BqError) Error() string { return string(e) }

func (BqError) BQError() string {
	return "beanq error"
}

var (
	ErrIdempotent     = BqError("duplicate id")
	BrokerDriverError = BqError("broker driver error, please check")
)
