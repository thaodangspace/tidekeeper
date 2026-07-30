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

type playerReaderStub struct {
	me  player.Me
	err error
}

func (stub playerReaderStub) GetMe(context.Context, player.ID) (player.Me, error) {
	return stub.me, stub.err
}

func TestMeGetReturnsPublicPlayerIdentity(t *testing.T) {
	voyageID := "voy_123"
	handler := NewMe(playerReaderStub{me: player.Me{
		PublicID:            "plr_123",
		OnboardingCompleted: true,
		Locale:              "en",
		Timezone:            "UTC",
		ActiveVoyageID:      &voyageID,
	}})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", response.Header().Get("Cache-Control"))
	}
	for _, want := range []string{`"playerId":"plr_123"`, `"onboardingCompleted":true`, `"activeVoyageId":"voy_123"`} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("body = %q, want %q", response.Body.String(), want)
		}
	}
}

func TestMeGetHidesReaderFailure(t *testing.T) {
	handler := NewMe(playerReaderStub{err: errors.New("database unavailable")})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.Get(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "database unavailable") {
		t.Errorf("body leaked internal error: %q", response.Body.String())
	}
}
