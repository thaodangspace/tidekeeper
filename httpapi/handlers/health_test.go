package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type pingerStub struct{ err error }

func (p pingerStub) Ping(context.Context) error { return p.err }

func TestHealthLive(t *testing.T) {
	health := NewHealth(pingerStub{}, time.Second, func() bool { return true })
	response := httptest.NewRecorder()
	health.Live(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if response.Code != http.StatusOK || response.Body.String() != "{\"status\":\"live\"}\n" {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestHealthReady(t *testing.T) {
	tests := []struct {
		name   string
		ready  bool
		ping   error
		status int
		body   string
	}{
		{name: "ready", ready: true, status: http.StatusOK, body: "{\"status\":\"ready\"}\n"},
		{name: "draining", ready: false, status: http.StatusServiceUnavailable, body: "{\"status\":\"unavailable\"}\n"},
		{name: "database unavailable", ready: true, ping: errors.New("unavailable"), status: http.StatusServiceUnavailable, body: "{\"status\":\"unavailable\"}\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			health := NewHealth(pingerStub{err: tt.ping}, time.Second, func() bool { return tt.ready })
			response := httptest.NewRecorder()
			health.Ready(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
			if response.Code != tt.status || response.Body.String() != tt.body {
				t.Fatalf("response = %d %q, want %d %q", response.Code, response.Body.String(), tt.status, tt.body)
			}
		})
	}
}
