package bredis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v4/helper/tool"
	public "github.com/retail-ai-inc/beanq/v4/internal"
	"github.com/retail-ai-inc/beanq/v4/internal/capture"
	"github.com/spf13/cast"
)

// executeMessage synchronously invokes a callback and records its execution metadata.
// The bool result is false when the parent context was canceled; callers must not
// advance or acknowledge that message in that case.
func executeMessage(ctx context.Context, job public.Stream, handler public.CallbackWithRetry, config *capture.Config) (public.Stream, bool) {
	val := job.Data
	// Copy the map for the handler to prevent it from racing with metadata updates.
	// The values intentionally remain shallow-copied to preserve existing behavior.
	copiedVal := make(map[string]any, len(val))
	for k, v := range val {
		copiedVal[k] = v
	}

	retrys := cast.ToInt(val["retry"])
	now := time.Now()
	val["status"] = bstatus.StatusReceived
	val["beginTime"] = now

	var timeToRunLimit []time.Duration
	if raw, ok := val["timeToRunLimit"]; ok {
		if encoded, ok := raw.(string); ok {
			if err := json.Unmarshal([]byte(encoded), &timeToRunLimit); err != nil {
				capture.Fail.When(config).If(&capture.Channel{Channel: job.Channel, Topic: []string{job.Stream}}).Then(err)
			}
		} else {
			err := fmt.Errorf("timeToRunLimit must be a JSON string, got %T", raw)
			capture.Fail.When(config).If(&capture.Channel{Channel: job.Channel, Topic: []string{job.Stream}}).Then(err)
		}
	}

	timeToRun := cast.ToDuration(val["timeToRun"])
	sessionCtx, cancel := context.WithTimeout(ctx, timeToRun)
	defer cancel()

	retry, err := tool.RetryInfo(sessionCtx, func() (handlerErr error) {
		defer func() {
			if p := recover(); p != nil {
				handlerErr = fmt.Errorf("[panic recover]: %+v\n%s", p, debug.Stack())
			}
		}()

		if len(timeToRunLimit) > 0 {
			go captureRunLimit(sessionCtx, now, timeToRunLimit, copiedVal, config)
		}

		_, handlerErr = handler.Handle(sessionCtx, copiedVal, cast.ToInt(val["retry"]))
		return handlerErr
	}, retrys)

	// Parent cancellation means ownership is being relinquished. Do not invoke the
	// error callback or return a result which could be logged and acknowledged.
	if ctx.Err() != nil {
		return job, false
	}

	if err != nil {
		handler.Error(sessionCtx, err)
		val["level"] = bstatus.ErrLevel
		val["info"] = err.Error()
		val["status"] = bstatus.StatusFailed
	} else {
		val["status"] = bstatus.StatusSuccess
	}

	val["endTime"] = time.Now()
	val["retry"] = retry
	val["runTime"] = cast.ToTime(val["endTime"]).Sub(cast.ToTime(val["beginTime"])).Seconds()
	hostname, _ := os.Hostname()
	val["hostName"] = hostname
	job.Data = val
	return job, true
}

func captureRunLimit(ctx context.Context, begin time.Time, limits []time.Duration, data map[string]any, config *capture.Config) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for i := 0; i < len(limits); {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Since(begin) >= limits[i] {
				i++
				capErr := fmt.Errorf("Info:Task execution timeout,Body:%+v", data)
				capture.System.When(config).If(nil).Then(capErr)
			}
		}
	}
}
