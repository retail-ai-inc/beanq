package routers

import "sync"

type Result struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (t *Result) Reset() {
	t.Data = nil
	t.Msg = "success"
	t.Code = "0000"
}

var resultPool = sync.Pool{New: func() any {
	return &Result{
		Code: "0000",
		Msg:  "success",
		Data: nil,
	}
}}
