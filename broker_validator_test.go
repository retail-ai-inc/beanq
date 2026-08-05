package beanq_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	beanq "github.com/retail-ai-inc/beanq/v4"
)

var validatorTestID atomic.Int64

func TestRegisterBrokerConfigValidator(t *testing.T) {
	broker := fmt.Sprintf("external-validator-test-%d", validatorTestID.Add(1))
	configuredBroker := " " + strings.ToUpper(broker) + " "
	sentinel := errors.New("external validator called")
	if err := beanq.RegisterBrokerConfigValidator(configuredBroker, func(config *beanq.BeanqConfig) error {
		if config.Broker != configuredBroker {
			t.Fatalf("validator received unexpected broker %q", config.Broker)
		}
		return sentinel
	}); err != nil {
		t.Fatal(err)
	}

	cfg := &beanq.BeanqConfig{Broker: configuredBroker}
	if err := cfg.Validate(); !errors.Is(err, sentinel) {
		t.Fatalf("expected external validator error, got %v", err)
	}
	if err := beanq.RegisterBrokerConfigValidator(broker, func(*beanq.BeanqConfig) error { return nil }); err == nil ||
		!strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}

func TestBrokerConfigValidatorConcurrentAccess(t *testing.T) {
	broker := fmt.Sprintf("concurrent-validator-test-%d", validatorTestID.Add(1))
	var calls atomic.Int64
	if err := beanq.RegisterBrokerConfigValidator(broker, func(*beanq.BeanqConfig) error {
		calls.Add(1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	const workers = 32
	var wait sync.WaitGroup
	for range workers {
		wait.Go(func() {
			if err := (&beanq.BeanqConfig{Broker: broker}).Validate(); err != nil {
				t.Errorf("validate: %v", err)
			}
		})
	}
	wait.Wait()
	if calls.Load() != workers {
		t.Fatalf("validator calls = %d, want %d", calls.Load(), workers)
	}
}
