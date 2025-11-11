package logger

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func InitLogger() error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("cannot initialize zap logger: %w", err)
	}
	log = logger.Sugar()
	return nil
}

func Sync() {
	log.Sync()
}

func Infof(template string, args ...interface{}) {
	log.Infof(template, args...)
}

func Errorf(template string, args ...interface{}) {
	log.Errorf(template, args)
}

func Fatalf(template string, args ...interface{}) {
	log.Fatalf(template, args)
}

type (
	responseData struct {
		status int
		size   int
	}
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: http.StatusOK,
		}

		lw := &loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}

		next.ServeHTTP(lw, r)

		duration := time.Since(start)

		log.Infow("request",
			"uri", r.RequestURI,
			"method", r.Method,
			"duration", duration,
		)

		log.Infow("response",
			"status", responseData.status,
			"size", responseData.size)
	})
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.responseData.size += n
	return n, err
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.responseData.status = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggingResponseWriter) Header() http.Header {
	return lrw.ResponseWriter.Header()
}
