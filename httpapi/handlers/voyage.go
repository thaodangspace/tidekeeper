package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/thaodangspace/tidekeepers-server/auth"
	apiMiddleware "github.com/thaodangspace/tidekeepers-server/httpapi/middleware"
	"github.com/thaodangspace/tidekeepers-server/httpapi/response"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

type VoyageCreator interface {
	Create(ctx context.Context, playerID string, idempotencyKey string) (*voyage.VoyageResponse, error)
}

type VoyageReader interface {
	GetCurrent(ctx context.Context, playerID string) (*voyage.VoyageResponse, error)
	GetByID(ctx context.Context, playerID string, voyageID string) (*voyage.VoyageResponse, error)
}

type VoyageAbandoner interface {
	Abandon(ctx context.Context, playerID string, voyageID string, idempotencyKey string) (*voyage.VoyageResponse, error)
}

type VoyageHistoryReader interface {
	GetHistory(ctx context.Context, playerID string, voyageID string) (*voyage.VoyageHistoryResponse, error)
}

type VoyageService interface {
	VoyageCreator
	VoyageReader
	VoyageAbandoner
	VoyageHistoryReader
}

type VoyageHandler struct {
	service VoyageService
}

func NewVoyageHandler(service VoyageService) *VoyageHandler {
	return &VoyageHandler{service: service}
}

func (h *VoyageHandler) Create(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	idempotencyKey := request.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.writeError(writer, request, voyage.ErrInvalidIdempotencyKey)
		return
	}

	resp, err := h.service.Create(request.Context(), principal.PlayerID, idempotencyKey)
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusCreated, resp)
}

func (h *VoyageHandler) GetCurrent(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	resp, err := h.service.GetCurrent(request.Context(), principal.PlayerID)
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, resp)
}

func (h *VoyageHandler) GetByID(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	voyageID := chi.URLParam(request, "voyageId")
	if voyageID == "" {
		h.writeError(writer, request, voyage.ErrVoyageNotFound)
		return
	}

	resp, err := h.service.GetByID(request.Context(), principal.PlayerID, voyageID)
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, resp)
}

func (h *VoyageHandler) Abandon(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	voyageID := chi.URLParam(request, "voyageId")
	if voyageID == "" {
		h.writeError(writer, request, voyage.ErrVoyageNotFound)
		return
	}

	idempotencyKey := request.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		h.writeError(writer, request, voyage.ErrInvalidIdempotencyKey)
		return
	}

	resp, err := h.service.Abandon(request.Context(), principal.PlayerID, voyageID, idempotencyKey)
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, resp)
}

func (h *VoyageHandler) GetHistory(writer http.ResponseWriter, request *http.Request) {
	principal, ok := auth.PrincipalFromContext(request.Context())
	if !ok {
		writeAuthenticationRequired(writer, request.Context())
		return
	}

	voyageID := chi.URLParam(request, "voyageId")
	if voyageID == "" {
		h.writeError(writer, request, voyage.ErrVoyageNotFound)
		return
	}

	resp, err := h.service.GetHistory(request.Context(), principal.PlayerID, voyageID)
	if err != nil {
		h.writeError(writer, request, err)
		return
	}

	writer.Header().Set("Cache-Control", "private, no-store")
	response.JSON(writer, http.StatusOK, resp)
}

func (h *VoyageHandler) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	writeVoyageAPIError(writer, request, err)
}

// writeVoyageAPIError maps Voyage service errors to stable public responses.
func writeVoyageAPIError(writer http.ResponseWriter, request *http.Request, err error) {
	writer.Header().Set("Cache-Control", "private, no-store")

	switch {
	case errors.Is(err, voyage.ErrInvalidIdempotencyKey):
		response.JSON(writer, http.StatusBadRequest, apiError{
			Code:      "INVALID_IDEMPOTENCY_KEY",
			Message:   "Invalid idempotency key.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrIdempotencyKeyReused):
		response.JSON(writer, http.StatusConflict, apiError{
			Code:      "IDEMPOTENCY_KEY_REUSED",
			Message:   "Idempotency key was reused with a different request.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrActiveVoyageExists):
		response.JSON(writer, http.StatusConflict, apiError{
			Code:      "ACTIVE_VOYAGE_EXISTS",
			Message:   "An active voyage already exists.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrNoActiveVoyage):
		response.JSON(writer, http.StatusNotFound, apiError{
			Code:      "NO_ACTIVE_VOYAGE",
			Message:   "No current voyage.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrVoyageNotFound):
		response.JSON(writer, http.StatusNotFound, apiError{
			Code:      "VOYAGE_NOT_FOUND",
			Message:   "Voyage not found.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrVoyageTerminal):
		response.JSON(writer, http.StatusConflict, apiError{
			Code:      "VOYAGE_NOT_ACTIVE",
			Message:   "Voyage is not active.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrInitializationUnavailable):
		response.JSON(writer, http.StatusServiceUnavailable, apiError{
			Code:      "VOYAGE_INITIALIZATION_UNAVAILABLE",
			Message:   "Voyage initialization is currently unavailable.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case errors.Is(err, voyage.ErrServiceUnavailable):
		response.JSON(writer, http.StatusServiceUnavailable, apiError{
			Code:      "SERVICE_UNAVAILABLE",
			Message:   "Service is temporarily unavailable.",
			RequestID: apiMiddleware.RequestIDFromContext(request.Context()),
		})
	case strings.Contains(err.Error(), "internal error") || strings.HasSuffix(err.Error(), "internal error"):
		response.JSON(writer, http.StatusInternalServerError, apiError{
			Code:      "INTERNAL_ERROR",
			Message:   "An unexpected error occurred.",
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
