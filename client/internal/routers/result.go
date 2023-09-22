package routers

import (
	"sync"

	"github.com/retail-ai-inc/client/internal/routers/consts"
)

type Result struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (t *Result) Reset() {
	t.Data = nil
	t.Msg = consts.SuccessMsg
	t.Code = consts.SuccessCode
}

var resultPool = sync.Pool{New: func() any {
	return &Result{
		Code: consts.SuccessCode,
		Msg:  consts.SuccessMsg,
		Data: nil,
	}
}}
