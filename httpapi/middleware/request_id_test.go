package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		reused   bool
	}{
		{name: "valid", incoming: "req_client-123", reused: true},
		{name: "missing"},
		{name: "invalid characters", incoming: "bad request id"},
		{name: "too long", incoming: string(make([]byte, 65))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var contextID string
			handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				contextID = RequestIDFromContext(request.Context())
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.Header.Set(requestIDHeader, tt.incoming)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			got := response.Header().Get(requestIDHeader)
			if got != contextID || !validRequestID.MatchString(got) {
				t.Fatalf("response/context request IDs = %q/%q", got, contextID)
			}
			if tt.reused && got != tt.incoming {
				t.Errorf("request ID = %q, want reused %q", got, tt.incoming)
			}
			if !tt.reused && tt.incoming != "" && got == tt.incoming {
				t.Errorf("invalid request ID was reused: %q", got)
			}
		})
	}
}

func TestGeneratedRequestIDShape(t *testing.T) {
	got := newRequestID()
	if !regexp.MustCompile(`^req_[0-9a-f]{32}$`).MatchString(got) {
		t.Fatalf("newRequestID() = %q", got)
	}
}
