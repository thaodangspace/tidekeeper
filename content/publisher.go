package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// ErrReleaseConflict means the requested release version already identifies
// different published content, or a persisted release is incomplete.
var ErrReleaseConflict = errors.New("content release conflict")

// PublishResult describes a completed gameplay-content publication attempt.
type PublishResult struct {
	Version               int64
	Checksum              [sha256.Size]byte
	KeeperMappingCount    int64
	KeeperDefinitionCount int64
	ComponentCount        int64
	NodeCount             int64
	StrategyCount         int64
	RelicCount            int64
	SynergyCount          int64
	ModifierCount         int64
	ObjectiveCount        int64
	GameRuleSetCount      int64
	Idempotent            bool
}

// Publisher transactionally persists validated complete gameplay content.
type Publisher struct {
	pool *pgxpool.Pool
}

// NewPublisher creates a gameplay-content publisher backed by PostgreSQL.
func NewPublisher(pool *pgxpool.Pool) *Publisher {
	return &Publisher{pool: pool}
}

// Publish validates and atomically publishes a complete gameplay release as
// schema-2 content. Re-publishing the identical release returns the persisted
// result without writing rows. Any existing content that differs, is incomplete,
// or predates schema-2 is rejected.
func (p *Publisher) Publish(ctx context.Context, release Release) (PublishResult, error) {
	checksum, err := Checksum(release)
	if err != nil {
		return PublishResult{}, err
	}
	expected, err := countReleaseRows(release)
	if err != nil {
		return PublishResult{}, err
	}

	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PublishResult{}, fmt.Errorf("begin content publication: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, lockErr := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", releaseLockScope(release.Version)); lockErr != nil {
		return PublishResult{}, fmt.Errorf("lock content release: %w", lockErr)
	}
	queries := query.New(tx)

	storedRelease, err := queries.GetContentRelease(ctx, release.Version)
	switch {
	case err == nil:
		if storedRelease.Status != "PUBLISHED" || storedRelease.ChecksumSchemaVersion != int16(ChecksumSchemaV2) || !bytes.Equal(storedRelease.Checksum, checksum[:]) {
			return PublishResult{}, fmt.Errorf("%w: version %d has different or incomplete content", ErrReleaseConflict, release.Version)
		}
		persisted, countErr := queries.CountContentReleaseRows(ctx, release.Version)
		if countErr != nil {
			return PublishResult{}, fmt.Errorf("count existing content release: %w", countErr)
		}
		if countsDiffer(persisted, expected) {
			return PublishResult{}, fmt.Errorf("%w: version %d has incomplete content", ErrReleaseConflict, release.Version)
		}
		return PublishResult{
			Version:               release.Version,
			Checksum:              checksum,
			KeeperMappingCount:    expected.BasketMappingCount,
			KeeperDefinitionCount: expected.KeeperDefinitionCount,
			ComponentCount:        expected.ComponentCount,
			NodeCount:             expected.NodeCount,
			StrategyCount:         expected.StrategyCount,
			RelicCount:            expected.RelicCount,
			SynergyCount:          expected.SynergyCount,
			ModifierCount:         expected.ModifierCount,
			ObjectiveCount:        expected.ObjectiveCount,
			GameRuleSetCount:      expected.GameRuleSetCount,
			Idempotent:            true,
		}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PublishResult{}, fmt.Errorf("load content release: %w", err)
	}

	if createErr := queries.CreateDraftContentRelease(ctx, query.CreateDraftContentReleaseParams{
		Version: release.Version, Checksum: checksum[:], ChecksumSchemaVersion: int16(ChecksumSchemaV2),
	}); createErr != nil {
		return PublishResult{}, fmt.Errorf("create content release: %w", createErr)
	}

	if writeErr := keeper.NewCatalogWriter(queries).Write(ctx, release.Keeper); writeErr != nil {
		return PublishResult{}, fmt.Errorf("persist keeper catalog: %w", writeErr)
	}
	if writeErr := writeGameplayContent(ctx, queries, release); writeErr != nil {
		return PublishResult{}, fmt.Errorf("persist gameplay content: %w", writeErr)
	}

	if publishErr := queries.PublishContentRelease(ctx, release.Version); publishErr != nil {
		return PublishResult{}, fmt.Errorf("publish content release: %w", publishErr)
	}
	persisted, countErr := queries.CountContentReleaseRows(ctx, release.Version)
	if countErr != nil {
		return PublishResult{}, fmt.Errorf("count content release: %w", countErr)
	}
	if err := tx.Commit(ctx); err != nil {
		return PublishResult{}, fmt.Errorf("commit content release: %w", err)
	}
	return PublishResult{
		Version:               release.Version,
		Checksum:              checksum,
		KeeperMappingCount:    persisted.BasketMappingCount,
		KeeperDefinitionCount: persisted.KeeperDefinitionCount,
		ComponentCount:        persisted.ComponentCount,
		NodeCount:             persisted.NodeCount,
		StrategyCount:         persisted.StrategyCount,
		RelicCount:            persisted.RelicCount,
		SynergyCount:          persisted.SynergyCount,
		ModifierCount:         persisted.ModifierCount,
		ObjectiveCount:        persisted.ObjectiveCount,
		GameRuleSetCount:      persisted.GameRuleSetCount,
	}, nil
}

type releaseRowCounts struct {
	BasketMappingCount    int64
	KeeperDefinitionCount int64
	ComponentCount        int64
	NodeCount             int64
	StrategyCount         int64
	RelicCount            int64
	SynergyCount          int64
	ModifierCount         int64
	ObjectiveCount        int64
	GameRuleSetCount      int64
}

func countReleaseRows(release Release) (releaseRowCounts, error) {
	var counts releaseRowCounts
	counts.BasketMappingCount = int64(len(release.Keeper.Baskets))
	counts.KeeperDefinitionCount = int64(len(release.Keeper.Definitions))
	for _, basket := range release.Keeper.Baskets {
		counts.ComponentCount += int64(len(basket.Components))
	}
	nodes, err := keeper.CountUpgradeNodes(release.Keeper)
	if err != nil {
		return releaseRowCounts{}, err
	}
	counts.NodeCount = nodes
	counts.StrategyCount = int64(len(release.Strategies))
	counts.RelicCount = int64(len(release.Relics))
	counts.SynergyCount = int64(len(release.Synergies))
	counts.ModifierCount = int64(len(release.Modifiers))
	counts.ObjectiveCount = int64(len(release.Objectives))
	counts.GameRuleSetCount = int64(len(release.GameRuleSets))
	return counts, nil
}

func countsDiffer(persisted query.CountContentReleaseRowsRow, expected releaseRowCounts) bool {
	return persisted.BasketMappingCount != expected.BasketMappingCount ||
		persisted.KeeperDefinitionCount != expected.KeeperDefinitionCount ||
		persisted.ComponentCount != expected.ComponentCount ||
		persisted.NodeCount != expected.NodeCount ||
		persisted.StrategyCount != expected.StrategyCount ||
		persisted.RelicCount != expected.RelicCount ||
		persisted.SynergyCount != expected.SynergyCount ||
		persisted.ModifierCount != expected.ModifierCount ||
		persisted.ObjectiveCount != expected.ObjectiveCount ||
		persisted.GameRuleSetCount != expected.GameRuleSetCount
}

func releaseLockScope(version int64) string {
	return fmt.Sprintf("content_release:%d", version)
}

func writeGameplayContent(ctx context.Context, queries *query.Queries, release Release) error {
	for _, definition := range release.Strategies {
		if err := queries.InsertStrategyDefinitionVersion(ctx, query.InsertStrategyDefinitionVersionParams{
			ID: asPGTypeUUID(uuid.New()), StrategyKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			Upside: definition.Upside, Downside: definition.Downside,
			RuleKey: definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert strategy %q: %w", definition.Key, err)
		}
	}
	for _, definition := range release.Relics {
		if err := queries.InsertRelicDefinitionVersion(ctx, query.InsertRelicDefinitionVersionParams{
			ID: asPGTypeUUID(uuid.New()), RelicKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert relic %q: %w", definition.Key, err)
		}
	}
	for _, definition := range release.Synergies {
		if err := queries.InsertSynergyDefinitionVersion(ctx, query.InsertSynergyDefinitionVersionParams{
			ID: asPGTypeUUID(uuid.New()), SynergyKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			RequiredCount: int16(definition.RequiredCount),
			RuleKey:       definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert synergy %q: %w", definition.Key, err)
		}
	}
	for _, definition := range release.Modifiers {
		if err := queries.InsertDailyModifierDefinitionVersion(ctx, query.InsertDailyModifierDefinitionVersionParams{
			ID: asPGTypeUUID(uuid.New()), ModifierKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert modifier %q: %w", definition.Key, err)
		}
	}
	for _, definition := range release.Objectives {
		if err := queries.InsertDailyObjectiveDefinitionVersion(ctx, query.InsertDailyObjectiveDefinitionVersionParams{
			ID: asPGTypeUUID(uuid.New()), ObjectiveKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			ProgressLabel: definition.ProgressLabel, RewardLabel: definition.RewardLabel,
			RuleKey: definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert objective %q: %w", definition.Key, err)
		}
	}
	for _, definition := range release.GameRuleSets {
		if err := queries.InsertGameRuleSetVersion(ctx, query.InsertGameRuleSetVersionParams{
			ID: asPGTypeUUID(uuid.New()), RuleSetKey: definition.Key, ContentVersion: release.Version,
			Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: definition.RuleConfig,
		}); err != nil {
			return fmt.Errorf("insert game rule set %q: %w", definition.Key, err)
		}
	}
	return nil
}

func asPGTypeUUID(id uuid.UUID) pgtype.UUID {
	var value pgtype.UUID
	copy(value.Bytes[:], id[:])
	value.Valid = true
	return value
}
