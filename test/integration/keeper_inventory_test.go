//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/player"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

func checksum32(seed byte) []byte {
	return bytes.Repeat([]byte{seed}, 32)
}

func timeMustParse(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func TestPlayerListUnlocks(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	conn, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "inv_unlocks")
	defer cleanup()

	mustExec(t, ctx, conn,
		"INSERT INTO content_releases (version, status, checksum) VALUES (2, 'DRAFT', $1)",
		checksum32(0x02),
	)
	mustExec(t, ctx, conn,
		"INSERT INTO basket_mapping_versions (id, mapping_key, content_version) VALUES ($1, 'harbor_focus', 2)",
		"00000000-0000-0000-0000-000000000012",
	)
	mustExec(t, ctx, conn,
		"INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight) VALUES ($1, $2, 0.50000000), ($1, $3, 0.50000000)",
		"00000000-0000-0000-0000-000000000012", assetOneID, assetTwoID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'ocean_titan', 2, 'Ocean Titan', 'titan_current', 'Titan Current',
			'HARBOR', 'WARDEN', 'RARE', 'MEDIUM', 'TEST_RULE', '{}'::jsonb, '{}'::jsonb, $2)`,
		"00000000-0000-0000-0000-000000000022", "00000000-0000-0000-0000-000000000012",
	)
	mustExec(t, ctx, conn, `
		INSERT INTO player_keeper_unlocks (
			player_id, keeper_key, unlocked_definition_version_id, unlock_source, unlocked_at
		) VALUES ($1, 'ocean_titan', $2, 'STARTER', '2026-01-03T00:00:00Z'),
		         ($1, 'crest_sovereign', $3, 'SHOP', '2026-01-05T00:00:00Z')
	`, voyagePlrOneID, "00000000-0000-0000-0000-000000000022", keeperOneID)

	service := player.NewService(player.NewRepository(pool))
	unlocks, err := service.ListUnlocks(ctx, player.ID(voyagePlrOneID))
	if err != nil {
		t.Fatalf("ListUnlocks: %v", err)
	}
	if len(unlocks) != 2 {
		t.Fatalf("unlock count = %d, want 2", len(unlocks))
	}

	first := unlocks[0]
	if first.DefinitionKey != "ocean_titan" {
		t.Errorf("first definition key = %q, want ocean_titan (ordered by unlocked_at)", first.DefinitionKey)
	}
	if first.DefinitionVersion != 2 {
		t.Errorf("first definition version = %d, want 2", first.DefinitionVersion)
	}
	if first.Name != "Ocean Titan" || first.CurrentName != "Titan Current" {
		t.Errorf("first names = %q/%q", first.Name, first.CurrentName)
	}
	if first.Sector != "HARBOR" || first.Role != "WARDEN" || first.Rarity != "RARE" {
		t.Errorf("first metadata = %q/%q/%q, want HARBOR/WARDEN/RARE", first.Sector, first.Role, first.Rarity)
	}
	if first.UnlockSource != "STARTER" {
		t.Errorf("first unlock source = %q, want STARTER", first.UnlockSource)
	}
	if !first.UnlockedAt.Equal(timeMustParse(t, "2026-01-03T00:00:00Z")) {
		t.Errorf("first unlocked at = %v, want 2026-01-03T00:00:00Z", first.UnlockedAt)
	}

	second := unlocks[1]
	if second.DefinitionKey != "crest_sovereign" {
		t.Errorf("second definition key = %q, want crest_sovereign", second.DefinitionKey)
	}
	if second.UnlockSource != "SHOP" {
		t.Errorf("second unlock source = %q, want SHOP", second.UnlockSource)
	}
}

func TestPlayerListUnlocksReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "inv_unlocks_empty")
	defer cleanup()

	service := player.NewService(player.NewRepository(pool))
	unlocks, err := service.ListUnlocks(ctx, player.ID(voyagePlrOneID))
	if err != nil {
		t.Fatalf("ListUnlocks: %v", err)
	}
	if len(unlocks) != 0 {
		t.Errorf("unlock count = %d, want 0", len(unlocks))
	}
}

func TestGetActiveVoyageKeepersNoActiveVoyage(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "inv_no_voyage")
	defer cleanup()

	service := voyage.NewService(pool)
	_, err := service.GetActiveVoyageKeepers(ctx, voyagePlrTwoID)
	if !errors.Is(err, voyage.ErrNoActiveVoyage) {
		t.Fatalf("error = %v, want ErrNoActiveVoyage", err)
	}
}
