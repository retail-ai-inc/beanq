package response

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/json"
	"github.com/spf13/cast"
)

type Result struct {
	Data any    `json:"data"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func (t *Result) reset() {
	t.Data = nil
	t.Msg = SuccessMsg
	t.Code = SuccessCode
}

func (t *Result) Json(w http.ResponseWriter, httpCode int) error {

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(httpCode)

	if err := json.NewEncoder(w).Encode(t); err != nil {
		return err
	}

	return nil
}

func (t *Result) EventMsg(w http.ResponseWriter, eventName string) error {

	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	var builder strings.Builder
	builder.Grow(10)

	builder.WriteString("retry:300\n")

	builder.WriteString("id:")
	builder.WriteString(cast.ToString(time.Now().Unix()))
	builder.WriteString("\n")

	builder.WriteString("event:")
	builder.WriteString(eventName)
	builder.WriteString("\n")

	builder.WriteString("data:")
	builder.Write(b)
	builder.WriteString("\n\n")

	_, err = w.Write([]byte(builder.String()))
	if err != nil {
		return err
	}
	return nil
}

var resultPool = sync.Pool{New: func() any {
	return &Result{
		Code: SuccessCode,
		Msg:  SuccessMsg,
		Data: nil,
	}
}}

func Get() (*Result, func()) {

	result := resultPool.Get().(*Result)
	return result, func() {
		result.reset()
		resultPool.Put(result)
	}
}
