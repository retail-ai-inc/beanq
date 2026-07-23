package beanq

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
)

// BrokerConfigValidator validates the broker-specific section of a BeanqConfig.
type BrokerConfigValidator func(*BeanqConfig) error

var registeredBrokerConfigValidators = struct {
	sync.RWMutex
	values map[string]BrokerConfigValidator
}{
	values: map[string]BrokerConfigValidator{
		"redis": func(config *BeanqConfig) error {
			return config.validateRedis()
		},
	},
}

// RegisterBrokerConfigValidator makes a broker-specific configuration validator
// available to BeanqConfig.Validate. Broker names are case-insensitive.
func RegisterBrokerConfigValidator(name string, validator BrokerConfigValidator) error {
	name = normalizeBrokerName(name)
	if name == "" {
		return errors.New("broker validator name is required")
	}
	if validator == nil {
		return errors.New("broker config validator is required")
	}

	registeredBrokerConfigValidators.Lock()
	defer registeredBrokerConfigValidators.Unlock()
	if _, exists := registeredBrokerConfigValidators.values[name]; exists {
		return fmt.Errorf("broker config validator %q is already registered", name)
	}
	registeredBrokerConfigValidators.values[name] = validator
	return nil
}

func validateRegisteredBrokerConfig(config *BeanqConfig) error {
	name := normalizeBrokerName(config.Broker)
	if name == "" {
		return berror.ErrInvalidConfig.WithMessage("broker is required")
	}

	registeredBrokerConfigValidators.RLock()
	validator, exists := registeredBrokerConfigValidators.values[name]
	registeredBrokerConfigValidators.RUnlock()
	if !exists {
		return berror.ErrUnsupportedBroker.WithMessage(config.Broker)
	}
	return validator(config)
}

func normalizeBrokerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
