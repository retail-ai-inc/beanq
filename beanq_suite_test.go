package beanq_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBeanq(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Beanq Suite")
}
