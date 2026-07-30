package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/player"
)

type playerKeeperReaderStub struct {
	gotID   player.ID
	keepers []player.Keeper
	err     error
}

func (stub *playerKeeperReaderStub) ListKeepers(_ context.Context, playerID player.ID) ([]player.Keeper, error) {
	stub.gotID = playerID
	return stub.keepers, stub.err
}

func TestPlayerKeepersListReturnsAuthenticatedPlayerInventory(t *testing.T) {
	reader := &playerKeeperReaderStub{keepers: []player.Keeper{{
		PublicID:      "kpr_123",
		DefinitionKey: "harbor_warden",
		Name:          "Harbor Warden",
		CurrentName:   "Harbor Current",
		Sector:        "HARBOR",
		Role:          "WARDEN",
		Rarity:        "COMMON",
		Level:         1,
	}}}
	handler := NewPlayerKeepers(reader)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if reader.gotID != "internal-player-id" {
		t.Errorf("reader player ID = %q, want internal-player-id", reader.gotID)
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", response.Header().Get("Cache-Control"))
	}
	for _, want := range []string{
		`"id":"kpr_123"`,
		`"definitionKey":"harbor_warden"`,
		`"currentName":"Harbor Current"`,
		`"level":1`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body = %q, want %q", response.Body.String(), want)
		}
	}
}

func TestPlayerKeepersListReturnsEmptyArray(t *testing.T) {
	handler := NewPlayerKeepers(&playerKeeperReaderStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"keepers\":[]}\n" {
		t.Errorf("body = %q, want empty keepers array", response.Body.String())
	}
}

func TestPlayerKeepersListRejectsMissingPrincipal(t *testing.T) {
	handler := NewPlayerKeepers(&playerKeeperReaderStub{})
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/v1/me/keepers", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Errorf("body = %q, want AUTH_REQUIRED", response.Body.String())
	}
}

func TestPlayerKeepersListHidesReaderFailure(t *testing.T) {
	handler := NewPlayerKeepers(&playerKeeperReaderStub{err: errors.New("database unavailable")})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keepers", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "database unavailable") {
		t.Errorf("body leaked internal error: %q", response.Body.String())
	}
}
