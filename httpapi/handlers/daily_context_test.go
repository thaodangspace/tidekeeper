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
	"github.com/thaodangspace/tidekeepers-server/daily"
)

type dailyContextReaderStub struct {
	ctx daily.DailyContext
	err error
}

func (stub *dailyContextReaderStub) GetCurrentDailyContext(_ context.Context, _ daily.ID) (daily.DailyContext, error) {
	return stub.ctx, stub.err
}

func TestDailyContextGetReturnsFullResponse(t *testing.T) {
	now := time.Date(2026, 7, 30, 4, 0, 0, 0, time.UTC)
	lockAt := time.Date(2026, 7, 30, 23, 55, 0, 0, time.UTC)
	settleAfter := time.Date(2026, 7, 31, 0, 5, 0, 0, time.UTC)

	reader := &dailyContextReaderStub{ctx: daily.DailyContext{
		ServerNow: now,
		Voyage: daily.VoyageSummary{
			PublicID:      "voy_123",
			Status:        "ACTIVE",
			DayNumber:     3,
			FundHealth:    82,
			MaxFundHealth: 100,
			Capital:       11,
			Score:         "245.0000",
		},
		Daily: daily.DailyDetail{
			Phase:              "PREPARATION",
			LockAt:             lockAt,
			SettleAfter:        settleAfter,
			Version:            9,
			SelectedStrategyID: strPtr("strategy_steady"),
			PendingRewardCount: 0,
			Modifier: daily.Modifier{
				ID: "mod_001", Name: "High Tide", Description: "All gains 1.5x",
			},
			Objective: daily.Objective{
				ID: "obj_001", Name: "Growth", Description: "Grow portfolio",
				ProgressLabel: strPtr("3/5%"), RewardLabel: strPtr("+1 Health"),
			},
			Signals: []daily.Signal{{
				ID: "sig_001", Name: "BTC Up", Description: "Bitcoin rising",
				Direction: "UP", Strength: "STRONG",
				ObservedFrom: now, ObservedTo: now,
			}},
			Lineup: daily.Lineup{
				MaxSlots: 3,
				Slots: []daily.FleetSlot{
					{Index: 0, Keeper: nil},
					{Index: 1, Keeper: &daily.Keeper{ID: "kpr_001", DefinitionID: "def_btc", Name: "Trader", Level: 5, Rarity: "RARE", Role: "TRADER", Sector: "CRYPTO", PassiveSummary: "Test"}},
					{Index: 2, Keeper: nil},
				},
				Synergies: []daily.Synergy{{
					ID: "syn_001", Name: "BTC Focus", Description: "BTC bonus", State: "ACTIVE", CurrentCount: 1, RequiredCount: 2,
				}},
				Warnings: []daily.FleetWarning{{
					Code: "EMPTY_SLOT", Message: "Slot 0 is empty", Severity: "WARNING",
				}},
			},
			Inventory: []daily.Keeper{{
				ID: "kpr_002", DefinitionID: "def_eth", Name: "Holder", Level: 3,
				Rarity: "COMMON", Role: "HOLDER", Sector: "CRYPTO", PassiveSummary: "HODL",
			}},
			Shop: daily.Shop{
				Offers: []daily.ShopOffer{{
					ID: "off_001", Cost: 5, Available: true,
					Keeper: daily.Keeper{ID: "kpr_003", DefinitionID: "def_sol", Name: "Sol Trader", Level: 2, Rarity: "UNCOMMON", Role: "TRADER", Sector: "CRYPTO", PassiveSummary: "Fast"},
				}},
				RefreshAt: nil, RerollCost: 3, RerollIndex: 0,
			},
			Strategies: []daily.Strategy{{
				ID: "strategy_steady", Name: "Steady", Description: "Safe", Upside: "Low", Downside: "Minimal", Available: true,
			}},
		},
	}}

	handler := NewDailyContextHandler(reader)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "internal-player-id"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Errorf("Cache-Control = %q, want private, no-store", response.Header().Get("Cache-Control"))
	}

	body := response.Body.String()
	for _, want := range []string{
		`"serverNow"`,
		`"voyage"`,
		`"daily"`,
		`"id":"voy_123"`,
		`"status":"ACTIVE"`,
		`"dayNumber":3`,
		`"score":"245.0000"`,
		`"phase":"PREPARATION"`,
		`"version":9`,
		`"selectedStrategyId":"strategy_steady"`,
		`"pendingRewardCount":0`,
		`"modifier"`,
		`"objective"`,
		`"lineup"`,
		`"signals"`,
		`"inventory"`,
		`"shop"`,
		`"strategies"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}

func TestDailyContextGetRejectsMissingPrincipal(t *testing.T) {
	handler := NewDailyContextHandler(&dailyContextReaderStub{})
	response := httptest.NewRecorder()

	handler.GetCurrent(response, httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if !strings.Contains(response.Body.String(), `"code":"AUTH_REQUIRED"`) {
		t.Errorf("body = %q, want AUTH_REQUIRED", response.Body.String())
	}
}

func TestDailyContextGetReturnsNoActiveVoyage(t *testing.T) {
	handler := NewDailyContextHandler(&dailyContextReaderStub{err: daily.ErrNoActiveVoyage})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "pid"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if !strings.Contains(response.Body.String(), `"code":"NO_ACTIVE_VOYAGE"`) {
		t.Errorf("body = %q, want NO_ACTIVE_VOYAGE", response.Body.String())
	}
}

func TestDailyContextGetReturnsVoyageNotFound(t *testing.T) {
	handler := NewDailyContextHandler(&dailyContextReaderStub{err: daily.ErrVoyageNotFound})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "pid"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if !strings.Contains(response.Body.String(), `"code":"VOYAGE_NOT_FOUND"`) {
		t.Errorf("body = %q, want VOYAGE_NOT_FOUND", response.Body.String())
	}
}

func TestDailyContextGetReturnsServiceUnavailable(t *testing.T) {
	handler := NewDailyContextHandler(&dailyContextReaderStub{err: daily.ErrServiceUnavailable})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "pid"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(response.Body.String(), `"code":"SERVICE_UNAVAILABLE"`) {
		t.Errorf("body = %q, want SERVICE_UNAVAILABLE", response.Body.String())
	}
}

func TestDailyContextGetHidesInternalError(t *testing.T) {
	handler := NewDailyContextHandler(&dailyContextReaderStub{err: errors.New("something sensitive")})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "pid"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "sensitive") {
		t.Errorf("body leaked internal error: %q", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Errorf("body = %q, want INTERNAL_ERROR", response.Body.String())
	}
}

func TestDailyContextGetUsesReturnedPhaseForTerminalVoyage(t *testing.T) {
	now := time.Date(2026, 7, 30, 4, 0, 0, 0, time.UTC)

	reader := &dailyContextReaderStub{ctx: daily.DailyContext{
		ServerNow: now,
		Voyage: daily.VoyageSummary{
			PublicID: "voy_999", Status: "COMPLETED", DayNumber: 7,
			FundHealth: 50, MaxFundHealth: 100, Capital: 5, Score: "500.0000",
		},
		Daily: daily.DailyDetail{
			Phase: "RESULT_READY", Version: 10,
		},
	}}

	handler := NewDailyContextHandler(reader)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/voyages/current/daily-context", nil)
	request = request.WithContext(auth.WithPrincipal(request.Context(), auth.Principal{PlayerID: "pid"}))
	response := httptest.NewRecorder()

	handler.GetCurrent(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"phase":"RESULT_READY"`) {
		t.Errorf("body = %q, want phase RESULT_READY", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"status":"COMPLETED"`) {
		t.Errorf("body = %q, want status COMPLETED", response.Body.String())
	}
}

func strPtr(s string) *string {
	return &s
}
