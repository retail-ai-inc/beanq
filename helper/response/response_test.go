package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
)

func TestResultJSONUsesFlatEnvelope(t *testing.T) {
	result := &Result{Code: berror.MissParameterCode, Msg: berror.MissParameterMsg}
	recorder := httptest.NewRecorder()
	if err := result.JSON(recorder, http.StatusBadRequest); err != nil {
		t.Fatalf("encode result: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if payload["code"] != berror.MissParameterCode || payload["msg"] != berror.MissParameterMsg {
		t.Fatalf("unexpected result: %#v", payload)
	}
	if _, exists := payload["error"]; exists {
		t.Fatalf("unexpected nested error: %#v", payload)
	}
}

func TestResultPoolResetsState(t *testing.T) {
	result, release := Get()
	result.Code, result.Msg, result.Data = "error", "failed", "data"
	release()

	result, release = Get()
	defer release()
	if result.Code != berror.SuccessCode || result.Msg != berror.SuccessMsg || result.Data != nil {
		t.Fatalf("pooled result was not reset: %#v", result)
	}
}

func TestResultEventMsgUsesFlatEnvelope(t *testing.T) {
	result := &Result{Code: berror.SuccessCode, Msg: berror.SuccessMsg, Data: "ready"}
	recorder := httptest.NewRecorder()
	if err := result.EventMsg(recorder, "queue"); err != nil {
		t.Fatalf("EventMsg returned error: %v", err)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "event:queue\n") || !strings.Contains(body, `"code":"0000"`) {
		t.Fatalf("unexpected SSE message: %q", body)
	}
}
