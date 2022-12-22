package base

import (
	"time"

	"beanq/helper/timex"
)

func ParseArgs(queue, name, payload string, retry int, maxLen int64, executeTime time.Time) map[string]any {
	values := make(map[string]any)
	values["queue"] = queue
	values["name"] = name
	values["payload"] = payload
	values["addtime"] = time.Now().Format(timex.DateTime)
	values["retry"] = retry
	values["maxLen"] = maxLen

	if !executeTime.IsZero() {
		values["executeTime"] = executeTime
	}
	return values
}
