package voyage

import (
	"errors"
	"testing"
	"time"
)

func TestVoyageApplyHullMutation(t *testing.T) {
	occurredAt := time.Date(2026, time.July, 30, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		status        Status
		current       int32
		delta         int32
		wantErr       error
		wantHull      int32
		wantStatus    Status
		wantCompleted bool
	}{
		{
			name:       "active voyage takes damage",
			status:     StatusActive,
			current:    80,
			delta:      -12,
			wantHull:   68,
			wantStatus: StatusActive,
		},
		{
			name:          "lethal damage fails voyage",
			status:        StatusActive,
			current:       8,
			delta:         -12,
			wantHull:      0,
			wantStatus:    StatusFailed,
			wantCompleted: true,
		},
		{
			name:       "active voyage heals",
			status:     StatusActive,
			current:    80,
			delta:      10,
			wantHull:   90,
			wantStatus: StatusActive,
		},
		{
			name:       "terminal voyage rejects mutation",
			status:     StatusCompleted,
			current:    80,
			delta:      -12,
			wantErr:    ErrVoyageNotActive,
			wantHull:   80,
			wantStatus: StatusCompleted,
		},
		{
			name:       "zero delta is rejected",
			status:     StatusActive,
			current:    80,
			delta:      0,
			wantErr:    ErrZeroHullMutation,
			wantHull:   80,
			wantStatus: StatusActive,
		},
		{
			name:       "capped healing with no effect is rejected",
			status:     StatusActive,
			current:    100,
			delta:      10,
			wantErr:    ErrHullMutationNoEffect,
			wantHull:   100,
			wantStatus: StatusActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hull, err := NewHull(tt.current, 100)
			if err != nil {
				t.Fatalf("NewHull() error = %v", err)
			}
			voyage := Voyage{
				ID:     "voy_test",
				Status: tt.status,
				Hull:   hull,
			}

			change, err := voyage.ApplyHullMutation(HullMutation{
				Delta:      tt.delta,
				SourceType: HullSourceSettlement,
				SourceID:   "tide_test",
				ReasonKey:  "settlement.score_deficit_damage",
			}, occurredAt)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ApplyHullMutation() error = %v, want %v", err, tt.wantErr)
			}
			if got := voyage.Hull.Current(); got != tt.wantHull {
				t.Errorf("Hull.Current() = %d, want %d", got, tt.wantHull)
			}
			if got := voyage.Status; got != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got, tt.wantStatus)
			}
			if tt.wantErr == nil {
				if got := change.After; got != tt.wantHull {
					t.Errorf("change.After = %d, want %d", got, tt.wantHull)
				}
			} else if change != (HullChange{}) {
				t.Errorf("change = %+v, want zero value after rejection", change)
			}

			if tt.wantCompleted {
				if voyage.CompletedAt == nil {
					t.Fatal("CompletedAt = nil, want supplied occurrence time")
				}
				if !voyage.CompletedAt.Equal(occurredAt) {
					t.Errorf("CompletedAt = %s, want %s", voyage.CompletedAt, occurredAt)
				}
			} else if voyage.CompletedAt != nil {
				t.Errorf("CompletedAt = %s, want nil", voyage.CompletedAt)
			}
		})
	}
}
