package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

// VoyageKeeperInventoryReader retrieves the authenticated player's active
// Voyage keeper inventory.
type VoyageKeeperInventoryReader interface {
	GetActiveVoyageKeepers(context.Context, string) (*voyage.VoyageKeepers, error)
}

// VoyageKeepers serves the authenticated player's current Voyage inventory.
type VoyageKeepers struct {
	service VoyageKeeperInventoryReader
}

// NewVoyageKeepers constructs the current Voyage inventory handler.
func NewVoyageKeepers(service VoyageKeeperInventoryReader) *VoyageKeepers {
	return &VoyageKeepers{service: service}
}

// List returns the active Voyage's keeper instances or NO_ACTIVE_VOYAGE.
func (h *VoyageKeepers) List(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	inventory, err := h.service.GetActiveVoyageKeepers(request.Context(), principal.PlayerID)
	if err != nil {
		writeVoyageAPIError(writer, request, err)
		return
	}

	keepers := make([]voyageKeeperResponse, len(inventory.Keepers))
	for index, keeper := range inventory.Keepers {
		keepers[index] = voyageKeeperResponse{
			ID:                keeper.PublicID,
			DefinitionKey:     keeper.DefinitionKey,
			DefinitionVersion: keeper.DefinitionVersion,
			Name:              keeper.Name,
			CurrentName:       keeper.CurrentName,
			Sector:            keeper.Sector,
			Role:              keeper.Role,
			Rarity:            keeper.Rarity,
			UpgradeNodeKey:    keeper.UpgradeNodeKey,
			Level:             keeper.Level,
			AcquiredDay:       keeper.AcquiredDay,
			AcquiredSource:    keeper.AcquiredSource,
			AcquiredAt:        keeper.AcquiredAt,
		}
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, voyageKeepersResponse{
		VoyageID: inventory.VoyageID,
		Keepers:  keepers,
	})
}

type voyageKeepersResponse struct {
	VoyageID string                 `json:"voyageId"`
	Keepers  []voyageKeeperResponse `json:"keepers"`
}

type voyageKeeperResponse struct {
	ID                string    `json:"id"`
	DefinitionKey     string    `json:"definitionKey"`
	DefinitionVersion int64     `json:"definitionVersion"`
	Name              string    `json:"name"`
	CurrentName       string    `json:"currentName"`
	Sector            string    `json:"sector"`
	Role              string    `json:"role"`
	Rarity            string    `json:"rarity"`
	UpgradeNodeKey    string    `json:"upgradeNodeKey"`
	Level             int       `json:"level"`
	AcquiredDay       int       `json:"acquiredDay"`
	AcquiredSource    string    `json:"acquiredSource"`
	AcquiredAt        time.Time `json:"acquiredAt"`
}
