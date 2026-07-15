package routers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
)

const (
	maxRequestBody = 1 << 20
	maxPageSize    = 100
)

type pageParams struct {
	Page     int64
	PageSize int64
}

type pageMeta struct {
	Page       int64 `json:"page"`
	PageSize   int64 `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int64 `json:"totalPages"`
}

func parsePage(r *http.Request) (pageParams, error) {
	page, err := positiveQueryInt(r, "page", 1)
	if err != nil {
		return pageParams{}, err
	}
	pageSize, err := positiveQueryInt(r, "pageSize", 10)
	if err != nil {
		return pageParams{}, err
	}
	if pageSize > maxPageSize {
		return pageParams{}, fmt.Errorf("pageSize must not exceed %d", maxPageSize)
	}
	return pageParams{Page: page, PageSize: pageSize}, nil
}

func positiveQueryInt(r *http.Request, name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}

func newPageMeta(page, pageSize, total int64) pageMeta {
	pages := int64(0)
	if total > 0 {
		pages = int64(math.Ceil(float64(total) / float64(pageSize)))
	}
	return pageMeta{Page: page, PageSize: pageSize, Total: total, TotalPages: pages}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	result, release := response.Get()
	defer release()
	result.Code = code
	result.Msg = message
	_ = result.Json(w, status)
}

func writeBadRequest(w http.ResponseWriter, err error) {
	writeAPIError(w, http.StatusBadRequest, berror.MissParameterCode, err.Error())
}

func streamEvents(w http.ResponseWriter, r *http.Request, interval time.Duration, emit func() error) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("streaming unsupported")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := emit(); err != nil {
			return err
		}
		flusher.Flush()
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
		}
	}
}

func jsonForm(handler HandleFunc, pathIDField string) HandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := make(url.Values)
		if pathIDField != "" {
			values.Set(pathIDField, r.PathValue("id"))
		}
		if r.Body != nil && r.ContentLength != 0 {
			var input map[string]any
			if err := decodeJSON(w, r, &input); err != nil {
				writeBadRequest(w, err)
				return
			}
			for key, value := range input {
				switch typed := value.(type) {
				case string:
					values.Set(key, typed)
				default:
					encoded, err := json.Marshal(typed)
					if err != nil {
						writeBadRequest(w, fmt.Errorf("invalid %s", key))
						return
					}
					values.Set(key, string(encoded))
				}
			}
		}
		r.Form, r.PostForm = values, values
		handler(w, r)
	}
}

func pathQuery(handler HandleFunc, name string) HandleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		query.Set(name, r.PathValue("id"))
		r.URL.RawQuery = query.Encode()
		handler(w, r)
	}
}
