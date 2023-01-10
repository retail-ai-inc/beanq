package devicex

import (
	"testing"
)

func TestInfo(t *testing.T) {

	if err := Device.Info(); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%+v \n", Device)
}
