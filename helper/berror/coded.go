package berror

import "fmt"

type CodedError struct {
	Code    string
	Message string
	Cause   error
}

func NewCodedError(code, message string, cause error) *CodedError {
	return &CodedError{Code: code, Message: message, Cause: cause}
}

func (e *CodedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *CodedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *CodedError) BQError() string {
	if e == nil {
		return "beanq error"
	}
	return e.Code
}

func (e *CodedError) Is(target error) bool {
	typed, ok := target.(*CodedError)
	return ok && e != nil && typed != nil && e.Code == typed.Code
}
