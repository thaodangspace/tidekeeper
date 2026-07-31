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
func NewFoundationRouter(health *handlers.Health, accountAuth *handlers.Auth, authenticator auth.Authenticator, playerMe *handlers.Me, keeperUnlocks *handlers.KeeperUnlocks, voyageKeepers *handlers.VoyageKeepers, dailyContext *handlers.DailyContextHandler, voyageHandler *handlers.VoyageHandler, cookieName string, cookieSecure bool, cookieDomain string, logger *slog.Logger) http.Handler {
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
		protected.Get("/api/v1/me/keeper-unlocks", keeperUnlocks.List)
		protected.Get("/api/v1/voyages/current/keepers", voyageKeepers.List)
		protected.Get("/api/v1/voyages/current/daily-context", dailyContext.GetCurrent)
		protected.Post("/api/v1/voyages", voyageHandler.Create)
		protected.Get("/api/v1/voyages/current", voyageHandler.GetCurrent)
		protected.Get("/api/v1/voyages/{voyageId}", voyageHandler.GetByID)
		protected.Post("/api/v1/voyages/{voyageId}/abandon", voyageHandler.Abandon)
		protected.Get("/api/v1/voyages/{voyageId}/history", voyageHandler.GetHistory)
	})
	return router
}
