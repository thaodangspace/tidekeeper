package sector

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/telemetry"
)

// Service composes pure Sector benchmark calculation with atomic persistence.
type Service struct {
	calculator BenchmarkCalculator
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

// CalculateAndPersist calculates one immutable target-sector benchmark and
// persists it exactly once for its Daily Tide/definition scope.
func (service *Service) CalculateAndPersist(ctx context.Context, input BenchmarkInput) (Benchmark, string, bool, error) {
	if service == nil || service.repository == nil {
		return Benchmark{}, "", false, errors.New("sector calculation service requires a repository")
	}
	started := time.Now()
	benchmark, err := service.calculator.Calculate(input)
	if err != nil {
		service.record(input, Benchmark{}, started, err)
		return Benchmark{}, "", false, err
	}
	benchmarkID, idempotent, err := service.repository.PersistBenchmark(ctx, input.DailyTideID, input.Definition, benchmark)
	service.record(input, benchmark, started, err)
	if err != nil {
		return Benchmark{}, "", false, err
	}
	return benchmark, benchmarkID, idempotent, nil
}

func (service *Service) record(input BenchmarkInput, benchmark Benchmark, started time.Time, err error) {
	if service.recorder == nil {
		return
	}
	service.recorder.RecordMarketCalculation(telemetry.Event{
		Operation: "calculate_sector_benchmark", DailyTideID: input.DailyTideID, Sector: string(input.Definition.Sector),
		Status: string(benchmark.Status), Checksum: hex.EncodeToString(benchmark.InputChecksum[:]), Duration: time.Since(started), Err: err,
	})
}
