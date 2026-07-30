package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := RequestID(RequestLogger(logger)(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	})))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/me?token=secret", nil)
	request.Header.Set(requestIDHeader, "req_client-123")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v\nlog: %s", err, logs.String())
	}
	if got := entry["msg"]; got != "request completed" {
		t.Errorf("message = %q", got)
	}
	if got := entry["request_id"]; got != "req_client-123" {
		t.Errorf("request_id = %q", got)
	}
	if got := entry["method"]; got != http.MethodPost {
		t.Errorf("method = %q", got)
	}
	if got := entry["path"]; got != "/api/v1/me" {
		t.Errorf("path = %q", got)
	}
	if got := entry["status"]; got != float64(http.StatusCreated) {
		t.Errorf("status = %v", got)
	}
	if _, ok := entry["duration"]; !ok {
		t.Error("duration is missing")
	}
}

func TestRequestLoggerDefaultsToOK(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := RequestLogger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/health/live", nil))

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v", err)
	}
	if got := entry["status"]; got != float64(http.StatusOK) {
		t.Errorf("status = %v, want %d", got, http.StatusOK)
	}
}
