package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/daily"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
)

// DailyContextReader retrieves the authenticated player's daily context.
type DailyContextReader interface {
	GetCurrentDailyContext(context.Context, daily.ID) (daily.DailyContext, error)
}

// DailyContextHandler serves the current voyage daily context endpoint.
type DailyContextHandler struct {
	dailyService DailyContextReader
}

// NewDailyContextHandler constructs the daily context HTTP handler.
func NewDailyContextHandler(dailyService DailyContextReader) *DailyContextHandler {
	return &DailyContextHandler{dailyService: dailyService}
}

// GetCurrent returns the current voyage daily context for the authenticated player.
func (h *DailyContextHandler) GetCurrent(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	ctx, err := h.dailyService.GetCurrentDailyContext(request.Context(), daily.ID(principal.PlayerID))
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, toResponse(ctx))
}

// writeError maps domain errors to stable HTTP status codes and JSON bodies.
func (h *DailyContextHandler) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	writer.Header().Set("Cache-Control", "private, no-store")

	switch {
	case errors.Is(err, daily.ErrNoActiveVoyage):
		response.JSON(writer, http.StatusNotFound, apiError{
			Code:      "NO_ACTIVE_VOYAGE",
			Message:   "No active voyage.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, daily.ErrVoyageNotFound):
		response.JSON(writer, http.StatusNotFound, apiError{
			Code:      "VOYAGE_NOT_FOUND",
			Message:   "Voyage not found.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, daily.ErrServiceUnavailable):
		response.JSON(writer, http.StatusServiceUnavailable, apiError{
			Code:      "SERVICE_UNAVAILABLE",
			Message:   "Service is temporarily unavailable.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	default:
		response.JSON(writer, http.StatusInternalServerError, apiError{
			Code:      "INTERNAL_ERROR",
			Message:   "An unexpected error occurred.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	}
}

func toResponse(ctx daily.DailyContext) map[string]any {
	dailyMap := map[string]any{
		"phase":              ctx.Daily.Phase,
		"lockAt":             ctx.Daily.LockAt,
		"settleAfter":        ctx.Daily.SettleAfter,
		"version":            ctx.Daily.Version,
		"modifier":           modifierToMap(ctx.Daily.Modifier),
		"objective":          objectiveToMap(ctx.Daily.Objective),
		"signals":            signalsToList(ctx.Daily.Signals),
		"lineup":             lineupToMap(ctx.Daily.Lineup),
		"inventory":          keepersToList(ctx.Daily.Inventory),
		"shop":               shopToMap(ctx.Daily.Shop),
		"strategies":         strategiesToList(ctx.Daily.Strategies),
		"selectedStrategyId": ctx.Daily.SelectedStrategyID,
		"pendingRewardCount": ctx.Daily.PendingRewardCount,
	}

	return map[string]any{
		"serverNow": ctx.ServerNow,
		"voyage": map[string]any{
			"id":            ctx.Voyage.PublicID,
			"status":        ctx.Voyage.Status,
			"dayNumber":     ctx.Voyage.DayNumber,
			"fundHealth":    ctx.Voyage.FundHealth,
			"maxFundHealth": ctx.Voyage.MaxFundHealth,
			"capital":       ctx.Voyage.Capital,
			"score":         ctx.Voyage.Score,
		},
		"daily": dailyMap,
	}
}

func modifierToMap(m daily.Modifier) map[string]any {
	return map[string]any{
		"id":          m.ID,
		"name":        m.Name,
		"description": m.Description,
	}
}

func objectiveToMap(o daily.Objective) map[string]any {
	return map[string]any{
		"id":            o.ID,
		"name":          o.Name,
		"description":   o.Description,
		"progressLabel": o.ProgressLabel,
		"rewardLabel":   o.RewardLabel,
	}
}

func signalsToList(signals []daily.Signal) []map[string]any {
	result := make([]map[string]any, len(signals))
	for i, s := range signals {
		result[i] = map[string]any{
			"id":           s.ID,
			"name":         s.Name,
			"description":  s.Description,
			"direction":    s.Direction,
			"strength":     s.Strength,
			"observedFrom": s.ObservedFrom,
			"observedTo":   s.ObservedTo,
		}
	}
	return result
}

func lineupToMap(l daily.Lineup) map[string]any {
	slots := make([]map[string]any, len(l.Slots))
	for i, slot := range l.Slots {
		slotMap := map[string]any{
			"index": slot.Index,
		}
		if slot.Keeper != nil {
			slotMap["keeper"] = keeperToMap(*slot.Keeper)
		} else {
			slotMap["keeper"] = nil
		}
		slots[i] = slotMap
	}

	return map[string]any{
		"lockedAt":  l.LockedAt,
		"maxSlots":  l.MaxSlots,
		"slots":     slots,
		"synergies": synergiesToList(l.Synergies),
		"warnings":  warningsToList(l.Warnings),
	}
}

func synergiesToList(synergies []daily.Synergy) []map[string]any {
	result := make([]map[string]any, len(synergies))
	for i, s := range synergies {
		result[i] = map[string]any{
			"id":            s.ID,
			"name":          s.Name,
			"description":   s.Description,
			"state":         s.State,
			"currentCount":  s.CurrentCount,
			"requiredCount": s.RequiredCount,
		}
	}
	return result
}

func warningsToList(warnings []daily.FleetWarning) []map[string]any {
	result := make([]map[string]any, len(warnings))
	for i, w := range warnings {
		result[i] = map[string]any{
			"code":     w.Code,
			"message":  w.Message,
			"severity": w.Severity,
		}
	}
	return result
}

func keeperToMap(k daily.Keeper) map[string]any {
	return map[string]any{
		"id":             k.ID,
		"definitionId":   k.DefinitionID,
		"name":           k.Name,
		"level":          k.Level,
		"rarity":         k.Rarity,
		"role":           k.Role,
		"sector":         k.Sector,
		"passiveSummary": k.PassiveSummary,
		"artworkUrl":     k.ArtworkURL,
	}
}

func keepersToList(keepers []daily.Keeper) []map[string]any {
	result := make([]map[string]any, len(keepers))
	for i, k := range keepers {
		result[i] = keeperToMap(k)
	}
	return result
}

func shopToMap(s daily.Shop) map[string]any {
	offers := make([]map[string]any, len(s.Offers))
	for i, o := range s.Offers {
		offer := map[string]any{
			"id":        o.ID,
			"keeper":    keeperToMap(o.Keeper),
			"cost":      o.Cost,
			"available": o.Available,
		}
		if o.SynergyHint != nil {
			offer["synergyHint"] = o.SynergyHint
		} else {
			offer["synergyHint"] = nil
		}
		offers[i] = offer
	}

	return map[string]any{
		"offers":      offers,
		"refreshAt":   s.RefreshAt,
		"rerollCost":  s.RerollCost,
		"rerollIndex": s.RerollIndex,
	}
}

func strategiesToList(strategies []daily.Strategy) []map[string]any {
	result := make([]map[string]any, len(strategies))
	for i, s := range strategies {
		result[i] = map[string]any{
			"id":          s.ID,
			"name":        s.Name,
			"description": s.Description,
			"upside":      s.Upside,
			"downside":    s.Downside,
			"available":   s.Available,
		}
	}
	return result
}
