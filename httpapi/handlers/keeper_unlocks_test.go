package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/player"
)

type keeperUnlockReaderStub struct {
	gotID   player.ID
	unlocks []player.KeeperUnlock
	err     error
}

func (stub *keeperUnlockReaderStub) ListUnlocks(_ context.Context, playerID player.ID) ([]player.KeeperUnlock, error) {
	stub.gotID = playerID
	return stub.unlocks, stub.err
}

func TestKeeperUnlocksListReturnsAuthenticatedUnlocks(t *testing.T) {
	unlockedAt := time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC)
	reader := &keeperUnlockReaderStub{unlocks: []player.KeeperUnlock{{
		DefinitionKey:     "crest_sovereign",
		DefinitionVersion: 1,
		Name:              "Crest Sovereign",
		CurrentName:       "Sovereign Current",
		Sector:            "CREST",
		Role:              "VANGUARD",
		Rarity:            "COMMON",
		UnlockSource:      "LEGACY_MIGRATION",
		UnlockedAt:        unlockedAt,
	}}}
	handler := NewKeeperUnlocks(reader)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keeper-unlocks", nil)
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
		`"definitionKey":"crest_sovereign"`,
		`"definitionVersion":1`,
		`"unlockSource":"LEGACY_MIGRATION"`,
		`"unlockedAt":"2026-01-15T08:30:00Z"`,
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body = %q, want %q", response.Body.String(), want)
		}
	}
}

func TestKeeperUnlocksListReturnsEmptyArray(t *testing.T) {
	handler := NewKeeperUnlocks(&keeperUnlockReaderStub{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keeper-unlocks", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.List(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "{\"unlocks\":[]}\n" {
		t.Errorf("body = %q, want empty unlocks array", response.Body.String())
	}
}

func TestKeeperUnlocksListRejectsMissingPrincipal(t *testing.T) {
	handler := NewKeeperUnlocks(&keeperUnlockReaderStub{})
	response := httptest.NewRecorder()

	handler.List(response, httptest.NewRequest(http.MethodGet, "/api/v1/me/keeper-unlocks", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Errorf("body = %q, want AUTH_REQUIRED", response.Body.String())
	}
}

func TestKeeperUnlocksListHidesReaderFailure(t *testing.T) {
	handler := NewKeeperUnlocks(&keeperUnlockReaderStub{err: errors.New("database unavailable")})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/keeper-unlocks", nil)
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
