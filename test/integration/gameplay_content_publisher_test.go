//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/content"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func TestGameplayContentPublisherIsIdempotentAndConflictSafe(t *testing.T) {
	ctx := context.Background()
	pool := gameplayContentTestPool(t, ctx, "gameplay_content_publisher")

	publisher := content.NewPublisher(pool)
	first, err := publisher.Publish(ctx, validBaselineGameplayRelease())
	if err != nil {
		t.Fatalf("first Publish(): %v", err)
	}
	if first.Idempotent || first.KeeperDefinitionCount != 10 || first.KeeperMappingCount != 10 || first.ComponentCount != 48 || first.NodeCount != 30 {
		t.Fatalf("first Publish() result = %#v", first)
	}
	if first.StrategyCount != 0 || first.RelicCount != 0 || first.SynergyCount != 0 ||
		first.ModifierCount != 1 || first.ObjectiveCount != 1 || first.GameRuleSetCount != 1 {
		t.Fatalf("first Publish() gameplay counts = %#v", first)
	}

	second, err := publisher.Publish(ctx, validBaselineGameplayRelease())
	if err != nil {
		t.Fatalf("second Publish(): %v", err)
	}
	if !second.Idempotent || second.Checksum != first.Checksum {
		t.Fatalf("second Publish() result = %#v", second)
	}

	conflicting := validBaselineGameplayRelease()
	conflicting.Modifiers[0].Name = "Changed Conditions"
	if _, err := publisher.Publish(ctx, conflicting); !errors.Is(err, content.ErrReleaseConflict) {
		t.Fatalf("Publish(conflicting) error = %v, want ErrReleaseConflict", err)
	}

	var definitionVersions, strategies, relics, synergies, modifiers, objectives, ruleSets int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM keeper_definition_versions),
			(SELECT count(*) FROM strategy_definition_versions),
			(SELECT count(*) FROM relic_definition_versions),
			(SELECT count(*) FROM synergy_definition_versions),
			(SELECT count(*) FROM daily_modifier_definition_versions),
			(SELECT count(*) FROM daily_objective_definition_versions),
			(SELECT count(*) FROM game_rule_set_versions)
	`).Scan(&definitionVersions, &strategies, &relics, &synergies, &modifiers, &objectives, &ruleSets); err != nil {
		t.Fatalf("count persisted gameplay rows: %v", err)
	}
	if definitionVersions != 10 || strategies != 0 || relics != 0 || synergies != 0 || modifiers != 1 || objectives != 1 || ruleSets != 1 {
		t.Fatalf("persisted counts = keepers:%d strategies:%d relics:%d synergies:%d modifiers:%d objectives:%d rules:%d",
			definitionVersions, strategies, relics, synergies, modifiers, objectives, ruleSets)
	}
}

func TestGameplayContentRepositoryRoundTripsRelease(t *testing.T) {
	ctx := context.Background()
	pool := gameplayContentTestPool(t, ctx, "gameplay_content_repository")

	publisher := content.NewPublisher(pool)
	original, err := publisher.Publish(ctx, validBaselineGameplayRelease())
	if err != nil {
		t.Fatalf("Publish(): %v", err)
	}

	repository := content.NewRepository(pool)
	loaded, err := repository.LoadRelease(ctx, 1)
	if err != nil {
		t.Fatalf("LoadRelease: %v", err)
	}
	loadedChecksum, err := content.Checksum(loaded)
	if err != nil {
		t.Fatalf("Checksum(loaded): %v", err)
	}
	if !bytes.Equal(loadedChecksum[:], original.Checksum[:]) {
		t.Fatalf("round-trip checksum = %x, want %x", loadedChecksum, original.Checksum)
	}

	var modifierID pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT id FROM daily_modifier_definition_versions WHERE modifier_key = 'mod_default' AND content_version = 1",
	).Scan(&modifierID); err != nil {
		t.Fatalf("load modifier version id: %v", err)
	}
	modifier, err := repository.GetModifier(ctx, modifierID)
	if err != nil {
		t.Fatalf("GetModifier: %v", err)
	}
	if modifier.Key != "mod_default" || modifier.RuleKey != content.RuleStandardConditions {
		t.Fatalf("GetModifier = %#v", modifier)
	}

	var objectiveID pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT id FROM daily_objective_definition_versions WHERE objective_key = 'obj_default' AND content_version = 1",
	).Scan(&objectiveID); err != nil {
		t.Fatalf("load objective version id: %v", err)
	}
	objective, err := repository.GetObjective(ctx, objectiveID)
	if err != nil {
		t.Fatalf("GetObjective: %v", err)
	}
	if objective.Key != "obj_default" || objective.RuleKey != content.RuleNavigate {
		t.Fatalf("GetObjective = %#v", objective)
	}

	var ruleSetID pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT id FROM game_rule_set_versions WHERE rule_set_key = 'rules_standard' AND content_version = 1",
	).Scan(&ruleSetID); err != nil {
		t.Fatalf("load game rule set version id: %v", err)
	}
	ruleSet, err := repository.GetGameRuleSet(ctx, ruleSetID)
	if err != nil {
		t.Fatalf("GetGameRuleSet: %v", err)
	}
	if ruleSet.Key != "rules_standard" || ruleSet.RuleKey != content.RuleStandardRules {
		t.Fatalf("GetGameRuleSet = %#v", ruleSet)
	}
}

func TestGameplayContentPublisherRejectsSchemaOneRelease(t *testing.T) {
	ctx := context.Background()
	pool := gameplayContentTestPool(t, ctx, "gameplay_content_schema_one")

	keeperPublisher := keeper.NewPublisher(pool)
	if _, err := keeperPublisher.Publish(ctx, keeper.CatalogV1()); err != nil {
		t.Fatalf("keeper Publish(CatalogV1): %v", err)
	}

	publisher := content.NewPublisher(pool)
	if _, err := publisher.Publish(ctx, validBaselineGameplayRelease()); !errors.Is(err, content.ErrReleaseConflict) {
		t.Fatalf("Publish(schema-one version) error = %v, want ErrReleaseConflict", err)
	}
}

func validBaselineGameplayRelease() content.Release {
	return content.Release{
		Version: 1,
		Keeper:  keeper.CatalogV1(),
		Modifiers: []content.Modifier{{
			Key: "mod_default", Name: "Standard Conditions", Description: "Default voyage conditions.",
			RuleKey: content.RuleStandardConditions, RuleConfig: gameplayRawConfig(`{}`),
		}},
		Objectives: []content.Objective{{
			Key: "obj_default", Name: "Navigate", Description: "Complete the daily voyage.",
			RuleKey: content.RuleNavigate, RuleConfig: gameplayRawConfig(`{}`),
		}},
		GameRuleSets: []content.GameRuleSet{{
			Key: "rules_standard", Name: "Standard Rules", Description: "Standard game rules.",
			RuleKey: content.RuleStandardRules, RuleConfig: gameplayRawConfig(`{}`),
		}},
	}
}

func gameplayRawConfig(parameters string) []byte {
	return []byte(`{"schemaVersion":1,"parameters":` + parameters + `}`)
}

func gameplayContentTestPool(t *testing.T, ctx context.Context, prefix string) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(func() { admin.Close(ctx) })

	schema := prefix + "_" + randomHex(t)
	mustExec(t, ctx, admin, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, admin, "SET search_path TO "+schema)
	for _, migration := range []string{
		"000001_foundation.up.sql",
		"000002_keepers_catalog.up.sql",
		"000005_four_sector_calculation.up.sql",
		"000007_keeper_definition_upgrade_nodes.up.sql",
		"000009_gameplay_content_foundation.up.sql",
	} {
		applyMigration(t, ctx, admin, migration)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	poolConfig.MinConns = 0
	poolConfig.MaxConns = 2
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open content pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}
