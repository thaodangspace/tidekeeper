// Package app composes and runs Tidekeepers process entry points.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/config"
	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/database"
	"github.com/thaodangspace/tidekeepers-server/httpapi"
	"github.com/thaodangspace/tidekeepers-server/httpapi/handlers"
	"github.com/thaodangspace/tidekeepers-server/player"
)

// API is the runnable HTTP API and its owned resources.
type API struct {
	server *http.Server
	pool   *pgxpool.Pool
	logger *slog.Logger
	ready  atomic.Bool
}

// NewAPI composes a lazy database pool, health handlers, and bounded HTTP server.
func NewAPI(ctx context.Context, cfg config.Config, logger *slog.Logger) (*API, error) {
	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	api := &API{pool: pool, logger: logger}
	api.ready.Store(true)
	health := handlers.NewHealth(pool, cfg.Database.HealthTimeout, api.ready.Load)
	sessions := auth.NewService(pool, cfg.Session.TTL)
	accounts := handlers.NewAuth(
		sessions,
		cfg.Session.CookieName,
		cfg.Session.CookieSecure,
		cfg.Session.CookieDomain,
	)
	players := player.NewService(player.NewRepository(pool))
	playerMe := handlers.NewMe(players)
	playerKeepers := handlers.NewPlayerKeepers(players)
	dailyService := daily.NewService(daily.NewRepository(pool))
	dailyContext := handlers.NewDailyContextHandler(dailyService)
	api.server = &http.Server{
		Addr: cfg.HTTP.Address,
		Handler: httpapi.NewFoundationRouter(
			health, accounts, sessions, playerMe, playerKeepers, dailyContext,
			cfg.Session.CookieName, cfg.Session.CookieSecure, cfg.Session.CookieDomain, logger,
		),
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}
	return api, nil
}

// Run serves until the context is canceled or the listener fails.
func (a *API) Run(ctx context.Context, shutdownTimeout config.HTTP) error {
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- a.server.ListenAndServe()
	}()

	select {
	case err := <-serveErrors:
		a.ready.Store(false)
		a.pool.Close()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		a.ready.Store(false)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout.ShutdownTimeout)
		defer cancel()
		err := a.server.Shutdown(shutdownCtx)
		a.pool.Close()
		return err
	}
}
