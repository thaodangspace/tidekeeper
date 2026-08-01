package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
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
	NodeCount       int64
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
		if storedRelease.Status != "PUBLISHED" || !bytes.Equal(storedRelease.Checksum, checksum[:]) || storedRelease.ChecksumSchemaVersion != 1 {
			return PublishResult{}, fmt.Errorf("%w: version %d has different or incomplete content", ErrReleaseConflict, release.Version)
		}
		counts, countErr := queries.CountCatalogReleaseRows(ctx, release.Version)
		if countErr != nil {
			return PublishResult{}, fmt.Errorf("count existing catalog release: %w", countErr)
		}
		return PublishResult{
			Version:         release.Version,
			Checksum:        checksum,
			DefinitionCount: counts.DefinitionCount,
			MappingCount:    counts.MappingCount,
			ComponentCount:  counts.ComponentCount,
			NodeCount:       counts.NodeCount,
			Idempotent:      true,
		}, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PublishResult{}, fmt.Errorf("load catalog release: %w", err)
	}

	if createErr := queries.CreateDraftContentRelease(ctx, query.CreateDraftContentReleaseParams{
		Version: release.Version, Checksum: checksum[:], ChecksumSchemaVersion: 1,
	}); createErr != nil {
		return PublishResult{}, fmt.Errorf("create catalog release: %w", createErr)
	}

	if writeErr := NewCatalogWriter(queries).Write(ctx, release); writeErr != nil {
		return PublishResult{}, fmt.Errorf("persist catalog children: %w", writeErr)
	}

	if publishErr := queries.PublishContentRelease(ctx, release.Version); publishErr != nil {
		return PublishResult{}, fmt.Errorf("publish catalog release: %w", publishErr)
	}
	counts, countErr := queries.CountCatalogReleaseRows(ctx, release.Version)
	if countErr != nil {
		return PublishResult{}, fmt.Errorf("count catalog release: %w", countErr)
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
		NodeCount:       counts.NodeCount,
	}, nil
}
