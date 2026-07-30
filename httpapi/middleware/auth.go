package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
)

// RequireAuthentication validates the configured opaque session cookie and
// attaches its authenticated principal to the request context.
func RequireAuthentication(authenticator auth.Authenticator, cookieName string, cookieSecure bool, cookieDomain string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			cookie, err := request.Cookie(cookieName)
			if errors.Is(err, http.ErrNoCookie) {
				writeUnauthorized(writer, request, "AUTH_REQUIRED", "Authentication is required.", false, cookieName, cookieSecure, cookieDomain)
				return
			}
			if err != nil {
				writeUnauthorized(writer, request, "SESSION_EXPIRED", "The session has expired.", true, cookieName, cookieSecure, cookieDomain)
				return
			}

			digest, err := auth.DigestToken(cookie.Value)
			if err != nil {
				writeUnauthorized(writer, request, "SESSION_EXPIRED", "The session has expired.", true, cookieName, cookieSecure, cookieDomain)
				return
			}
			principal, err := authenticator.Authenticate(request.Context(), digest)
			if errors.Is(err, auth.ErrSessionExpired) {
				writeUnauthorized(writer, request, "SESSION_EXPIRED", "The session has expired.", true, cookieName, cookieSecure, cookieDomain)
				return
			}
			if err != nil {
				writer.Header().Set("Cache-Control", "private, no-store")
				response.JSON(writer, http.StatusInternalServerError, apiError{
					Code:      "INTERNAL_ERROR",
					Message:   "An unexpected error occurred.",
					RequestID: RequestIDFromContext(request.Context()),
				})
				return
			}

			next.ServeHTTP(writer, request.WithContext(auth.WithPrincipal(request.Context(), principal)))
		})
	}
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
}

func writeUnauthorized(writer http.ResponseWriter, request *http.Request, code, message string, clearCookie bool, cookieName string, cookieSecure bool, cookieDomain string) {
	writer.Header().Set("Cache-Control", "private, no-store")
	if clearCookie {
		http.SetCookie(writer, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			Domain:   cookieDomain,
			Expires:  time.Unix(1, 0).UTC(),
			MaxAge:   -1,
			Secure:   cookieSecure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}
	response.JSON(writer, http.StatusUnauthorized, apiError{
		Code:      code,
		Message:   message,
		RequestID: RequestIDFromContext(request.Context()),
	})
}
