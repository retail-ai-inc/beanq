package beanq

import (
	"context"
	"testing"

	"beanq/internal/driver"
)

func TestStart(t *testing.T) {
	ctx := context.Background()
	check := newHealthCheck(driver.NewRdb(optionParameter.RedisOptions))
	err := check.start(ctx)
	if err != nil {
		t.Fatal(err.Error())
	}
}
func TestCtx(t *testing.T) {

}
