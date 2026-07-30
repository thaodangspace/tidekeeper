// Package database owns PostgreSQL connectivity and dependency error classification.
package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/config"
)

var (
	// ErrInvalidConfiguration avoids returning a PostgreSQL URL through parse errors.
	ErrInvalidConfiguration = errors.New("invalid database configuration")
)

// NewPool creates a lazy PostgreSQL pool. Readiness, not startup, verifies connectivity.
func NewPool(ctx context.Context, cfg config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.ConnConfig.ConnectTimeout = cfg.AcquireTimeout
	poolConfig.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, ErrInvalidConfiguration
	}
	return pool, nil
}
