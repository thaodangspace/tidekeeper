// Package middleware provides defensive HTTP middleware shared by all routes.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"sync/atomic"
)

const requestIDHeader = "X-Request-ID"

var (
	validRequestID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	fallbackID     atomic.Uint64
)

type requestIDContextKey struct{}

// RequestID validates an incoming request ID or creates a cryptographically random one.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get(requestIDHeader)
		if !validRequestID.MatchString(requestID) {
			requestID = newRequestID()
		}
		writer.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(request.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// RequestIDFromContext returns the ID assigned by RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "req_fallback_" + strconv.FormatUint(fallbackID.Add(1), 36)
	}
	return "req_" + hex.EncodeToString(bytes)
}
