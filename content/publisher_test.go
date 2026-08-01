package content

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thaodangspace/tidekeepers-server/database/query"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func TestCountReleaseRows(t *testing.T) {
	release := gameplayReleaseFixture()
	counts, err := countReleaseRows(release)
	if err != nil {
		t.Fatalf("countReleaseRows: %v", err)
	}
	if counts.BasketMappingCount != 10 || counts.KeeperDefinitionCount != 10 || counts.ComponentCount != 48 || counts.NodeCount != 30 {
		t.Fatalf("keeper counts = mappings:%d definitions:%d components:%d nodes:%d", counts.BasketMappingCount, counts.KeeperDefinitionCount, counts.ComponentCount, counts.NodeCount)
	}
	if counts.StrategyCount != 2 || counts.RelicCount != 1 || counts.SynergyCount != 1 || counts.ModifierCount != 1 || counts.ObjectiveCount != 1 || counts.GameRuleSetCount != 1 {
		t.Fatalf("gameplay counts = strategies:%d relics:%d synergies:%d modifiers:%d objectives:%d rules:%d",
			counts.StrategyCount, counts.RelicCount, counts.SynergyCount, counts.ModifierCount, counts.ObjectiveCount, counts.GameRuleSetCount)
	}
}

func TestCountReleaseRowsRejectsUnparsableUpgradeTree(t *testing.T) {
	release := gameplayReleaseFixture()
	release.Keeper.Definitions[0].UpgradeTree = json.RawMessage(`{`)
	if _, err := countReleaseRows(release); err == nil {
		t.Fatal("countReleaseRows(bad upgrade tree) succeeded")
	}
}

func TestCountsDiffer(t *testing.T) {
	base := releaseRowCounts{
		BasketMappingCount: 10, KeeperDefinitionCount: 10, ComponentCount: 48, NodeCount: 30,
		StrategyCount: 2, RelicCount: 1, SynergyCount: 1, ModifierCount: 1, ObjectiveCount: 1, GameRuleSetCount: 1,
	}
	if countsDiffer(asPersistedCounts(base), base) {
		t.Fatal("countsDiffer(equal counts) = true")
	}
	fields := []string{"BasketMappingCount", "KeeperDefinitionCount", "ComponentCount", "NodeCount",
		"StrategyCount", "RelicCount", "SynergyCount", "ModifierCount", "ObjectiveCount", "GameRuleSetCount"}
	for _, field := range fields {
		mutated := base
		switch field {
		case "BasketMappingCount":
			mutated.BasketMappingCount++
		case "KeeperDefinitionCount":
			mutated.KeeperDefinitionCount++
		case "ComponentCount":
			mutated.ComponentCount++
		case "NodeCount":
			mutated.NodeCount++
		case "StrategyCount":
			mutated.StrategyCount++
		case "RelicCount":
			mutated.RelicCount++
		case "SynergyCount":
			mutated.SynergyCount++
		case "ModifierCount":
			mutated.ModifierCount++
		case "ObjectiveCount":
			mutated.ObjectiveCount++
		case "GameRuleSetCount":
			mutated.GameRuleSetCount++
		}
		if !countsDiffer(asPersistedCounts(mutated), base) {
			t.Fatalf("countsDiffer(different %s) = false", field)
		}
	}
}

func TestReleaseLockScope(t *testing.T) {
	if got := releaseLockScope(3); got != "content_release:3" {
		t.Fatalf("releaseLockScope(3) = %q", got)
	}
}

func TestASPGTypeUUID(t *testing.T) {
	id := uuid.New()
	converted := asPGTypeUUID(id)
	if !converted.Valid || converted.Bytes != [16]byte(id) {
		t.Fatalf("asPGTypeUUID = %v", converted)
	}
}

func TestPublishRejectsInvalidReleaseBeforeTransaction(t *testing.T) {
	release := validBaselineRelease()
	release.Modifiers[0].RuleConfig = json.RawMessage(`[]`)
	_, err := NewPublisher(nil).Publish(context.Background(), release)
	if err == nil {
		t.Fatal("Publish(invalid release) succeeded")
	}
	var validationError *ValidationError
	if !errors.As(err, &validationError) {
		t.Fatalf("Publish(invalid release) error = %v, want ValidationError", err)
	}
}

func TestCatalogWriterNodesMatchFixture(t *testing.T) {
	nodes, err := keeper.CountUpgradeNodes(keeper.CatalogV1())
	if err != nil {
		t.Fatalf("CountUpgradeNodes(CatalogV1): %v", err)
	}
	if nodes != 30 {
		t.Fatalf("CountUpgradeNodes(CatalogV1) = %d, want 30", nodes)
	}
	v2Nodes, err := keeper.CountUpgradeNodes(keeper.CatalogV2())
	if err != nil {
		t.Fatalf("CountUpgradeNodes(CatalogV2): %v", err)
	}
	if v2Nodes != 39 {
		t.Fatalf("CountUpgradeNodes(CatalogV2) = %d, want 39", v2Nodes)
	}
}

func TestFixedPointUnits(t *testing.T) {
	cases := []struct {
		name  string
		numer pgtype.Numeric
		scale int64
		want  int64
	}{
		{name: "as stored", numer: pgtype.Numeric{Int: big.NewInt(125000000), Exp: -8, Valid: true}, scale: keeper.WeightScale, want: 125000000},
		{name: "whole numeric", numer: pgtype.Numeric{Int: big.NewInt(50000000), Exp: 0, Valid: true}, scale: keeper.WeightScale, want: 5000000000000000},
		{name: "zero", numer: pgtype.Numeric{Int: big.NewInt(0), Exp: 0, Valid: true}, scale: keeper.WeightScale, want: 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := fixedPointUnits(testCase.numer, testCase.scale)
			if err != nil {
				t.Fatalf("fixedPointUnits: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("fixedPointUnits = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestFixedPointUnitsRejectsFractional(t *testing.T) {
	fractional := pgtype.Numeric{Int: big.NewInt(1), Exp: -9, Valid: true}
	if _, err := fixedPointUnits(fractional, keeper.WeightScale); err == nil {
		t.Fatal("fixedPointUnits(fractional) succeeded")
	}
	if _, err := fixedPointUnits(pgtype.Numeric{Valid: false}, keeper.WeightScale); err == nil {
		t.Fatal("fixedPointUnits(null) succeeded")
	}
}

// asPersistedCounts converts a releaseRowCounts to the persisted query row
// shape so countsDiffer can be exercised without a database.
func asPersistedCounts(counts releaseRowCounts) query.CountContentReleaseRowsRow {
	return query.CountContentReleaseRowsRow{
		BasketMappingCount:    counts.BasketMappingCount,
		KeeperDefinitionCount: counts.KeeperDefinitionCount,
		ComponentCount:        counts.ComponentCount,
		NodeCount:             counts.NodeCount,
		StrategyCount:         counts.StrategyCount,
		RelicCount:            counts.RelicCount,
		SynergyCount:          counts.SynergyCount,
		ModifierCount:         counts.ModifierCount,
		ObjectiveCount:        counts.ObjectiveCount,
		GameRuleSetCount:      counts.GameRuleSetCount,
	}
}
