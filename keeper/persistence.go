package keeper

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// CatalogWriter persists the Keeper catalog children of one release through a
// transaction-scoped query set. Both the legacy Keeper publisher and the
// generalized gameplay-content publisher reuse this writer so child persistence
// never diverges.
type CatalogWriter struct {
	queries *query.Queries
}

// NewCatalogWriter returns a Keeper catalog writer bound to the supplied
// transaction-scoped queries.
func NewCatalogWriter(queries *query.Queries) *CatalogWriter {
	return &CatalogWriter{queries: queries}
}

// Write persists every Keeper catalog child row for release. The owning
// content_releases row must already exist in DRAFT state; the caller owns
// publication and counting. Asset rows are shared across releases and are
// reused by key with a symbol-conflict check.
func (w *CatalogWriter) Write(ctx context.Context, release CatalogRelease) error {
	assetIDs := make(map[string]pgtype.UUID, len(release.Assets))
	for _, asset := range release.Assets {
		storedAsset, err := w.queries.GetMarketAssetByKey(ctx, asset.Key)
		switch {
		case err == nil:
			if storedAsset.Symbol != asset.Symbol {
				return fmt.Errorf("%w: asset %q has symbol %q, not %q", ErrReleaseConflict, asset.Key, storedAsset.Symbol, asset.Symbol)
			}
			assetIDs[asset.Key] = storedAsset.ID
		case errors.Is(err, pgx.ErrNoRows):
			assetID, idErr := randomUUID()
			if idErr != nil {
				return idErr
			}
			if insertErr := w.queries.InsertMarketAsset(ctx, query.InsertMarketAssetParams{
				ID: assetID, AssetKey: asset.Key, Symbol: asset.Symbol,
			}); insertErr != nil {
				return fmt.Errorf("insert market asset %q: %w", asset.Key, insertErr)
			}
			assetIDs[asset.Key] = assetID
		default:
			return fmt.Errorf("load market asset %q: %w", asset.Key, err)
		}
	}

	sectorDefinitionIDs := make(map[string]pgtype.UUID, len(release.SectorDefinitions))
	for _, definition := range release.SectorDefinitions {
		definitionID, err := randomUUID()
		if err != nil {
			return err
		}
		if err := w.queries.InsertSectorDefinitionVersion(ctx, query.InsertSectorDefinitionVersionParams{
			ID: definitionID, SectorKey: definition.Sector, ContentVersion: release.Version,
			BenchmarkMethod: definition.BenchmarkMethod, MinimumEligibleBaskets: int32(definition.MinimumEligibleBaskets),
			RelativeScaleUnits: definition.RelativeScale, RelativeBlendWeightUnits: definition.RelativeBlendWeight,
			RankBlendWeightUnits: definition.RankBlendWeight, ScoreCapUnits: definition.ScoreCap,
		}); err != nil {
			return fmt.Errorf("insert sector definition %q: %w", definition.Sector, err)
		}
		sectorDefinitionIDs[definition.Sector] = definitionID
	}

	turbulencePolicyIDs := make(map[string]pgtype.UUID, len(release.TurbulencePolicies))
	for _, policy := range release.TurbulencePolicies {
		policyID, err := randomUUID()
		if err != nil {
			return err
		}
		if err := w.queries.InsertExpectedTurbulencePolicy(ctx, query.InsertExpectedTurbulencePolicyParams{
			ID: policyID, PolicyKey: policy.Key, ContentVersion: release.Version, PolicyType: policy.Type,
			StaticValueUnits: int64Pointer(policy.StaticValue), FloorUnits: policy.Floor, RoundingMode: policy.RoundingMode,
		}); err != nil {
			return fmt.Errorf("insert expected turbulence policy %q: %w", policy.Key, err)
		}
		turbulencePolicyIDs[policy.Key] = policyID
	}

	mappingIDs := make(map[string]pgtype.UUID, len(release.Baskets))
	for _, basket := range release.Baskets {
		mappingID, err := randomUUID()
		if err != nil {
			return err
		}
		if err := w.queries.InsertBasketMappingVersion(ctx, query.InsertBasketMappingVersionParams{
			ID: mappingID, MappingKey: basket.Key, ContentVersion: release.Version,
			SectorDefinitionVersionID:  sectorDefinitionIDs[basket.Sector],
			MinimumCoveredWeightUnits:  int64Pointer(basket.MinimumCoveredWeight),
			ExpectedTurbulencePolicyID: turbulencePolicyIDs[basket.ExpectedTurbulencePolicyKey],
			NormalizationCapUnits:      int64Pointer(basket.NormalizationCap),
			BenchmarkEligible:          boolPointer(basket.BenchmarkEligible, basket.Sector != ""),
		}); err != nil {
			return fmt.Errorf("insert basket mapping %q: %w", basket.Key, err)
		}
		mappingIDs[basket.Key] = mappingID
		for _, component := range basket.Components {
			if err := w.queries.InsertBasketMappingComponent(ctx, query.InsertBasketMappingComponentParams{
				BasketMappingVersionID: mappingID,
				MarketAssetID:          assetIDs[component.AssetKey],
				Weight:                 fixedWeight(component.Weight),
			}); err != nil {
				return fmt.Errorf("insert basket component %q/%q: %w", basket.Key, component.AssetKey, err)
			}
		}
	}

	for _, definition := range release.Definitions {
		definitionID, err := randomUUID()
		if err != nil {
			return err
		}
		if insertErr := w.queries.InsertKeeperDefinitionVersion(ctx, query.InsertKeeperDefinitionVersionParams{
			ID:                     definitionID,
			KeeperKey:              definition.Key,
			ContentVersion:         release.Version,
			Name:                   definition.Name,
			CurrentKey:             definition.CurrentKey,
			CurrentName:            definition.CurrentName,
			SectorKey:              definition.Sector,
			RoleKey:                definition.Role,
			RarityKey:              definition.Rarity,
			BaseRiskKey:            definition.BaseRisk,
			ExpectedTurbulenceBps:  int32Pointer(definition.ExpectedTurbulenceBPS),
			PassiveRuleKey:         definition.PassiveRuleKey,
			PassiveRuleConfig:      definition.PassiveRuleConfig,
			UpgradeTree:            definition.UpgradeTree,
			BasketMappingVersionID: mappingIDs[definition.BasketMappingKey],
		}); insertErr != nil {
			return fmt.Errorf("insert Keeper definition %q: %w", definition.Key, insertErr)
		}
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			return fmt.Errorf("parse Keeper %q upgrade tree: %w", definition.Key, err)
		}
		for _, node := range nodes {
			if insertErr := w.queries.InsertKeeperDefinitionUpgradeNode(ctx, query.InsertKeeperDefinitionUpgradeNodeParams{
				KeeperDefinitionVersionID: definitionID,
				NodeKey:                   node.Key,
				Depth:                     node.Depth,
				IsRoot:                    node.IsRoot,
			}); insertErr != nil {
				return fmt.Errorf("insert Keeper %q upgrade node %q: %w", definition.Key, node.Key, insertErr)
			}
		}
	}
	return nil
}

// CountUpgradeNodes returns the total number of upgrade-node rows a validated
// catalog release will persist.
func CountUpgradeNodes(release CatalogRelease) (int64, error) {
	var total int64
	for _, definition := range release.Definitions {
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			return 0, fmt.Errorf("parse Keeper %q upgrade tree: %w", definition.Key, err)
		}
		total += int64(len(nodes))
	}
	return total, nil
}

func fixedWeight(weight int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(weight), Exp: -8, Valid: true}
}

func int32Pointer(value *int) *int32 {
	if value == nil {
		return nil
	}
	converted := int32(*value)
	return &converted
}

func int64Pointer(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func boolPointer(value bool, valid bool) *bool {
	if !valid {
		return nil
	}
	return &value
}

func randomUUID() (pgtype.UUID, error) {
	var value pgtype.UUID
	if _, err := rand.Read(value.Bytes[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("generate catalog UUID: %w", err)
	}
	value.Bytes[6] = (value.Bytes[6] & 0x0f) | 0x40
	value.Bytes[8] = (value.Bytes[8] & 0x3f) | 0x80
	value.Valid = true
	return value, nil
}
