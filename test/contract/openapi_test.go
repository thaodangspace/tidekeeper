package contract_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
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
		"/api/v1/me/keepers":                    "GET",
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

func loadDocument(t *testing.T) *openapi3.T {
	t.Helper()
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatalf("load OpenAPI: %v", err)
	}
	return document
}
