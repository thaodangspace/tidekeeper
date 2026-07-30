package basket

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/telemetry"
)

// Service composes pure basket calculation with atomic evidence persistence.
type Service struct {
	calculator Calculator
	repository *Repository
	recorder   telemetry.Recorder
}

func NewService(repository *Repository, recorders ...telemetry.Recorder) *Service {
	recorder := telemetry.Recorder(telemetry.NopRecorder{})
	if len(recorders) > 0 && recorders[0] != nil {
		recorder = recorders[0]
	}
	return &Service{repository: repository, recorder: recorder}
}

// CalculateAndPersist produces one deterministic metric and persists it exactly
// once for its Daily Tide/mapping scope.
func (service *Service) CalculateAndPersist(ctx context.Context, input CalculationInput) (Metric, string, bool, error) {
	if service == nil || service.repository == nil {
		return Metric{}, "", false, errors.New("basket calculation service requires a repository")
	}
	started := time.Now()
	metric, err := service.calculator.Calculate(input)
	if err != nil {
		service.record(input, Metric{}, started, err)
		return Metric{}, "", false, err
	}
	metricID, idempotent, err := service.repository.PersistMetric(ctx, input, metric)
	service.record(input, metric, started, err)
	if err != nil {
		return Metric{}, "", false, err
	}
	return metric, metricID, idempotent, nil
}

func (service *Service) record(input CalculationInput, metric Metric, started time.Time, err error) {
	if service.recorder == nil {
		return
	}
	service.recorder.RecordMarketCalculation(telemetry.Event{
		Operation: "calculate_basket_metrics", DailyTideID: input.Window.DailyTideID, Sector: string(input.Mapping.Sector),
		MappingID: input.Mapping.ID, Status: string(metric.Status), CoveredWeight: metric.CoveredWeight,
		Checksum: hex.EncodeToString(metric.InputChecksum[:]), Duration: time.Since(started), Err: err,
	})
}
