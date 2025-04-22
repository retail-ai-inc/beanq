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
	GzWriter *gzip.Writer
}

func (gzw *GzipResponseWriter) Write(data []byte) (int, error) {
	return gzw.Writer.Write(data)
}

func NewGzipResponseWriter(w http.ResponseWriter) *GzipResponseWriter {
	gw, _ := gzip.NewWriterLevel(w, gzip.DefaultCompression)
	return &GzipResponseWriter{
		Writer:         gw,
		ResponseWriter: w,
		GzWriter:       gw,
	}
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
