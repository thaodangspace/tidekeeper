package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
)

const maxAuthRequestBytes = 8 << 10

// AccountAuthenticator is the registration/login dependency used by Auth.
type AccountAuthenticator interface {
	Register(context.Context, string, string) (auth.Session, error)
	Login(context.Context, string, string) (auth.Session, error)
}

// Auth serves public account registration and login endpoints.
type Auth struct {
	authenticator AccountAuthenticator
	cookieName    string
	cookieSecure  bool
	cookieDomain  string
}

// NewAuth constructs auth HTTP handlers with the deployment cookie settings.
func NewAuth(authenticator AccountAuthenticator, cookieName string, cookieSecure bool, cookieDomain string) *Auth {
	return &Auth{
		authenticator: authenticator,
		cookieName:    cookieName,
		cookieSecure:  cookieSecure,
		cookieDomain:  cookieDomain,
	}
}

// Register creates an account and returns its newly issued session cookie.
func (h *Auth) Register(writer http.ResponseWriter, request *http.Request) {
	credentials, ok := decodeCredentials(writer, request)
	if !ok {
		return
	}
	session, err := h.authenticator.Register(request.Context(), credentials.Email, credentials.Password)
	if !h.respondError(writer, request, err) {
		return
	}
	h.setSessionCookie(writer, session)
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusCreated)
}

// Login verifies credentials and returns a newly issued session cookie.
func (h *Auth) Login(writer http.ResponseWriter, request *http.Request) {
	credentials, ok := decodeCredentials(writer, request)
	if !ok {
		return
	}
	session, err := h.authenticator.Login(request.Context(), credentials.Email, credentials.Password)
	if !h.respondError(writer, request, err) {
		return
	}
	h.setSessionCookie(writer, session)
	writer.Header().Set("Cache-Control", "no-store")
	writer.WriteHeader(http.StatusNoContent)
}

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func writeAuthenticationRequired(writer http.ResponseWriter, ctx context.Context) {
	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusUnauthorized, apiError{
		Code:      "AUTH_REQUIRED",
		Message:   "Authentication is required.",
		RequestID: apiMiddleware.RequestIDFromContext(ctx),
	})
}

func decodeCredentials(writer http.ResponseWriter, request *http.Request) (credentialsRequest, bool) {
	request.Body = http.MaxBytesReader(writer, request.Body, maxAuthRequestBytes)
	defer request.Body.Close()

	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var credentials credentialsRequest
	if err := decoder.Decode(&credentials); err != nil {
		writeAuthError(writer, request, http.StatusBadRequest, "INVALID_AUTH_INPUT", "Enter a valid email and password.")
		return credentialsRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAuthError(writer, request, http.StatusBadRequest, "INVALID_AUTH_INPUT", "Enter a valid email and password.")
		return credentialsRequest{}, false
	}
	return credentials, true
}

func (h *Auth) respondError(writer http.ResponseWriter, request *http.Request, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, auth.ErrInvalidInput):
		writeAuthError(writer, request, http.StatusBadRequest, "INVALID_AUTH_INPUT", "Enter a valid email and password.")
	case errors.Is(err, auth.ErrEmailTaken):
		writeAuthError(writer, request, http.StatusConflict, "EMAIL_ALREADY_REGISTERED", "An account already exists for this email address.")
	case errors.Is(err, auth.ErrInvalidCredentials):
		writeAuthError(writer, request, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password.")
	default:
		writeAuthError(writer, request, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
	}
	return false
}

func (h *Auth) setSessionCookie(writer http.ResponseWriter, session auth.Session) {
	http.SetCookie(writer, &http.Cookie{
		Name:     h.cookieName,
		Value:    session.Token,
		Path:     "/",
		Domain:   h.cookieDomain,
		Expires:  session.ExpiresAt.UTC(),
		MaxAge:   max(1, int(time.Until(session.ExpiresAt).Seconds())),
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeAuthError(writer http.ResponseWriter, request *http.Request, status int, code, message string) {
	writer.Header().Set("Cache-Control", "no-store")
	response.JSON(writer, status, apiError{
		Code:      code,
		Message:   message,
		RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
	})
}
