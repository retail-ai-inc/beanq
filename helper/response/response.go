package response

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/json"
	"github.com/spf13/cast"
)

type Result struct {
	Data any    `json:"data"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func (result *Result) reset() {
	result.Data = nil
	result.Msg = berror.SuccessMsg
	result.Code = berror.SuccessCode
}

func (result *Result) JSON(writer http.ResponseWriter, status int) error {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	return json.NewEncoder(writer).Encode(result)
}

func (result *Result) Json(writer http.ResponseWriter, status int) error {
	return result.JSON(writer, status)
}

func (result *Result) EventMsg(writer http.ResponseWriter, eventName string) error {
	payload, err := json.Marshal(result)
	if err != nil {
		return err
	}
	var builder strings.Builder
	builder.Grow(len(payload) + len(eventName) + 48)
	builder.WriteString("retry:300\n")
	builder.WriteString("id:")
	builder.WriteString(cast.ToString(time.Now().UnixNano()))
	builder.WriteString("\nevent:")
	builder.WriteString(eventName)
	builder.WriteString("\ndata:")
	builder.Write(payload)
	builder.WriteString("\n\n")
	_, err = writer.Write([]byte(builder.String()))
	return err
}

var resultPool = sync.Pool{New: func() any {
	return &Result{Code: berror.SuccessCode, Msg: berror.SuccessMsg}
}}

func Get() (*Result, func()) {
	result := resultPool.Get().(*Result)
	return result, func() {
		result.reset()
		resultPool.Put(result)
	}
}
