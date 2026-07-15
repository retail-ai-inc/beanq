package berror

import "fmt"

type ErrorCode string

const (
	CodeUnsupportedBroker ErrorCode = "broker.unsupported"
	CodeInvalidConfig     ErrorCode = "config.invalid"
)

type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e DomainError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e DomainError) Unwrap() error {
	return e.Cause
}

func (e DomainError) Is(target error) bool {
	switch target := target.(type) {
	case DomainError:
		return e.Code == target.Code
	case *DomainError:
		return target != nil && e.Code == target.Code
	default:
		return false
	}
}

func (e DomainError) WithCause(cause error) DomainError {
	e.Cause = cause
	return e
}

func (e DomainError) WithMessage(detail string) DomainError {
	if detail != "" {
		e.Message = fmt.Sprintf("%s: %s", e.Message, detail)
	}
	return e
}

var (
	ErrUnsupportedBroker = DomainError{Code: CodeUnsupportedBroker, Message: "unsupported broker"}
	ErrInvalidConfig     = DomainError{Code: CodeInvalidConfig, Message: "invalid config"}
)
