package timex

import (
	"fmt"
	"testing"
	"time"
)

func TestHalfHour(t *testing.T) {

	now, _ := time.Parse(DateTime, "2023-04-24 04:30:15")
	now2 := HalfHour(now)
	fmt.Println(now2.Format(DateTime))
}
