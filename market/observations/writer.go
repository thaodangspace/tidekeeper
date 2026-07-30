// Package observations provides the controlled internal write boundary for
// provider-independent market calculation evidence.
package observations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

var ErrInvalidObservation = errors.New("invalid market observation")

type Window struct {
	ID                          string
	DailyTideID                 string
	ProviderKey                 string
	Start                       time.Time
	End                         time.Time
	RequiredGranularitySeconds  int
	ObservationToleranceSeconds int
}

type Observation struct {
	ID                  string
	CalculationWindowID string
	MarketAssetID       string
	ObservedAt          time.Time
	RawPrice            string
	Price               *string
	QualityStatus       string
	QualityFlags        []string
	RawResponseHash     []byte
}

type Writer struct{ pool *pgxpool.Pool }

func NewWriter(pool *pgxpool.Pool) *Writer { return &Writer{pool: pool} }

func (writer *Writer) InsertWindow(ctx context.Context, window Window) error {
	if writer == nil || writer.pool == nil || window.ID == "" || window.DailyTideID == "" || window.ProviderKey == "" || !window.Start.Before(window.End) || window.RequiredGranularitySeconds <= 0 || window.ObservationToleranceSeconds < 0 {
		return ErrInvalidObservation
	}
	id, err := uuidValue(window.ID)
	if err != nil {
		return fmt.Errorf("window ID: %w", err)
	}
	tideID, err := uuidValue(window.DailyTideID)
	if err != nil {
		return fmt.Errorf("daily tide ID: %w", err)
	}
	return query.New(writer.pool).InsertMarketCalculationWindow(ctx, query.InsertMarketCalculationWindowParams{
		ID: id, DailyTideID: tideID, ProviderKey: window.ProviderKey,
		DataWindowStart: timestamptz(window.Start), DataWindowEnd: timestamptz(window.End),
		RequiredGranularitySeconds: int32(window.RequiredGranularitySeconds), ObservationToleranceSeconds: int32(window.ObservationToleranceSeconds),
	})
}

func (writer *Writer) InsertObservation(ctx context.Context, observation Observation) error {
	if writer == nil || writer.pool == nil || observation.ID == "" || observation.CalculationWindowID == "" || observation.MarketAssetID == "" || observation.ObservedAt.IsZero() || observation.RawPrice == "" {
		return ErrInvalidObservation
	}
	if observation.QualityStatus != "VALID" && observation.QualityStatus != "INVALID_PRICE" && observation.QualityStatus != "OUTLIER_FLAGGED" && observation.QualityStatus != "PROVIDER_REJECTED" {
		return ErrInvalidObservation
	}
	id, err := uuidValue(observation.ID)
	if err != nil {
		return fmt.Errorf("observation ID: %w", err)
	}
	windowID, err := uuidValue(observation.CalculationWindowID)
	if err != nil {
		return fmt.Errorf("calculation window ID: %w", err)
	}
	assetID, err := uuidValue(observation.MarketAssetID)
	if err != nil {
		return fmt.Errorf("market asset ID: %w", err)
	}
	price := pgtype.Numeric{}
	if observation.Price != nil {
		if err := price.Scan(*observation.Price); err != nil {
			return fmt.Errorf("observation price: %w", err)
		}
	}
	if observation.QualityStatus == "VALID" && (!price.Valid || price.Int.Sign() <= 0) {
		return ErrInvalidObservation
	}
	flags := observation.QualityFlags
	if flags == nil {
		flags = []string{}
	}
	encodedFlags, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("quality flags: %w", err)
	}
	if len(observation.RawResponseHash) != 0 && len(observation.RawResponseHash) != 32 {
		return ErrInvalidObservation
	}
	return query.New(writer.pool).InsertMarketPriceObservation(ctx, query.InsertMarketPriceObservationParams{
		ID: id, CalculationWindowID: windowID, MarketAssetID: assetID, ObservedAt: timestamptz(observation.ObservedAt),
		RawPrice: observation.RawPrice, Price: price, QualityStatus: observation.QualityStatus, QualityFlags: encodedFlags, RawResponseHash: observation.RawResponseHash,
	})
}

func uuidValue(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}
