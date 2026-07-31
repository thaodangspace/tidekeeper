package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
	"github.com/thaodangspace/tidekeepers-server/player"
)

// KeeperUnlockReader retrieves the authenticated player's permanent unlocks.
type KeeperUnlockReader interface {
	ListUnlocks(context.Context, player.ID) ([]player.KeeperUnlock, error)
}

// KeeperUnlocks serves the authenticated player's permanent Keeper unlocks.
type KeeperUnlocks struct {
	players KeeperUnlockReader
}

// NewKeeperUnlocks constructs the meta unlock handler.
func NewKeeperUnlocks(players KeeperUnlockReader) *KeeperUnlocks {
	return &KeeperUnlocks{players: players}
}

// List returns the current player's permanent archetype unlocks.
func (h *KeeperUnlocks) List(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	unlocks, err := h.players.ListUnlocks(request.Context(), player.ID(principal.PlayerID))
	if err != nil {
		writer.Header().Set("Cache-Control", "private, no-store")
		response.JSON(writer, http.StatusInternalServerError, apiError{
			Code:      "INTERNAL_ERROR",
			Message:   "An unexpected error occurred.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
		return
	}

	result := make([]keeperUnlockResponse, len(unlocks))
	for index, unlock := range unlocks {
		result[index] = keeperUnlockResponse{
			DefinitionKey:     unlock.DefinitionKey,
			DefinitionVersion: unlock.DefinitionVersion,
			Name:              unlock.Name,
			CurrentName:       unlock.CurrentName,
			Sector:            unlock.Sector,
			Role:              unlock.Role,
			Rarity:            unlock.Rarity,
			UnlockSource:      unlock.UnlockSource,
			UnlockedAt:        unlock.UnlockedAt,
		}
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, keeperUnlocksResponse{Unlocks: result})
}

type keeperUnlocksResponse struct {
	Unlocks []keeperUnlockResponse `json:"unlocks"`
}

type keeperUnlockResponse struct {
	DefinitionKey     string    `json:"definitionKey"`
	DefinitionVersion int64     `json:"definitionVersion"`
	Name              string    `json:"name"`
	CurrentName       string    `json:"currentName"`
	Sector            string    `json:"sector"`
	Role              string    `json:"role"`
	Rarity            string    `json:"rarity"`
	UnlockSource      string    `json:"unlockSource"`
	UnlockedAt        time.Time `json:"unlockedAt"`
}
