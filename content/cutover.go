package content

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// CutoverParams describes one legacy Daily Tide cutover attempt. Apply=false
// produces a dry-run report and writes nothing.
type CutoverParams struct {
	SourceVersion int64
	TargetVersion int64
	Apply         bool
}

// CutoverResult summarizes a cutover attempt. Counts and identifiers are
// evidence an operator can act on without revealing rule configurations.
type CutoverResult struct {
	SourceVersion     int64
	TargetVersion     int64
	FingerprintLabels []string
	SourceCatalog     string
	TargetChecksum    [32]byte
	Idempotent        bool
	Applied           bool
	TideCount         int
	StateCount        int
	StrategyCount     int
	AvailabilityCount int
	ChangedTideCount  int
}

// CutoverError identifies why a legacy cutover cannot proceed safely. Offending
// identifiers let operators act on evidence; the transaction that produced the
// error leaves every gameplay row unchanged.
type CutoverError struct {
	Category    string
	Detail      string
	TideIDs     []string
	StateIDs    []string
	StrategyIDs []string
}

func (e *CutoverError) Error() string {
	message := fmt.Sprintf("cutover %s: %s", e.Category, e.Detail)
	if len(e.TideIDs) > 0 {
		message += " (tides: " + sortedJoin(e.TideIDs) + ")"
	}
	if len(e.StateIDs) > 0 {
		message += " (states: " + sortedJoin(e.StateIDs) + ")"
	}
	if len(e.StrategyIDs) > 0 {
		message += " (strategies: " + sortedJoin(e.StrategyIDs) + ")"
	}
	return message
}

// Cutover migrates recognized legacy Daily Tide projections and selections to
// exact schema-2 references in one transaction.
type Cutover struct {
	pool *pgxpool.Pool
}

// NewCutover creates a legacy Daily Tide cutover service backed by PostgreSQL.
func NewCutover(pool *pgxpool.Pool) *Cutover {
	return &Cutover{pool: pool}
}

// Run validates the recognized source and the compatible published schema-2
// target, inspects every projection attached to each source Tide, and either
// reports the plan or applies it in one transaction. Re-running a completed
// cutover returns an idempotent result with zero changed rows.
func (c *Cutover) Run(ctx context.Context, params CutoverParams) (CutoverResult, error) {
	if params.SourceVersion <= 0 || params.TargetVersion <= 0 {
		return CutoverResult{}, errors.New("cutover source and target versions must be positive")
	}
	if params.SourceVersion == params.TargetVersion {
		return CutoverResult{}, errors.New("cutover source and target versions must differ")
	}

	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return CutoverResult{}, fmt.Errorf("begin cutover: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := query.New(tx)

	sourceCatalog, recognized := SourceCatalogForVersion(params.SourceVersion)
	if !recognized {
		return CutoverResult{}, &CutoverError{
			Category: "unknown-source-version",
			Detail:   fmt.Sprintf("source version %d is not a recognized legacy source", params.SourceVersion),
		}
	}

	targetRelease, err := queries.GetContentRelease(ctx, params.TargetVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CutoverResult{}, &CutoverError{
				Category: "missing-target",
				Detail:   fmt.Sprintf("content release %d does not exist", params.TargetVersion),
			}
		}
		return CutoverResult{}, fmt.Errorf("load target release: %w", err)
	}
	if targetRelease.Status != "PUBLISHED" {
		return CutoverResult{}, &CutoverError{
			Category: "target-not-published",
			Detail:   fmt.Sprintf("content release %d is not published", params.TargetVersion),
		}
	}
	if targetRelease.ChecksumSchemaVersion != int16(ChecksumSchemaV2) {
		return CutoverResult{}, &CutoverError{
			Category: "target-not-schema-two",
			Detail:   fmt.Sprintf("content release %d does not use checksum schema 2", params.TargetVersion),
		}
	}
	var targetChecksum [32]byte
	copy(targetChecksum[:], targetRelease.Checksum)

	targetKeeper, err := loadKeeperCatalog(ctx, queries, params.TargetVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load target keeper catalog: %w", err)
	}
	compatible, err := keeperCompatible(sourceCatalogRelease(sourceCatalog), targetKeeper)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("validate target keeper compatibility: %w", err)
	}
	if !compatible {
		return CutoverResult{}, &CutoverError{
			Category: "target-mismatch",
			Detail:   fmt.Sprintf("content release %d does not embed the approved %s keeper catalog", params.TargetVersion, sourceCatalog),
		}
	}

	modifiers, err := queries.ListModifiersForRelease(ctx, params.TargetVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load target modifiers: %w", err)
	}
	objectives, err := queries.ListObjectivesForRelease(ctx, params.TargetVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load target objectives: %w", err)
	}
	gameRuleSets, err := queries.ListGameRuleSetsForRelease(ctx, params.TargetVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load target game rule sets: %w", err)
	}
	strategies, err := queries.ListStrategiesForRelease(ctx, params.TargetVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load target strategies: %w", err)
	}

	modifierByKey := make(map[string]pgtype.UUID, len(modifiers))
	for _, row := range modifiers {
		modifierByKey[row.ModifierKey] = row.ID
	}
	objectiveByKey := make(map[string]pgtype.UUID, len(objectives))
	for _, row := range objectives {
		objectiveByKey[row.ObjectiveKey] = row.ID
	}
	rulesetByKey := make(map[string]pgtype.UUID, len(gameRuleSets))
	for _, row := range gameRuleSets {
		rulesetByKey[row.RuleSetKey] = row.ID
	}
	strategyByKey := make(map[string]pgtype.UUID, len(strategies))
	for _, row := range strategies {
		strategyByKey[row.StrategyKey] = row.ID
	}

	expectedModifierID := modifierByKey[BaselineModifierKey]
	expectedObjectiveID := objectiveByKey[BaselineObjectiveKey]
	expectedRulesetID := rulesetByKey[BaselineRuleSetKey]
	if !expectedModifierID.Valid || !expectedObjectiveID.Valid || !expectedRulesetID.Valid {
		return CutoverResult{}, &CutoverError{
			Category: "target-mismatch",
			Detail:   fmt.Sprintf("content release %d does not define the baseline modifier, objective, and game rule set", params.TargetVersion),
		}
	}

	tides, err := queries.ListLegacyCutoverTides(ctx, query.ListLegacyCutoverTidesParams{
		SourceVersion: params.SourceVersion,
		TargetVersion: params.TargetVersion,
	})
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load legacy cutover tides: %w", err)
	}
	projections, err := queries.ListCutoverProjectionsForTides(ctx, params.SourceVersion)
	if err != nil {
		return CutoverResult{}, fmt.Errorf("load cutover projections: %w", err)
	}

	plan, err := buildCutoverPlan(cutoverInput{
		sourceVersion:       params.SourceVersion,
		targetVersion:       params.TargetVersion,
		sourceCatalog:       sourceCatalog,
		expectedModifierID:  expectedModifierID,
		expectedObjectiveID: expectedObjectiveID,
		expectedRulesetID:   expectedRulesetID,
		strategyByKey:       strategyByKey,
		tides:               tides,
		projections:         projections,
	})
	if err != nil {
		return CutoverResult{}, err
	}

	if params.Apply {
		for _, update := range plan.tideUpdates {
			if err := queries.UpdateDailyTideExactContent(ctx, update); err != nil {
				return CutoverResult{}, fmt.Errorf("update tide exact content: %w", err)
			}
		}
		for _, availability := range plan.availability {
			if err := queries.UpsertDailyTideStrategyVersion(ctx, availability); err != nil {
				return CutoverResult{}, fmt.Errorf("create strategy availability: %w", err)
			}
		}
		for _, selection := range plan.selections {
			if err := queries.UpdatePlayerDailyStateSelectedStrategy(ctx, selection); err != nil {
				return CutoverResult{}, fmt.Errorf("map selected strategy: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CutoverResult{}, fmt.Errorf("commit cutover: %w", err)
	}
	return plan.result(params.SourceVersion, params.TargetVersion, targetChecksum, params.Apply), nil
}

type cutoverInput struct {
	sourceVersion       int64
	targetVersion       int64
	sourceCatalog       string
	expectedModifierID  pgtype.UUID
	expectedObjectiveID pgtype.UUID
	expectedRulesetID   pgtype.UUID
	strategyByKey       map[string]pgtype.UUID
	tides               []query.ListLegacyCutoverTidesRow
	projections         []query.ListCutoverProjectionsForTidesRow
}

type cutoverPlan struct {
	fingerprintLabels []string
	sourceCatalog     string
	idempotent        bool
	tideCount         int
	stateCount        int
	strategyCount     int
	availabilityCount int
	changedTideCount  int
	tideUpdates       []query.UpdateDailyTideExactContentParams
	availability      []query.UpsertDailyTideStrategyVersionParams
	selections        []query.UpdatePlayerDailyStateSelectedStrategyParams
}

func (p cutoverPlan) result(sourceVersion, targetVersion int64, targetChecksum [32]byte, applied bool) CutoverResult {
	return CutoverResult{
		SourceVersion:     sourceVersion,
		TargetVersion:     targetVersion,
		FingerprintLabels: p.fingerprintLabels,
		SourceCatalog:     p.sourceCatalog,
		TargetChecksum:    targetChecksum,
		Idempotent:        p.idempotent,
		Applied:           applied,
		TideCount:         p.tideCount,
		StateCount:        p.stateCount,
		StrategyCount:     p.strategyCount,
		AvailabilityCount: p.availabilityCount,
		ChangedTideCount:  p.changedTideCount,
	}
}

func buildCutoverPlan(input cutoverInput) (cutoverPlan, error) {
	var legacy, inconsistent []query.ListLegacyCutoverTidesRow
	complete := 0
	for _, tide := range input.tides {
		switch {
		case tide.ContentVersion == input.sourceVersion && !hasAnyExact(tide):
			legacy = append(legacy, tide)
		case tide.ContentVersion == input.targetVersion &&
			isCompleteExact(tide, input.expectedModifierID, input.expectedObjectiveID, input.expectedRulesetID):
			complete++
		default:
			inconsistent = append(inconsistent, tide)
		}
	}

	if len(inconsistent) > 0 {
		return cutoverPlan{}, &CutoverError{
			Category: "already-partially-mapped",
			Detail:   "found Daily Tides that are not fully mapped to the target release",
			TideIDs:  tideUUIDStrings(inconsistent),
		}
	}
	if len(legacy) == 0 {
		_ = complete
		return cutoverPlan{
			sourceCatalog:    input.sourceCatalog,
			idempotent:       true,
			tideCount:        0,
			stateCount:       0,
			changedTideCount: 0,
		}, nil
	}

	projectionsByTide := make(map[pgtype.UUID][]query.ListCutoverProjectionsForTidesRow)
	for _, row := range input.projections {
		projectionsByTide[row.DailyTideID] = append(projectionsByTide[row.DailyTideID], row)
	}

	var plan cutoverPlan
	plan.sourceCatalog = input.sourceCatalog
	plan.tideCount = len(legacy)
	plan.changedTideCount = len(legacy)
	fingerprintSet := make(map[string]struct{})
	strategySet := make(map[string]struct{})

	var missingProjectionTides []string
	var contradictionTides []string
	var unknownSourceTides []string
	var unknownStrategyIDs []string
	var unknownStrategyStates []string
	var unknownStrategyTides []string

	for _, tide := range legacy {
		rows := projectionsByTide[tide.ID]
		if len(rows) == 0 {
			missingProjectionTides = append(missingProjectionTides, uuidString(tide.ID))
			continue
		}

		decoded := make([]decodedProjection, 0, len(rows))
		for _, row := range rows {
			identity, decodeErr := daily.DecodeProjectionIdentity(row.SchemaVersion, row.Projection)
			if decodeErr != nil {
				return cutoverPlan{}, &CutoverError{
					Category: "invalid-projection",
					Detail:   fmt.Sprintf("projection could not be decoded: %v", decodeErr),
					TideIDs:  []string{uuidString(tide.ID)},
					StateIDs: []string{uuidString(row.StateID)},
				}
			}
			decoded = append(decoded, decodedProjection{stateID: row.StateID, identity: identity})
		}

		consistent := true
		for _, projection := range decoded[1:] {
			if !decoded[0].identity.Compatible(projection.identity) {
				consistent = false
				break
			}
		}
		if !consistent {
			contradictionTides = append(contradictionTides, uuidString(tide.ID))
			continue
		}

		fingerprint, recognized := RecognizeSourceFingerprint(tide.ContentVersion, decoded[0].identity)
		if !recognized {
			unknownSourceTides = append(unknownSourceTides, uuidString(tide.ID))
			continue
		}
		if fingerprint.SourceCatalog != input.sourceCatalog {
			return cutoverPlan{}, &CutoverError{
				Category: "source-catalog-mismatch",
				Detail:   fmt.Sprintf("tide source fingerprint %q expects catalog %s", fingerprint.Label, fingerprint.SourceCatalog),
				TideIDs:  []string{uuidString(tide.ID)},
			}
		}
		fingerprintSet[fingerprint.Label] = struct{}{}

		plan.stateCount += len(decoded)
		for _, strategyID := range decoded[0].identity.StrategyIDs {
			targetID, ok := input.strategyByKey[strategyID]
			if !ok {
				unknownStrategyIDs = append(unknownStrategyIDs, strategyID)
				unknownStrategyTides = append(unknownStrategyTides, uuidString(tide.ID))
				continue
			}
			strategySet[strategyID] = struct{}{}
			plan.availability = append(plan.availability, query.UpsertDailyTideStrategyVersionParams{
				DailyTideID:                 tide.ID,
				StrategyDefinitionVersionID: targetID,
				ContentVersion:              input.targetVersion,
			})
		}

		plan.tideUpdates = append(plan.tideUpdates, query.UpdateDailyTideExactContentParams{
			ID:                           tide.ID,
			ContentVersion:               input.targetVersion,
			ModifierDefinitionVersionID:  input.expectedModifierID,
			ObjectiveDefinitionVersionID: input.expectedObjectiveID,
			GameRuleSetVersionID:         input.expectedRulesetID,
		})

		for _, projection := range decoded {
			if projection.identity.SelectedStrategyID == nil {
				continue
			}
			targetID, ok := input.strategyByKey[*projection.identity.SelectedStrategyID]
			if !ok {
				unknownStrategyIDs = append(unknownStrategyIDs, *projection.identity.SelectedStrategyID)
				unknownStrategyStates = append(unknownStrategyStates, uuidString(projection.stateID))
				unknownStrategyTides = append(unknownStrategyTides, uuidString(tide.ID))
				continue
			}
			plan.selections = append(plan.selections, query.UpdatePlayerDailyStateSelectedStrategyParams{
				ID:                                  projection.stateID,
				SelectedStrategyDefinitionVersionID: targetID,
			})
		}
	}

	if len(missingProjectionTides) > 0 {
		return cutoverPlan{}, &CutoverError{
			Category: "missing-projections",
			Detail:   "Daily Tide has no attached projections to recognize",
			TideIDs:  missingProjectionTides,
		}
	}
	if len(contradictionTides) > 0 {
		return cutoverPlan{}, &CutoverError{
			Category: "mixed-contradictory-projections",
			Detail:   "Daily Tide projections disagree on modifier, objective, or available strategy set",
			TideIDs:  contradictionTides,
		}
	}
	if len(unknownSourceTides) > 0 {
		return cutoverPlan{}, &CutoverError{
			Category: "unknown-source-fingerprint",
			Detail:   "projection identity is not a recognized legacy source",
			TideIDs:  unknownSourceTides,
		}
	}
	if len(unknownStrategyIDs) > 0 {
		return cutoverPlan{}, &CutoverError{
			Category:    "unknown-strategy",
			Detail:      "projection references a strategy not present in the target release",
			TideIDs:     unknownStrategyTides,
			StateIDs:    unknownStrategyStates,
			StrategyIDs: dedupeStrings(unknownStrategyIDs),
		}
	}

	plan.fingerprintLabels = make([]string, 0, len(fingerprintSet))
	for label := range fingerprintSet {
		plan.fingerprintLabels = append(plan.fingerprintLabels, label)
	}
	sort.Strings(plan.fingerprintLabels)
	plan.strategyCount = len(strategySet)
	plan.availabilityCount = len(plan.availability)
	return plan, nil
}

type decodedProjection struct {
	stateID  pgtype.UUID
	identity daily.ProjectionIdentity
}

func hasAnyExact(tide query.ListLegacyCutoverTidesRow) bool {
	return tide.ModifierDefinitionVersionID.Valid ||
		tide.ObjectiveDefinitionVersionID.Valid ||
		tide.GameRuleSetVersionID.Valid
}

func isCompleteExact(tide query.ListLegacyCutoverTidesRow, modifierID, objectiveID, rulesetID pgtype.UUID) bool {
	return tide.ModifierDefinitionVersionID.Valid &&
		tide.ObjectiveDefinitionVersionID.Valid &&
		tide.GameRuleSetVersionID.Valid &&
		tide.ModifierDefinitionVersionID == modifierID &&
		tide.ObjectiveDefinitionVersionID == objectiveID &&
		tide.GameRuleSetVersionID == rulesetID
}

func uuidString(id pgtype.UUID) string {
	return id.String()
}

func tideUUIDStrings(tides []query.ListLegacyCutoverTidesRow) []string {
	result := make([]string, 0, len(tides))
	for _, tide := range tides {
		result = append(result, uuidString(tide.ID))
	}
	return result
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func sortedJoin(values []string) string {
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}
