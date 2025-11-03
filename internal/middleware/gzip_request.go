package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

func GzipRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzReader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to create gzip reader", http.StatusBadRequest)
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
