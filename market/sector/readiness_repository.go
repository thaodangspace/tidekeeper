package sector

import (
	"context"
	"fmt"

	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// LoadReadiness reads the durable target-sector gate for one Daily Tide/content release.
func (repository *Repository) LoadReadiness(ctx context.Context, dailyTideID string, contentVersion int64) (Readiness, error) {
	if repository == nil || repository.pool == nil {
		return Readiness{}, fmt.Errorf("sector readiness requires a repository")
	}
	tideID, err := parseUUID(dailyTideID)
	if err != nil {
		return Readiness{}, fmt.Errorf("daily tide ID: %w", err)
	}
	rows, err := query.New(repository.pool).ListTargetSectorBenchmarkReadiness(ctx, query.ListTargetSectorBenchmarkReadinessParams{
		DailyTideID: tideID, ContentVersion: contentVersion,
	})
	if err != nil {
		return Readiness{}, fmt.Errorf("load target sector readiness: %w", err)
	}
	bySector := make(map[Key]ReadinessEntry, len(rows))
	for _, row := range rows {
		key := Key(row.SectorKey)
		if !key.IsCalculationTarget() {
			return Readiness{}, fmt.Errorf("invalid persisted target sector readiness row")
		}
		entry := ReadinessEntry{Sector: key, MinimumRequiredBaskets: int(row.MinimumRequiredBaskets)}
		if row.Status != nil {
			entry.Status = Status(*row.Status)
			entry.EligibleBasketCount = int(*row.EligibleBasketCount)
			entry.Ready = entry.Status == StatusReady && entry.EligibleBasketCount >= entry.MinimumRequiredBaskets
		}
		bySector[key] = entry
	}
	result := Readiness{Ready: true, Sectors: make([]ReadinessEntry, 0, len(TargetKeys()))}
	for _, key := range TargetKeys() {
		entry, exists := bySector[key]
		if !exists {
			entry = ReadinessEntry{Sector: key}
		}
		if !entry.Ready {
			result.Ready = false
		}
		result.Sectors = append(result.Sectors, entry)
	}
	return result, nil
}
