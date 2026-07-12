package bstatus

import "fmt"

// CodedError keeps API errors machine-readable while preserving the existing Error interface.
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
	t, ok := target.(*CodedError)
	return ok && e != nil && t != nil && e.Code == t.Code
}
