package beanq

type Error interface {
	error
	BQError() string
}

type bqError string

func (e bqError) Error() string { return string(e) }

func (bqError) BQError() string {
	return "beanq error"
}
