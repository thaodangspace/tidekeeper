// Package httpapi composes the Tidekeepers HTTP routes and middleware.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/httpapi/handlers"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
)

// NewFoundationRouter builds public, authenticated, and operational routes.
func NewFoundationRouter(health *handlers.Health, accountAuth *handlers.Auth, authenticator auth.Authenticator, playerMe *handlers.Me, playerKeepers *handlers.PlayerKeepers, cookieName string, cookieSecure bool, cookieDomain string, logger *slog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(apiMiddleware.RequestID)
	router.Use(apiMiddleware.SecurityHeaders)
	router.Use(apiMiddleware.RequestLogger(logger))
	router.Post("/auth/register", accountAuth.Register)
	router.Post("/auth/login", accountAuth.Login)
	router.Get("/health/live", health.Live)
	router.Get("/health/ready", health.Ready)
	router.Group(func(protected chi.Router) {
		protected.Use(apiMiddleware.RequireAuthentication(authenticator, cookieName, cookieSecure, cookieDomain))
		protected.Get("/api/v1/me", playerMe.Get)
		protected.Get("/api/v1/me/keepers", playerKeepers.List)
	})
	return router
}
