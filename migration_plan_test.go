package beanq

import (
	"reflect"
	"testing"
)

func TestMigrateContextPendingVersions(t *testing.T) {
	ctx := NewMigrateContext(nil)
	got := ctx.PendingVersions(20260108002, []string{
		"migrations/mongo/20260108001_beanq.config.up.json",
		"migrations/mongo/20260108003_beanq.managers.up.json",
		"bad-version.json",
	})
	want := []uint64{20260108003}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pending versions = %v, want %v", got, want)
	}
}
