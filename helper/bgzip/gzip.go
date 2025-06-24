//go:generate fzgen -o ../../test/fuzz/gzipfuzz_test.go
package bgzip

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (gzw *GzipResponseWriter) Write(data []byte) (int, error) {
	return gzw.Writer.Write(data)
}

func (gzw *GzipResponseWriter) Close() error {
	if gw, ok := gzw.Writer.(*gzip.Writer); ok {
		return gw.Close()
	}
	return nil
}

func NewGzipResponseWriter(w http.ResponseWriter) (*GzipResponseWriter, error) {
	gw, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
	if err != nil {
		return nil, err
	}
	return &GzipResponseWriter{
		Writer:         gw,
		ResponseWriter: w,
	}, nil
}

func MatchGzipEncoding(r *http.Request) bool {
	encodings := r.Header.Get("Accept-Encoding")
	parts := strings.Split(encodings, ",")
	for _, part := range parts {
		if strings.TrimSpace(part) == "gzip" {
			return true
		}
	}
	return false
}
