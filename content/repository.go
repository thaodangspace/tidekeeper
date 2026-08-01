package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// Repository reads complete gameplay releases and resolves exact definition
// versions by UUID without imposing N+1 query patterns.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a gameplay-content repository backed by PostgreSQL.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// LoadRelease reconstructs the complete release at version, including the Keeper
// catalog children and every versioned definition kind.
func (r *Repository) LoadRelease(ctx context.Context, version int64) (Release, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Release{}, fmt.Errorf("begin release load: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := query.New(tx)

	if _, loadErr := queries.GetContentRelease(ctx, version); loadErr != nil {
		return Release{}, fmt.Errorf("load content release %d: %w", version, loadErr)
	}

	release := Release{Version: version}
	release.Keeper.Version = version
	keeperCatalog, err := loadKeeperCatalog(ctx, queries, version)
	if err != nil {
		return Release{}, err
	}
	release.Keeper = keeperCatalog

	if err := loadGameplayContent(ctx, queries, version, &release); err != nil {
		return Release{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Release{}, fmt.Errorf("commit release load: %w", err)
	}
	return release, nil
}

// loadKeeperCatalog reconstructs the approved Keeper catalog children persisted
// for one release through a transaction-scoped query set. It is shared by
// Release loading and cutover target validation.
func loadKeeperCatalog(ctx context.Context, queries *query.Queries, version int64) (keeper.CatalogRelease, error) {
	catalog := keeper.CatalogRelease{Version: version}

	mappings, err := queries.ListBasketMappingsForRelease(ctx, version)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load basket mappings: %w", err)
	}
	components, err := queries.ListBasketComponentsForRelease(ctx, version)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load basket components: %w", err)
	}
	componentsByMapping := make(map[string][]keeper.BasketComponent, len(mappings))
	referencedAssetKeys := make(map[string]struct{}, len(components))
	for _, component := range components {
		units, unitsErr := fixedPointUnits(component.Weight, keeper.WeightScale)
		if unitsErr != nil {
			return keeper.CatalogRelease{}, fmt.Errorf("decode weight for %q/%q: %w", component.MappingKey, component.AssetKey, unitsErr)
		}
		componentsByMapping[component.MappingKey] = append(componentsByMapping[component.MappingKey], keeper.BasketComponent{
			AssetKey: component.AssetKey, Weight: units,
		})
		referencedAssetKeys[component.AssetKey] = struct{}{}
	}

	assets, err := queries.ListMarketAssets(ctx)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load market assets: %w", err)
	}
	for _, asset := range assets {
		if _, referenced := referencedAssetKeys[asset.AssetKey]; !referenced {
			continue
		}
		catalog.Assets = append(catalog.Assets, keeper.MarketAsset{Key: asset.AssetKey, Symbol: asset.Symbol})
	}

	sectors, err := queries.ListSectorDefinitionsForRelease(ctx, version)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load sector definitions: %w", err)
	}
	for _, definition := range sectors {
		catalog.SectorDefinitions = append(catalog.SectorDefinitions, keeper.SectorDefinition{
			Sector: definition.SectorKey, BenchmarkMethod: definition.BenchmarkMethod,
			MinimumEligibleBaskets: int(definition.MinimumEligibleBaskets),
			RelativeScale:          definition.RelativeScaleUnits,
			RelativeBlendWeight:    definition.RelativeBlendWeightUnits,
			RankBlendWeight:        definition.RankBlendWeightUnits,
			ScoreCap:               definition.ScoreCapUnits,
		})
	}

	policies, err := queries.ListTurbulencePoliciesForRelease(ctx, version)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load turbulence policies: %w", err)
	}
	for _, policy := range policies {
		catalog.TurbulencePolicies = append(catalog.TurbulencePolicies, keeper.ExpectedTurbulencePolicy{
			Key: policy.PolicyKey, Type: policy.PolicyType,
			StaticValue: derefInt64(policy.StaticValueUnits),
			Floor:       policy.FloorUnits, RoundingMode: policy.RoundingMode,
		})
	}

	for _, mapping := range mappings {
		catalog.Baskets = append(catalog.Baskets, keeper.BasketMapping{
			Key: mapping.MappingKey, Sector: mapping.SectorKey,
			MinimumCoveredWeight:        derefInt64(mapping.MinimumCoveredWeightUnits),
			ExpectedTurbulencePolicyKey: mapping.TurbulencePolicyKey,
			NormalizationCap:            derefInt64(mapping.NormalizationCapUnits),
			BenchmarkEligible:           derefBool(mapping.BenchmarkEligible),
			Components:                  componentsByMapping[mapping.MappingKey],
		})
	}

	definitions, err := queries.ListKeeperDefinitionsForRelease(ctx, version)
	if err != nil {
		return keeper.CatalogRelease{}, fmt.Errorf("load keeper definitions: %w", err)
	}
	for _, definition := range definitions {
		catalog.Definitions = append(catalog.Definitions, keeper.Definition{
			Key: definition.KeeperKey, Name: definition.Name,
			CurrentKey: definition.CurrentKey, CurrentName: definition.CurrentName,
			Sector: definition.SectorKey, Role: definition.RoleKey,
			Rarity: definition.RarityKey, BaseRisk: definition.BaseRiskKey,
			ExpectedTurbulenceBPS: int32Pointer(definition.ExpectedTurbulenceBps),
			PassiveRuleKey:        definition.PassiveRuleKey,
			PassiveRuleConfig:     json.RawMessage(definition.PassiveRuleConfig),
			UpgradeTree:           json.RawMessage(definition.UpgradeTree),
			BasketMappingKey:      definition.BasketMappingKey,
		})
	}
	return catalog, nil
}

// GetStrategy resolves one strategy definition version by UUID.
func (r *Repository) GetStrategy(ctx context.Context, id pgtype.UUID) (Strategy, error) {
	row, err := r.queries().GetStrategyDefinitionVersion(ctx, id)
	if err != nil {
		return Strategy{}, err
	}
	return strategyFromModel(row), nil
}

// GetRelic resolves one relic definition version by UUID.
func (r *Repository) GetRelic(ctx context.Context, id pgtype.UUID) (Relic, error) {
	row, err := r.queries().GetRelicDefinitionVersion(ctx, id)
	if err != nil {
		return Relic{}, err
	}
	return relicFromModel(row), nil
}

// GetSynergy resolves one synergy definition version by UUID.
func (r *Repository) GetSynergy(ctx context.Context, id pgtype.UUID) (Synergy, error) {
	row, err := r.queries().GetSynergyDefinitionVersion(ctx, id)
	if err != nil {
		return Synergy{}, err
	}
	return synergyFromModel(row), nil
}

// GetModifier resolves one daily modifier definition version by UUID.
func (r *Repository) GetModifier(ctx context.Context, id pgtype.UUID) (Modifier, error) {
	row, err := r.queries().GetDailyModifierDefinitionVersion(ctx, id)
	if err != nil {
		return Modifier{}, err
	}
	return modifierFromModel(row), nil
}

// GetObjective resolves one daily objective definition version by UUID.
func (r *Repository) GetObjective(ctx context.Context, id pgtype.UUID) (Objective, error) {
	row, err := r.queries().GetDailyObjectiveDefinitionVersion(ctx, id)
	if err != nil {
		return Objective{}, err
	}
	return objectiveFromModel(row), nil
}

// GetGameRuleSet resolves one game-rule-set definition version by UUID.
func (r *Repository) GetGameRuleSet(ctx context.Context, id pgtype.UUID) (GameRuleSet, error) {
	row, err := r.queries().GetGameRuleSetVersion(ctx, id)
	if err != nil {
		return GameRuleSet{}, err
	}
	return gameRuleSetFromModel(row), nil
}

func loadGameplayContent(ctx context.Context, queries *query.Queries, version int64, release *Release) error {
	strategies, err := queries.ListStrategiesForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load strategies: %w", err)
	}
	for _, row := range strategies {
		release.Strategies = append(release.Strategies, strategyFromRow(row))
	}
	relics, err := queries.ListRelicsForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load relics: %w", err)
	}
	for _, row := range relics {
		release.Relics = append(release.Relics, relicFromRow(row))
	}
	synergies, err := queries.ListSynergiesForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load synergies: %w", err)
	}
	for _, row := range synergies {
		release.Synergies = append(release.Synergies, synergyFromRow(row))
	}
	modifiers, err := queries.ListModifiersForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load modifiers: %w", err)
	}
	for _, row := range modifiers {
		release.Modifiers = append(release.Modifiers, modifierFromRow(row))
	}
	objectives, err := queries.ListObjectivesForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load objectives: %w", err)
	}
	for _, row := range objectives {
		release.Objectives = append(release.Objectives, objectiveFromRow(row))
	}
	gameRuleSets, err := queries.ListGameRuleSetsForRelease(ctx, version)
	if err != nil {
		return fmt.Errorf("load game rule sets: %w", err)
	}
	for _, row := range gameRuleSets {
		release.GameRuleSets = append(release.GameRuleSets, gameRuleSetFromRow(row))
	}
	return nil
}

func (r *Repository) queries() *query.Queries {
	return query.New(r.pool)
}

func strategyFromRow(row query.ListStrategiesForReleaseRow) Strategy {
	return Strategy{
		Key: row.StrategyKey, Name: row.Name, Description: row.Description,
		Upside: row.Upside, Downside: row.Downside,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func strategyFromModel(row query.StrategyDefinitionVersion) Strategy {
	return Strategy{
		Key: row.StrategyKey, Name: row.Name, Description: row.Description,
		Upside: row.Upside, Downside: row.Downside,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func relicFromRow(row query.ListRelicsForReleaseRow) Relic {
	return Relic{
		Key: row.RelicKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func relicFromModel(row query.RelicDefinitionVersion) Relic {
	return Relic{
		Key: row.RelicKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func synergyFromRow(row query.ListSynergiesForReleaseRow) Synergy {
	return Synergy{
		Key: row.SynergyKey, Name: row.Name, Description: row.Description,
		RequiredCount: int(row.RequiredCount),
		RuleKey:       row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func synergyFromModel(row query.SynergyDefinitionVersion) Synergy {
	return Synergy{
		Key: row.SynergyKey, Name: row.Name, Description: row.Description,
		RequiredCount: int(row.RequiredCount),
		RuleKey:       row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func modifierFromRow(row query.ListModifiersForReleaseRow) Modifier {
	return Modifier{
		Key: row.ModifierKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func modifierFromModel(row query.DailyModifierDefinitionVersion) Modifier {
	return Modifier{
		Key: row.ModifierKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func objectiveFromRow(row query.ListObjectivesForReleaseRow) Objective {
	return Objective{
		Key: row.ObjectiveKey, Name: row.Name, Description: row.Description,
		ProgressLabel: row.ProgressLabel, RewardLabel: row.RewardLabel,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func objectiveFromModel(row query.DailyObjectiveDefinitionVersion) Objective {
	return Objective{
		Key: row.ObjectiveKey, Name: row.Name, Description: row.Description,
		ProgressLabel: row.ProgressLabel, RewardLabel: row.RewardLabel,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func gameRuleSetFromRow(row query.ListGameRuleSetsForReleaseRow) GameRuleSet {
	return GameRuleSet{
		Key: row.RuleSetKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func gameRuleSetFromModel(row query.GameRuleSetVersion) GameRuleSet {
	return GameRuleSet{
		Key: row.RuleSetKey, Name: row.Name, Description: row.Description,
		RuleKey: row.RuleKey, RuleConfig: json.RawMessage(row.RuleConfig),
	}
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefBool(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func int32Pointer(value *int32) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

// fixedPointUnits converts a stored numeric back to the fixed-point integer
// represented by scale units (units = value * scale), rejecting fractional
// remainders so corrupted rows surface instead of silently truncating.
func fixedPointUnits(value pgtype.Numeric, scale int64) (int64, error) {
	if !value.Valid {
		return 0, errors.New("null fixed-point value")
	}
	factor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(value.Exp))), nil)
	rat := new(big.Rat)
	if value.Exp >= 0 {
		rat.SetInt(new(big.Int).Mul(value.Int, factor))
	} else {
		rat.SetFrac(value.Int, factor)
	}
	rat.Mul(rat, new(big.Rat).SetInt(big.NewInt(scale)))
	if !rat.IsInt() {
		return 0, errors.New("non-integral fixed-point value")
	}
	units := rat.Num()
	if !units.IsInt64() {
		return 0, errors.New("fixed-point value overflows int64")
	}
	return units.Int64(), nil
}

func abs(value int32) int32 {
	if value < 0 {
		return -value
	}
	return value
}
