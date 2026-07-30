package keeper

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// ErrReleaseConflict means the requested release version already identifies
// different published or draft content.
var ErrReleaseConflict = errors.New("content release conflict")

// PublishResult describes a completed catalog publication attempt.
type PublishResult struct {
	Version         int64
	Checksum        [32]byte
	DefinitionCount int64
	MappingCount    int64
	ComponentCount  int64
	Idempotent      bool
}

// Publisher transactionally persists validated Keeper catalog content.
type Publisher struct {
	pool *pgxpool.Pool
}

// NewPublisher creates a catalog publisher backed by PostgreSQL.
func NewPublisher(pool *pgxpool.Pool) *Publisher {
	return &Publisher{pool: pool}
}

// Publish validates and atomically publishes a catalog release. Re-publishing
// the same version/checksum returns the persisted result without writing rows.
func (p *Publisher) Publish(ctx context.Context, release CatalogRelease) (PublishResult, error) {
	if err := Validate(release); err != nil {
		return PublishResult{}, fmt.Errorf("validate catalog: %w", err)
	}
	checksum, err := Checksum(release)
	if err != nil {
		return PublishResult{}, err
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PublishResult{}, fmt.Errorf("begin catalog publication: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := query.New(tx)

	storedRelease, err := queries.GetContentRelease(ctx, release.Version)
	switch {
	case err == nil:
		if storedRelease.Status != "PUBLISHED" || !bytes.Equal(storedRelease.Checksum, checksum[:]) {
			return PublishResult{}, fmt.Errorf("%w: version %d has different or incomplete content", ErrReleaseConflict, release.Version)
		}
		counts, err := queries.CountCatalogReleaseRows(ctx, release.Version)
		if err != nil {
			return PublishResult{}, fmt.Errorf("count existing catalog release: %w", err)
		}
		return PublishResult{
			Version:         release.Version,
			Checksum:        checksum,
			DefinitionCount: counts.DefinitionCount,
			MappingCount:    counts.MappingCount,
			ComponentCount:  counts.ComponentCount,
			Idempotent:      true,
		}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PublishResult{}, fmt.Errorf("load catalog release: %w", err)
	}

	if err := queries.CreateDraftContentRelease(ctx, query.CreateDraftContentReleaseParams{
		Version: release.Version, Checksum: checksum[:],
	}); err != nil {
		return PublishResult{}, fmt.Errorf("create catalog release: %w", err)
	}

	assetIDs := make(map[string]pgtype.UUID, len(release.Assets))
	for _, asset := range release.Assets {
		storedAsset, err := queries.GetMarketAssetByKey(ctx, asset.Key)
		switch {
		case err == nil:
			if storedAsset.Symbol != asset.Symbol {
				return PublishResult{}, fmt.Errorf("%w: asset %q has symbol %q, not %q", ErrReleaseConflict, asset.Key, storedAsset.Symbol, asset.Symbol)
			}
			assetIDs[asset.Key] = storedAsset.ID
		case errors.Is(err, pgx.ErrNoRows):
			assetID, err := randomUUID()
			if err != nil {
				return PublishResult{}, err
			}
			if err := queries.InsertMarketAsset(ctx, query.InsertMarketAssetParams{
				ID: assetID, AssetKey: asset.Key, Symbol: asset.Symbol,
			}); err != nil {
				return PublishResult{}, fmt.Errorf("insert market asset %q: %w", asset.Key, err)
			}
			assetIDs[asset.Key] = assetID
		default:
			return PublishResult{}, fmt.Errorf("load market asset %q: %w", asset.Key, err)
		}
	}

	sectorDefinitionIDs := make(map[string]pgtype.UUID, len(release.SectorDefinitions))
	for _, definition := range release.SectorDefinitions {
		definitionID, err := randomUUID()
		if err != nil {
			return PublishResult{}, err
		}
		if err := queries.InsertSectorDefinitionVersion(ctx, query.InsertSectorDefinitionVersionParams{
			ID: definitionID, SectorKey: definition.Sector, ContentVersion: release.Version,
			BenchmarkMethod: definition.BenchmarkMethod, MinimumEligibleBaskets: int32(definition.MinimumEligibleBaskets),
			RelativeScaleUnits: definition.RelativeScale, RelativeBlendWeightUnits: definition.RelativeBlendWeight,
			RankBlendWeightUnits: definition.RankBlendWeight, ScoreCapUnits: definition.ScoreCap,
		}); err != nil {
			return PublishResult{}, fmt.Errorf("insert sector definition %q: %w", definition.Sector, err)
		}
		sectorDefinitionIDs[definition.Sector] = definitionID
	}

	turbulencePolicyIDs := make(map[string]pgtype.UUID, len(release.TurbulencePolicies))
	for _, policy := range release.TurbulencePolicies {
		policyID, err := randomUUID()
		if err != nil {
			return PublishResult{}, err
		}
		if err := queries.InsertExpectedTurbulencePolicy(ctx, query.InsertExpectedTurbulencePolicyParams{
			ID: policyID, PolicyKey: policy.Key, ContentVersion: release.Version, PolicyType: policy.Type,
			StaticValueUnits: int64Pointer(policy.StaticValue), FloorUnits: policy.Floor, RoundingMode: policy.RoundingMode,
		}); err != nil {
			return PublishResult{}, fmt.Errorf("insert expected turbulence policy %q: %w", policy.Key, err)
		}
		turbulencePolicyIDs[policy.Key] = policyID
	}

	mappingIDs := make(map[string]pgtype.UUID, len(release.Baskets))
	for _, basket := range release.Baskets {
		mappingID, err := randomUUID()
		if err != nil {
			return PublishResult{}, err
		}
		if err := queries.InsertBasketMappingVersion(ctx, query.InsertBasketMappingVersionParams{
			ID: mappingID, MappingKey: basket.Key, ContentVersion: release.Version,
			SectorDefinitionVersionID:  sectorDefinitionIDs[basket.Sector],
			MinimumCoveredWeightUnits:  int64Pointer(basket.MinimumCoveredWeight),
			ExpectedTurbulencePolicyID: turbulencePolicyIDs[basket.ExpectedTurbulencePolicyKey],
			NormalizationCapUnits:      int64Pointer(basket.NormalizationCap),
			BenchmarkEligible:          boolPointer(basket.BenchmarkEligible, basket.Sector != ""),
		}); err != nil {
			return PublishResult{}, fmt.Errorf("insert basket mapping %q: %w", basket.Key, err)
		}
		mappingIDs[basket.Key] = mappingID
		for _, component := range basket.Components {
			if err := queries.InsertBasketMappingComponent(ctx, query.InsertBasketMappingComponentParams{
				BasketMappingVersionID: mappingID,
				MarketAssetID:          assetIDs[component.AssetKey],
				Weight:                 fixedWeight(component.Weight),
			}); err != nil {
				return PublishResult{}, fmt.Errorf("insert basket component %q/%q: %w", basket.Key, component.AssetKey, err)
			}
		}
	}

	for _, definition := range release.Definitions {
		definitionID, err := randomUUID()
		if err != nil {
			return PublishResult{}, err
		}
		if err := queries.InsertKeeperDefinitionVersion(ctx, query.InsertKeeperDefinitionVersionParams{
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
		}); err != nil {
			return PublishResult{}, fmt.Errorf("insert Keeper definition %q: %w", definition.Key, err)
		}
	}

	if err := queries.PublishContentRelease(ctx, release.Version); err != nil {
		return PublishResult{}, fmt.Errorf("publish catalog release: %w", err)
	}
	counts, err := queries.CountCatalogReleaseRows(ctx, release.Version)
	if err != nil {
		return PublishResult{}, fmt.Errorf("count catalog release: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return PublishResult{}, fmt.Errorf("commit catalog release: %w", err)
	}
	return PublishResult{
		Version:         release.Version,
		Checksum:        checksum,
		DefinitionCount: counts.DefinitionCount,
		MappingCount:    counts.MappingCount,
		ComponentCount:  counts.ComponentCount,
	}, nil
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
