// Package daily defines daily-context read models, errors, and service boundary.
package daily

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNoActiveVoyage is returned when the player has no current voyage.
	ErrNoActiveVoyage = errors.New("no active voyage")
	// ErrVoyageNotFound is returned when the current voyage cannot be resolved.
	ErrVoyageNotFound = errors.New("voyage not found")
	// ErrInternal is returned for unexpected internal failures (projection, state, etc.).
	ErrInternal = errors.New("internal error")
	// ErrServiceUnavailable is returned when the database is unreachable.
	ErrServiceUnavailable = errors.New("service unavailable")
)

// ID is an internal player identifier used only for authorization and joins.
type ID string

// VoyageSummary is the transport-independent voyage portion of daily context.
type VoyageSummary struct {
	PublicID      string
	Status        string
	DayNumber     int32
	FundHealth    int32
	MaxFundHealth int32
	Capital       int32
	Score         string
}

// DailyContext is the complete daily context including server time, voyage, and daily detail.
type DailyContext struct {
	ServerNow time.Time
	Voyage    VoyageSummary
	Daily     DailyDetail
}

// DailyDetail contains the daily state and decoded projection for the current day.
type DailyDetail struct {
	Phase              string
	LockAt             time.Time
	SettleAfter        time.Time
	Version            int64
	Modifier           Modifier
	Objective          Objective
	Signals            []Signal
	Lineup             Lineup
	Inventory          []Keeper
	Shop               Shop
	Strategies         []Strategy
	SelectedStrategyID *string
	PendingRewardCount int32
}

// Modifier is the transport-independent daily modifier summary.
type Modifier struct {
	ID          string
	Name        string
	Description string
}

// Objective is the transport-independent daily objective summary.
type Objective struct {
	ID            string
	Name          string
	Description   string
	ProgressLabel *string
	RewardLabel   *string
}

// Signal is a market signal affecting the daily context.
type Signal struct {
	ID           string
	Name         string
	Description  string
	Direction    string
	Strength     string
	ObservedFrom time.Time
	ObservedTo   time.Time
}

// Lineup is the player's assigned fleet lineup for the day.
type Lineup struct {
	LockedAt  *time.Time
	MaxSlots  int32
	Slots     []FleetSlot
	Synergies []Synergy
	Warnings  []FleetWarning
}

// FleetSlot is a single slot in the fleet lineup.
type FleetSlot struct {
	Index  int32
	Keeper *Keeper
}

// FleetWarning is a warning about the current fleet configuration.
type FleetWarning struct {
	Code     string
	Message  string
	Severity string
}

// Synergy is an active Keeper synergy in the lineup.
type Synergy struct {
	ID            string
	Name          string
	Description   string
	State         string
	CurrentCount  int32
	RequiredCount int32
}

// Keeper is a player-owned Keeper instance in the daily context.
type Keeper struct {
	ID             string
	DefinitionID   string
	Name           string
	Level          int32
	Rarity         string
	Role           string
	Sector         string
	PassiveSummary string
	ArtworkURL     *string
}

// Shop is the daily shop offering Keeper purchases.
type Shop struct {
	Offers      []ShopOffer
	RefreshAt   *time.Time
	RerollCost  int32
	RerollIndex int32
}

// ShopOffer is a single Keeper offer in the daily shop.
type ShopOffer struct {
	ID          string
	Keeper      Keeper
	Cost        int32
	Available   bool
	SynergyHint *string
}

// Strategy is an available investment strategy for the day.
type Strategy struct {
	ID          string
	Name        string
	Description string
	Upside      string
	Downside    string
	Available   bool
}

// Reader retrieves the current daily context for a player.
type Reader interface {
	GetCurrentDailyContext(ctx context.Context, playerID ID) (DailyContext, error)
}

// Service orchestrates daily context reads.
type Service struct {
	reader Reader
}

// NewService creates a daily context read service.
func NewService(reader Reader) *Service {
	return &Service{reader: reader}
}

// GetCurrentDailyContext returns the current voyage daily context for the player.
func (s *Service) GetCurrentDailyContext(ctx context.Context, playerID ID) (DailyContext, error) {
	return s.reader.GetCurrentDailyContext(ctx, playerID)
}
