package contract_test

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/thaodangspace/tidekeepers-server/auth"
	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/httpapi"
	"github.com/thaodangspace/tidekeepers-server/httpapi/handlers"
	"github.com/thaodangspace/tidekeepers-server/player"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

func TestOpenAPIIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "api", "openapi.yaml")
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false

	document, err := loader.LoadFromFile(path)
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI: %v", err)
	}

	operations := map[string]string{
		"/auth/register":                        "POST",
		"/auth/login":                           "POST",
		"/api/v1/me":                            "GET",
		"/api/v1/me/keeper-unlocks":             "GET",
		"/api/v1/voyages":                       "POST",
		"/api/v1/voyages/current":               "GET",
		"/api/v1/voyages/current/keepers":       "GET",
		"/api/v1/voyages/{voyageId}":            "GET",
		"/api/v1/voyages/{voyageId}/abandon":    "POST",
		"/api/v1/voyages/{voyageId}/history":    "GET",
		"/api/v1/voyages/current/daily-context": "GET",
		"/health/live":                          "GET",
		"/health/ready":                         "GET",
	}
	for path, method := range operations {
		item := document.Paths.Find(path)
		if item == nil || item.GetOperation(method) == nil {
			t.Errorf("OpenAPI is missing %s %s", method, path)
		}
	}
}

func TestDailyPhaseRemainsExtensible(t *testing.T) {
	document := loadDocument(t)
	daily := document.Components.Schemas["DailySummary"]
	if daily == nil || daily.Value == nil {
		t.Fatal("DailySummary schema is missing")
	}
	phase := daily.Value.Properties["phase"]
	if phase == nil || phase.Value == nil {
		t.Fatal("DailySummary.phase schema is missing")
	}
	if len(phase.Value.Enum) != 0 {
		t.Fatalf("DailySummary.phase has a closed enum: %v", phase.Value.Enum)
	}
	if _, ok := phase.Value.Extensions["x-known-values"]; !ok {
		t.Fatal("DailySummary.phase does not document x-known-values")
	}
}

func TestRuntimeRoutesMatchOpenAPI(t *testing.T) {
	router := buildRouter(t)
	got := collectRoutes(router)

	expected := map[string]string{
		"/auth/register":                        "POST",
		"/auth/login":                           "POST",
		"/api/v1/me":                            "GET",
		"/api/v1/me/keeper-unlocks":             "GET",
		"/api/v1/voyages":                       "POST",
		"/api/v1/voyages/current":               "GET",
		"/api/v1/voyages/current/keepers":       "GET",
		"/api/v1/voyages/{voyageId}":            "GET",
		"/api/v1/voyages/{voyageId}/abandon":    "POST",
		"/api/v1/voyages/{voyageId}/history":    "GET",
		"/api/v1/voyages/current/daily-context": "GET",
		"/health/live":                          "GET",
		"/health/ready":                         "GET",
	}
	for route, method := range expected {
		if got[route] != method {
			t.Errorf("runtime route %s %s: missing or method mismatch (want %s)", method, route, method)
		}
	}
	for route, method := range got {
		if expected[route] != method {
			t.Errorf("unexpected runtime route %s %s", method, route)
		}
	}
}

func collectRoutes(router chi.Routes) map[string]string {
	routes := make(map[string]string)
	_ = chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes[route] = method
		return nil
	})
	return routes
}

func buildRouter(t *testing.T) chi.Router {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(discardWriter{}, nil))
	health := handlers.NewHealth(&nopPinger{}, 0, func() bool { return true })
	authHandler := handlers.NewAuth(&nopAuth{}, "sid", false, "")
	var authenticator auth.Authenticator = &nopAuth{}
	playerSvc := player.NewService(&nopPlayerReader{})
	me := handlers.NewMe(playerSvc)
	keeperUnlocks := handlers.NewKeeperUnlocks(playerSvc)
	dailySvc := daily.NewService(&nopDailyReader{})
	dailyCtx := handlers.NewDailyContextHandler(dailySvc)
	voyageSvc := &nopVoyageService{}
	voyageHandler := handlers.NewVoyageHandler(voyageSvc)
	voyageKeepers := handlers.NewVoyageKeepers(voyageSvc)

	return httpapi.NewFoundationRouter(
		health, authHandler, authenticator, me, keeperUnlocks, voyageKeepers, dailyCtx, voyageHandler,
		"sid", false, "", logger,
	).(*chi.Mux)
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

type nopPinger struct{}

func (n *nopPinger) Ping(context.Context) error { return nil }

type nopAuth struct{}

func (n *nopAuth) Register(_ context.Context, _, _ string) (auth.Session, error) {
	return auth.Session{}, nil
}
func (n *nopAuth) Login(_ context.Context, _, _ string) (auth.Session, error) {
	return auth.Session{}, nil
}
func (n *nopAuth) Authenticate(_ context.Context, _ auth.Digest) (auth.Principal, error) {
	return auth.Principal{}, nil
}

type nopPlayerReader struct{}

func (n *nopPlayerReader) GetMe(_ context.Context, _ player.ID) (player.Me, error) {
	return player.Me{}, nil
}
func (n *nopPlayerReader) ListUnlocks(_ context.Context, _ player.ID) ([]player.KeeperUnlock, error) {
	return nil, nil
}

type nopDailyReader struct{}

func (n *nopDailyReader) GetCurrentDailyContext(_ context.Context, _ daily.ID) (daily.DailyContext, error) {
	return daily.DailyContext{}, nil
}

type nopVoyageService struct{}

func (n *nopVoyageService) Create(_ context.Context, _, _ string) (*voyage.VoyageResponse, error) {
	return nil, nil
}
func (n *nopVoyageService) GetCurrent(_ context.Context, _ string) (*voyage.VoyageResponse, error) {
	return nil, nil
}
func (n *nopVoyageService) GetByID(_ context.Context, _, _ string) (*voyage.VoyageResponse, error) {
	return nil, nil
}
func (n *nopVoyageService) Abandon(_ context.Context, _, _, _ string) (*voyage.VoyageResponse, error) {
	return nil, nil
}
func (n *nopVoyageService) GetHistory(_ context.Context, _, _ string) (*voyage.VoyageHistoryResponse, error) {
	return nil, nil
}
func (n *nopVoyageService) GetActiveVoyageKeepers(_ context.Context, _ string) (*voyage.VoyageKeepers, error) {
	return nil, nil
}

func loadDocument(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}
	return document
}
