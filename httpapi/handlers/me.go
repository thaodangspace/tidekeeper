package handlers

import (
	"context"
	"net/http"

	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
	"github.com/thaodangspace/tidekeepers-server/player"
)

// PlayerReader retrieves the authenticated player's public identity summary.
type PlayerReader interface {
	GetMe(context.Context, player.ID) (player.Me, error)
}

// Me serves the authenticated player identity endpoint.
type Me struct {
	players PlayerReader
}

// NewMe constructs the current-player handler.
func NewMe(players PlayerReader) *Me {
	return &Me{players: players}
}

// Get returns the current authenticated player without exposing internal IDs.
func (h *Me) Get(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	me, err := h.players.GetMe(request.Context(), player.ID(principal.PlayerID))
	if err != nil {
		writer.Header().Set("Cache-Control", "private, no-store")
		response.JSON(writer, http.StatusInternalServerError, apiError{
			Code:      "INTERNAL_ERROR",
			Message:   "An unexpected error occurred.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, meResponse{
		PlayerID:            me.PublicID,
		OnboardingCompleted: me.OnboardingCompleted,
		Locale:              me.Locale,
		Timezone:            me.Timezone,
		ActiveVoyageID:      me.ActiveVoyageID,
	})
}

type meResponse struct {
	PlayerID            string  `json:"playerId"`
	OnboardingCompleted bool    `json:"onboardingCompleted"`
	Locale              string  `json:"locale"`
	Timezone            string  `json:"timezone"`
	ActiveVoyageID      *string `json:"activeVoyageId"`
}
