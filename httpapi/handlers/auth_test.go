package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
)

type authStub struct {
	registerSession auth.Session
	registerErr     error
	loginSession    auth.Session
	loginErr        error
}

func (stub authStub) Register(context.Context, string, string) (auth.Session, error) {
	return stub.registerSession, stub.registerErr
}

func (stub authStub) Login(context.Context, string, string) (auth.Session, error) {
	return stub.loginSession, stub.loginErr
}

func TestAuthRegisterSetsSecureSessionCookie(t *testing.T) {
	session := auth.Session{Token: "opaque-session", ExpiresAt: time.Now().Add(time.Hour)}
	handler := NewAuth(authStub{registerSession: session}, "tidekeepers_session", true, "")
	response := serveAuth(handler.Register, `{"email":"player@example.com","password":"a-long-password"}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookie count = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "tidekeepers_session" || cookie.Value != session.Token {
		t.Errorf("cookie = %#v, want session cookie", cookie)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" {
		t.Errorf("cookie security settings = %#v", cookie)
	}
}

func TestAuthLoginMapsErrorsWithoutCookie(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid input", err: auth.ErrInvalidInput, wantStatus: http.StatusBadRequest, wantCode: "INVALID_AUTH_INPUT"},
		{name: "invalid credentials", err: auth.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: "INVALID_CREDENTIALS"},
		{name: "unexpected", err: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewAuth(authStub{loginErr: test.err}, "tidekeepers_session", false, "")
			response := serveAuth(handler.Login, `{"email":"player@example.com","password":"a-long-password"}`)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), `"code":"`+test.wantCode+`"`) {
				t.Errorf("body = %q, want code %q", response.Body.String(), test.wantCode)
			}
			if len(response.Result().Cookies()) != 0 {
				t.Error("error response set a session cookie")
			}
		})
	}
}

func TestAuthRejectsMalformedCredentialRequest(t *testing.T) {
	handler := NewAuth(authStub{}, "tidekeepers_session", false, "")
	response := serveAuth(handler.Login, `{"email":"player@example.com","password":"a-long-password","unexpected":true}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if !strings.Contains(response.Body.String(), `"requestId":"req_test"`) {
		t.Errorf("body = %q, want request ID", response.Body.String())
	}
}

func serveAuth(handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "req_test")
	response := httptest.NewRecorder()
	apiMiddleware.RequestID(handler).ServeHTTP(response, request)
	return response
}
