package devicex

import (
	"fmt"
	"testing"
)

func TestInfo(t *testing.T) {
	d := Device
	if err := d.Info(); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v \n", d)
}
