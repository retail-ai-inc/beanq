package beanq

import (
	"errors"
	"testing"
)

func TestDomainErrorIsByCode(t *testing.T) {
	err := ErrUnsupportedBroker.WithMessage("kafka")
	if !errors.Is(err, ErrUnsupportedBroker) {
		t.Fatalf("expected errors.Is to match by code")
	}
}
