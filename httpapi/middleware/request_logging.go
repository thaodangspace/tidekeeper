package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLogger records one structured log entry after every HTTP request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			response := &statusResponseWriter{ResponseWriter: writer}
			next.ServeHTTP(response, request)

			logger.Info("request completed",
				slog.String("request_id", RequestIDFromContext(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", response.status()),
				slog.Duration("duration", time.Since(started)),
			)
		})
	}
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (writer *statusResponseWriter) WriteHeader(statusCode int) {
	if writer.statusCode != 0 {
		return
	}
	writer.statusCode = statusCode
	writer.ResponseWriter.WriteHeader(statusCode)
}

func (writer *statusResponseWriter) Write(body []byte) (int, error) {
	if writer.statusCode == 0 {
		writer.WriteHeader(http.StatusOK)
	}
	return writer.ResponseWriter.Write(body)
}

func (writer *statusResponseWriter) status() int {
	if writer.statusCode == 0 {
		return http.StatusOK
	}
	return writer.statusCode
}
