package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w gzipResponseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {

			if strings.Contains(r.Header.Get("Content-Type"), "application/json") || strings.Contains(r.Header.Get("Content-Type"), "text/html") {
				reader, err := gzip.NewReader(r.Body)

				if err != nil {
					logger.Errorf("cannot create gzip reader: %w", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				defer reader.Close()

				decodedBody, err := io.ReadAll(reader)
				if err != nil {
					logger.Errorf("cannot decode body: %w", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}

				r.Body = io.NopCloser(bytes.NewReader(decodedBody))

			} else {

				logger.Errorf("invalid Content-Type for gzip encoding")
				http.Error(w, "invalid Content-Type for gzip encoding", http.StatusBadRequest)
				return
			}
		}

		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")

			gzipWriter := gzip.NewWriter(w)
			defer gzipWriter.Close()

			gzipResponseWriter := &gzipResponseWriter{ResponseWriter: w, Writer: gzipWriter}

			next.ServeHTTP(gzipResponseWriter, r)
		} else {
			next.ServeHTTP(w, r)
		}

	})
}
