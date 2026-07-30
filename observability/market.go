package observability

import (
	"log/slog"

	"github.com/thaodangspace/tidekeepers-server/market/telemetry"
)

// MarketRecorder writes calculation events as bounded structured logs. Metrics
// backends can implement telemetry.Recorder alongside this adapter.
type MarketRecorder struct{ logger *slog.Logger }

func NewMarketRecorder(logger *slog.Logger) *MarketRecorder { return &MarketRecorder{logger: logger} }

func (recorder *MarketRecorder) RecordMarketCalculation(event telemetry.Event) {
	if recorder == nil || recorder.logger == nil {
		return
	}
	attributes := []slog.Attr{
		slog.String("operation", event.Operation), slog.String("daily_tide_id", event.DailyTideID),
		slog.String("sector_key", event.Sector), slog.String("basket_mapping_version_id", event.MappingID),
		slog.String("status", event.Status), slog.Int64("covered_weight", event.CoveredWeight),
		slog.String("input_checksum", event.Checksum), slog.Duration("duration", event.Duration),
	}
	if event.Err != nil {
		attributes = append(attributes, slog.String("error", event.Err.Error()))
		recorder.logger.LogAttrs(nil, slog.LevelError, "market calculation failed", attributes...)
		return
	}
	recorder.logger.LogAttrs(nil, slog.LevelInfo, "market calculation completed", attributes...)
}
