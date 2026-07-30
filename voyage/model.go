// Package voyage defines Voyage domain and read values shared by application services.
package voyage

import "time"

// Status is the materialized Voyage lifecycle state.
type Status string

const (
	// StatusActive is a Voyage still in progress.
	StatusActive Status = "ACTIVE"
	// StatusCompleted is a successfully completed Voyage.
	StatusCompleted Status = "COMPLETED"
	// StatusFailed is a Voyage ended by depleted Hull.
	StatusFailed Status = "FAILED"
	// StatusAbandoned is a player-abandoned Voyage.
	StatusAbandoned Status = "ABANDONED"
)

// Summary is the transport-independent Voyage portion of daily context.
type Summary struct {
	PublicID      string
	Status        Status
	DayNumber     int
	FundHealth    int
	MaxFundHealth int
	Capital       int
	Score         string
}

// Voyage is the aggregate that owns authoritative Hull state.
type Voyage struct {
	ID          string
	Status      Status
	Hull        Hull
	CompletedAt *time.Time
}

// HullSourceType identifies the domain operation that requested a Hull change.
type HullSourceType string

const (
	HullSourceSettlement     HullSourceType = "SETTLEMENT"
	HullSourceDailyObjective HullSourceType = "DAILY_OBJECTIVE"
	HullSourceReward         HullSourceType = "REWARD"
	HullSourceRelic          HullSourceType = "RELIC"
	HullSourceSynergy        HullSourceType = "SYNERGY"
	HullSourceBoss           HullSourceType = "BOSS"
	HullSourceEvent          HullSourceType = "EVENT"
	HullSourceVoyageStart    HullSourceType = "VOYAGE_START"
	HullSourceAdminRepair    HullSourceType = "ADMIN_REPAIR"
)

// HullMutation describes a requested Hull effect and its stable provenance.
type HullMutation struct {
	Delta      int32
	SourceType HullSourceType
	SourceID   string
	ReasonKey  string
}

// ApplyHullMutation applies an ordinary gameplay Hull effect to an active Voyage.
// It transitions the Voyage to FAILED at the caller-supplied authoritative time
// when the resulting Hull is destroyed.
func (v *Voyage) ApplyHullMutation(mutation HullMutation, occurredAt time.Time) (HullChange, error) {
	if v.Status != StatusActive {
		return HullChange{}, ErrVoyageNotActive
	}
	if mutation.Delta == 0 {
		return HullChange{}, ErrZeroHullMutation
	}

	next, change := v.Hull.ApplyDelta(mutation.Delta)
	if change.EffectiveDelta == 0 {
		return HullChange{}, ErrHullMutationNoEffect
	}

	v.Hull = next
	if change.Destroyed {
		v.Status = StatusFailed
		completedAt := occurredAt
		v.CompletedAt = &completedAt
	}

	return change, nil
}

// IsTerminal reports whether status takes presentation precedence over daily phase.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusCompleted, StatusFailed, StatusAbandoned:
		return true
	case StatusActive:
		return false
	default:
		return false
	}
}
