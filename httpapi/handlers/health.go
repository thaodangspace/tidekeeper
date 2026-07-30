// Package handlers implements the Tidekeepers HTTP boundary.
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
)

// DatabasePinger is the readiness dependency used by the health handler.
type DatabasePinger interface {
	Ping(context.Context) error
}

// Health serves process liveness and bounded dependency readiness.
type Health struct {
	database DatabasePinger
	timeout  time.Duration
	ready    func() bool
}

// NewHealth constructs health handlers.
func NewHealth(database DatabasePinger, timeout time.Duration, ready func() bool) *Health {
	return &Health{database: database, timeout: timeout, ready: ready}
}

// Live reports only process responsiveness.
func (h *Health) Live(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	response.JSON(writer, http.StatusOK, map[string]string{"status": "live"})
}

// Ready performs a bounded PostgreSQL ping and observes shutdown state.
func (h *Health) Ready(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if !h.ready() {
		response.JSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}

	ctx, cancel := context.WithTimeout(request.Context(), h.timeout)
	defer cancel()
	if err := h.database.Ping(ctx); err != nil {
		response.JSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	response.JSON(writer, http.StatusOK, map[string]string{"status": "ready"})
}
