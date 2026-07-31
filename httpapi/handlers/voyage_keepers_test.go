package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

type voyageKeeperInventoryStub struct {
	gotID     string
	inventory *voyage.VoyageKeepers
	err       error
}

func (stub *voyageKeeperInventoryStub) GetActiveVoyageKeepers(_ context.Context, playerID string) (*voyage.VoyageKeepers, error) {
	stub.gotID = playerID
	return stub.inventory, stub.err
}

func TestVoyageKeepersListReturnsActiveInventory(t *testing.T) {
	acquiredAt := time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC)
	stub := &voyageKeeperInventoryStub{inventory: &voyage.VoyageKeepers{
		VoyageID: "voy_123",
		Keepers: []voyage.VoyageKeeper{{
			PublicID:          "kpr_123",
			DefinitionKey:     "crest_sovereign",
			DefinitionVersion: 1,
			Name:              "Crest Sovereign",
			CurrentName:       "Sovereign Current",
			Sector:            "CREST",
			Role:              "VANGUARD",
			Rarity:            "COMMON",
			UpgradeNodeKey:    "base",
			Level:             1,
			AcquiredDay:       1,
			AcquiredSource:    "STARTER",
			AcquiredAt:        acquiredAt,
		}},
	}}
	handler := NewVoyageKeepers(stub)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if stub.gotID != "internal-player-id" {
		t.Errorf("service player ID = %q, want internal-player-id", stub.gotID)
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", response.Header().Get("Cache-Control"))
	}
	for _, want := range []string{
		`"voyageId":"voy_123"`,
		`"id":"kpr_123"`,
		`"upgradeNodeKey":"base"`,
		`"level":1`,
		`"acquiredDay":1`,
		`"acquiredSource":"STARTER"`,
		`"acquiredAt":"2026-01-15T08:30:00Z"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body = %q, want %q", response.Body.String(), want)
		}
	}
}

func TestVoyageKeepersListReturnsEmptyInventory(t *testing.T) {
	handler := NewVoyageKeepers(&voyageKeeperInventoryStub{inventory: &voyage.VoyageKeepers{VoyageID: "voy_123"}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"voyageId\":\"voy_123\",\"keepers\":[]}\n" {
		t.Errorf("body = %q, want active voyage with empty keepers", response.Body.String())
	}
}

func TestVoyageKeepersListRejectsMissingPrincipal(t *testing.T) {
	handler := NewVoyageKeepers(&voyageKeeperInventoryStub{})
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Errorf("body = %q, want AUTH_REQUIRED", response.Body.String())
	}
}

func TestVoyageKeepersListMapsNoActiveVoyage(t *testing.T) {
	handler := NewVoyageKeepers(&voyageKeeperInventoryStub{err: voyage.ErrNoActiveVoyage})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if !strings.Contains(response.Body.String(), `"code":"NO_ACTIVE_VOYAGE"`) {
		t.Errorf("body = %q, want NO_ACTIVE_VOYAGE", response.Body.String())
	}
}

func TestVoyageKeepersListMapsServiceUnavailable(t *testing.T) {
	handler := NewVoyageKeepers(&voyageKeeperInventoryStub{err: voyage.ErrServiceUnavailable})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(response.Body.String(), `"code":"SERVICE_UNAVAILABLE"`) {
		t.Errorf("body = %q, want SERVICE_UNAVAILABLE", response.Body.String())
	}
}

func TestVoyageKeepersListHidesInternalFailure(t *testing.T) {
	handler := NewVoyageKeepers(&voyageKeeperInventoryStub{err: voyage.ErrInternal})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(response.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Errorf("body = %q, want INTERNAL_ERROR", response.Body.String())
	}
}
