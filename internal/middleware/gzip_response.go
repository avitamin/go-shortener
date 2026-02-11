package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// GzipResponse возвращает middleware для автоматического сжатия ответов gzip.
// Проверяет заголовок Accept-Encoding и сжимает тело ответа, если клиент поддерживает gzip.
func GzipResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gzWriter := gzip.NewWriter(w)
		defer gzWriter.Close()
		gzResponseWriter := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gzWriter,
		}

		next.ServeHTTP(gzResponseWriter, r)
	})
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer *gzip.Writer
}

// Write записывает данные в gzip Writer для сжатия ответа.
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
