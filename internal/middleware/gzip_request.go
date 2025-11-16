package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func GzipRequest(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gzReader, err := gzip.NewReader(r.Body)
			if err != nil {
				logger.Error(err.Error())
				http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
				return
			}
			defer gzReader.Close()

			r.Body = struct {
				io.Reader
				io.Closer
			}{
				Reader: gzReader,
				Closer: r.Body,
			}

			next.ServeHTTP(w, r)

		})
	}
}
