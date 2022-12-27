package base

import (
	"time"

	"beanq/helper/stringx"
	"beanq/helper/timex"
	"github.com/spf13/cast"
)

/*
* ParseArgs
*  @Description:

* @param queue
* @param name
* @param payload
* @param retry
* @param maxLen
* @param executeTime
* @return map[string]any
 */
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

/*
* ParseMapToTask
*  @Description:
* @param msg
* @param streamStr
* @return payload
* @return id
* @return stream
* @return addTime
* @return queue
* @return executeTime
* @return retry
* @return maxLen
 */
type BqMessage struct {
	ID     string
	Values map[string]interface{}
}

func ParseMapTask(msg BqMessage, streamStr string) (payload []byte, id, stream, addTime, queue string, executeTime time.Time, retry int, maxLen int64) {

	id = msg.ID
	stream = streamStr

	if queueVal, ok := msg.Values["queue"]; ok {
		if v, ok := queueVal.(string); ok {
			queue = v
		}
	}
	if maxLenV, ok := msg.Values["maxLen"]; ok {
		if v, ok := maxLenV.(string); ok {
			maxLen = cast.ToInt64(v)
		}
	}
	if retryVal, ok := msg.Values["retry"]; ok {
		if v, ok := retryVal.(string); ok {
			retry = cast.ToInt(v)
		}
	}
	if payloadVal, ok := msg.Values["payload"]; ok {
		if payloadV, ok := payloadVal.(string); ok {
			payload = stringx.StringToByte(payloadV)
		}
	}
	if addtimeV, ok := msg.Values["addtime"]; ok {
		if addtimeStr, ok := addtimeV.(string); ok {
			addTime = addtimeStr
		}
	}
	if executeTVal, ok := msg.Values["executeTime"]; ok {
		if executeTm, ok := executeTVal.(string); ok {
			executeTime = cast.ToTime(executeTm)
		}
	}
	return
}
