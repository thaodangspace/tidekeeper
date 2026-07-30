package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/auth"
)

type sessionAuthenticatorStub struct {
	principal auth.Principal
	err       error
}

func (stub sessionAuthenticatorStub) Authenticate(context.Context, auth.Digest) (auth.Principal, error) {
	return stub.principal, stub.err
}

func TestRequireAuthenticationAddsPrincipal(t *testing.T) {
	token, _, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	next := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		principal, ok := auth.PrincipalFromContext(request.Context())
		if !ok || principal.PlayerID != "player-id" {
			t.Fatal("authenticated principal missing from request context")
		}
		writer.WriteHeader(http.StatusNoContent)
	})
	handler := RequireAuthentication(sessionAuthenticatorStub{principal: auth.Principal{PlayerID: "player-id"}}, "tidekeepers_session", false, "")(next)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request.AddCookie(&http.Cookie{Name: "tidekeepers_session", Value: token})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestRequireAuthenticationRejectsMissingAndExpiredSessions(t *testing.T) {
	tests := []struct {
		name      string
		cookie    *http.Cookie
		err       error
		wantCode  string
		wantClear bool
	}{
		{name: "missing", wantCode: "AUTH_REQUIRED"},
		{name: "malformed", cookie: &http.Cookie{Name: "tidekeepers_session", Value: "not-a-token"}, wantCode: "SESSION_EXPIRED", wantClear: true},
		{name: "expired", cookie: validSessionCookie(t), err: auth.ErrSessionExpired, wantCode: "SESSION_EXPIRED", wantClear: true},
		{name: "backend failure", cookie: validSessionCookie(t), err: errors.New("database unavailable"), wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := RequireAuthentication(sessionAuthenticatorStub{err: test.err}, "tidekeepers_session", false, "")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler was called")
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			if test.cookie != nil {
				request.AddCookie(test.cookie)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			wantStatus := http.StatusUnauthorized
			if test.err != nil && !errors.Is(test.err, auth.ErrSessionExpired) {
				wantStatus = http.StatusInternalServerError
			}
			if response.Code != wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, wantStatus)
			}
			if !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Errorf("body = %q, want code %q", response.Body.String(), test.wantCode)
			}
			if got := len(response.Result().Cookies()) > 0; got != test.wantClear {
				t.Errorf("cleared cookie = %t, want %t", got, test.wantClear)
			}
		})
	}
}

func validSessionCookie(t *testing.T) *http.Cookie {
	t.Helper()
	token, _, err := auth.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	return &http.Cookie{Name: "tidekeepers_session", Value: token}
}
