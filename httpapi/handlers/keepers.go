package handlers

import (
	"context"
	"net/http"

	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
	"github.com/thaodangspace/tidekeepers-server/player"
)

// PlayerKeeperReader retrieves the authenticated player's Keeper inventory.
type PlayerKeeperReader interface {
	ListKeepers(context.Context, player.ID) ([]player.Keeper, error)
}

// PlayerKeepers serves the authenticated player's Keeper inventory.
type PlayerKeepers struct {
	players PlayerKeeperReader
}

// NewPlayerKeepers constructs the player Keeper inventory handler.
func NewPlayerKeepers(players PlayerKeeperReader) *PlayerKeepers {
	return &PlayerKeepers{players: players}
}

// List returns the current player's permanently owned Keeper instances.
func (h *PlayerKeepers) List(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	keepers, err := h.players.ListKeepers(request.Context(), player.ID(principal.PlayerID))
	if err != nil {
		writer.Header().Set("Cache-Control", "private, no-store")
		response.JSON(writer, http.StatusInternalServerError, apiError{
			Code:      "INTERNAL_ERROR",
			Message:   "An unexpected error occurred.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
		return
	}

	result := make([]keeperResponse, len(keepers))
	for index, keeper := range keepers {
		result[index] = keeperResponse{
			ID:            keeper.PublicID,
			DefinitionKey: keeper.DefinitionKey,
			Name:          keeper.Name,
			CurrentName:   keeper.CurrentName,
			Sector:        keeper.Sector,
			Role:          keeper.Role,
			Rarity:        keeper.Rarity,
			Level:         keeper.Level,
		}
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, playerKeepersResponse{Keepers: result})
}

type playerKeepersResponse struct {
	Keepers []keeperResponse `json:"keepers"`
}

type keeperResponse struct {
	ID            string `json:"id"`
	DefinitionKey string `json:"definitionKey"`
	Name          string `json:"name"`
	CurrentName   string `json:"currentName"`
	Sector        string `json:"sector"`
	Role          string `json:"role"`
	Rarity        string `json:"rarity"`
	Level         int    `json:"level"`
}

func writeAuthenticationRequired(writer http.ResponseWriter, ctx context.Context) {
	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusUnauthorized, apiError{
		Code:      "AUTH_REQUIRED",
		Message:   "Authentication is required.",
		RequestID: apiMiddleware.RequestIDFromContext(ctx),
	})
}
