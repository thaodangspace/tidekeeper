# Tidekeepers — Hull Domain Implementation Specification

**Document type:** Domain and engineering specification  
**Target:** Tidekeepers backend MVP  
**Language:** Go  
**Primary datastore:** PostgreSQL  
**Related bounded contexts:** Voyage, Settlement, Daily, Reward, Lineup, Audit  
**Status:** Draft ready for implementation  

---

## 1. Purpose

This document specifies how to implement **Hull** in Tidekeepers.

Hull is the player-facing survival resource for a Voyage. In earlier domain terminology it is called `Fund Health`. The implementation should standardize on **Hull** in Go and public API contracts while preserving compatibility with existing database or generated-client names during migration when necessary.

The Hull implementation must provide:

- One authoritative Hull balance per Voyage.
- A maximum Hull capacity.
- Deterministic damage and healing.
- Atomic persistence with settlement, rewards and Voyage state changes.
- Immutable history through Voyage ledger entries.
- Before/delta/after evidence in daily results.
- Snapshot evidence for deterministic settlement and replay.
- Strict bounds and concurrency guarantees.
- Explainable failure and recovery outcomes.

Hull is not a separate top-level aggregate. It is an owned value object inside the **Voyage aggregate**.

---

## 2. Gameplay rules represented by the domain

Hull represents the Voyage's ability to survive future Tides.

Baseline gameplay rules:

```text
Starting Hull: 100
Minimum Hull: 0
Maximum Hull: defined by Voyage content version
```

A Voyage fails when current Hull reaches zero.

Hull can decrease because of:

- A Daily Score below the required threshold.
- Boss or Daily Modifier rules.
- Event choices.
- Explicit risk or pressure effects defined by a versioned rule.

Hull can increase because of:

- Harbor synergy.
- Recovery rewards or nodes.
- Relics.
- Boss rewards.
- Daily Objective rewards.
- Explicit event choices.

Hull and Supplies are separate resources:

```text
Hull: survival
Supplies: purchasing and economy
Voyage Score: performance and final ranking
```

The domain must not infer Hull changes directly from raw market return. Settlement first calculates a Daily Score and applies the versioned Hull damage or healing policy.

---

## 3. Domain ownership and boundaries

### 3.1. Aggregate ownership

The `Voyage` aggregate owns current Hull state.

```text
Voyage
├── status
├── current day
├── hull
│   ├── current
│   └── maximum
├── supplies
├── score
└── row version
```

Only operations that load and lock the Voyage aggregate may mutate authoritative Hull state.

### 3.2. Supporting records

Other models may record Hull evidence but do not own the authoritative balance.

```text
LineupSnapshot
└── hull_before

DailyResult
├── hull_before
├── hull_delta
└── hull_after

VoyageLedgerEntry
└── hull_delta
```

### 3.3. Prohibited ownership

Hull must not be authoritative on:

- `Player`.
- `PlayerDailyState`.
- `Fleet` or lineup draft.
- `KeeperInstance`.
- Frontend stores.
- Daily result read models.

Those models may reference or display Hull but cannot independently change it.

---

## 4. Ubiquitous language

| Domain term | Meaning |
|---|---|
| Hull | Current survival balance of a Voyage |
| Maximum Hull | Upper bound for Hull during the current Voyage |
| Hull damage | A negative Hull change |
| Hull healing | A positive Hull change |
| Hull delta | Signed change applied atomically to current Hull |
| Effective delta | Actual applied change after clamping |
| Requested delta | Change requested before bounds are applied |
| Hull state | Derived display classification such as Stable or Critical |
| Destroyed | Current Hull equals zero |
| Damage curve | Versioned conversion from score deficit to damage |
| Hull effect | Settlement, reward, event or admin input requesting a Hull change |

Use `Hull` for player-facing and new domain code. Use `fund_health` only as a legacy storage or API compatibility name during migration.

---

## 5. Core invariants

The following invariants must hold after every committed transaction:

1. `0 <= current_hull <= max_hull`.
2. `max_hull > 0` for every active Voyage.
3. A Voyage with `current_hull == 0` cannot remain `ACTIVE` after an authoritative Hull mutation completes.
4. A terminal Voyage cannot receive ordinary gameplay Hull mutations.
5. One logical Hull effect is applied at most once.
6. Every committed Hull change has one immutable ledger entry.
7. The Voyage materialized Hull balance and the sum of its ledger deltas are reconcilable.
8. A daily result's `hull_after` equals the Voyage Hull after applying that result transaction.
9. The settlement engine cannot mutate the database directly.
10. The client cannot submit an authoritative current Hull, maximum Hull or Hull delta.
11. Bounds and failure transitions are enforced in the domain and rechecked within the database transaction.
12. Rule versions used to calculate damage or healing are persisted with the result evidence.

---

## 6. Go domain model

Recommended package:

```text
internal/voyage
```

Hull is a value object in the Voyage package rather than a standalone bounded context.

### 6.1. Hull value object

```go
package voyage

import "fmt"

type Hull struct {
    current int32
    maximum int32
}

func NewHull(current, maximum int32) (Hull, error) {
    if maximum <= 0 {
        return Hull{}, fmt.Errorf("maximum hull must be greater than zero")
    }
    if current < 0 || current > maximum {
        return Hull{}, fmt.Errorf("current hull must be between zero and maximum")
    }
    return Hull{current: current, maximum: maximum}, nil
}

func (h Hull) Current() int32 { return h.current }
func (h Hull) Maximum() int32 { return h.maximum }
func (h Hull) IsDestroyed() bool { return h.current == 0 }

func (h Hull) RatioBasisPoints() int32 {
    return h.current * 10_000 / h.maximum
}
```

Fields should remain private so invalid Hull values cannot be constructed through direct assignment.

### 6.2. Derived Hull state

Hull state is derived and should not be persisted unless analytics explicitly requires a historical snapshot.

```go
type HullState string

const (
    HullStateStable    HullState = "STABLE"
    HullStateDamaged   HullState = "DAMAGED"
    HullStateCritical  HullState = "CRITICAL"
    HullStateDestroyed HullState = "DESTROYED"
)

func (h Hull) State() HullState {
    if h.current == 0 {
        return HullStateDestroyed
    }

    ratio := h.RatioBasisPoints()
    switch {
    case ratio <= 2_500:
        return HullStateCritical
    case ratio <= 6_000:
        return HullStateDamaged
    default:
        return HullStateStable
    }
}
```

Thresholds are presentation defaults. If gameplay rules depend on these thresholds, move them into a versioned content policy rather than hard-coding them.

### 6.3. Hull mutation result

```go
type HullChange struct {
    Before         int32
    RequestedDelta int32
    EffectiveDelta int32
    After          int32
    Capped          bool
    Destroyed       bool
}
```

The effective delta may differ from the requested delta.

Examples:

```text
Current 95, requested +10, maximum 100
Effective delta: +5

Current 8, requested -12
Effective delta: -8
After: 0
```

### 6.4. Apply method

```go
func (h Hull) ApplyDelta(delta int32) (Hull, HullChange) {
    before := h.current
    requestedAfter := int64(before) + int64(delta)

    after := requestedAfter
    if after < 0 {
        after = 0
    }
    if after > int64(h.maximum) {
        after = int64(h.maximum)
    }

    next := Hull{
        current: int32(after),
        maximum: h.maximum,
    }

    effective := next.current - before
    return next, HullChange{
        Before:         before,
        RequestedDelta: delta,
        EffectiveDelta: effective,
        After:          next.current,
        Capped:          effective != delta,
        Destroyed:       next.IsDestroyed(),
    }
}
```

Use `int64` for intermediate addition to avoid overflow even though persisted values are `int32`-sized.

### 6.5. Maximum Hull changes

Maximum Hull changes are not required by the base MVP, but the domain should define behavior explicitly.

```go
type MaxHullChangePolicy string

const (
    MaxHullKeepCurrentRatio MaxHullChangePolicy = "KEEP_CURRENT_RATIO"
    MaxHullKeepCurrentValue MaxHullChangePolicy = "KEEP_CURRENT_VALUE"
    MaxHullFillIncrease     MaxHullChangePolicy = "FILL_INCREASE"
)
```

For MVP, use `KEEP_CURRENT_VALUE`:

- Increasing maximum Hull does not automatically heal.
- Decreasing maximum Hull clamps current Hull to the new maximum.
- Maximum Hull cannot become zero or negative.
- A maximum-Hull mutation requires its own ledger or audit representation.

Do not implement maximum Hull mutation until a published Relic, class or event requires it.

---

## 7. Voyage aggregate behavior

### 7.1. Voyage model

```go
type Voyage struct {
    ID              VoyageID
    PlayerID        PlayerID
    Status          Status
    DefinitionID    VoyageDefinitionVersionID
    CurrentDay      int16
    Hull            Hull
    Supplies        int32
    ScoreMicros     int64
    RowVersion      int64
    StartedAt       time.Time
    CompletedAt     *time.Time
}
```

### 7.2. Hull mutation method

```go
type HullMutation struct {
    Delta      int32
    SourceType HullSourceType
    SourceID   string
    ReasonKey  string
}

func (v *Voyage) ApplyHullMutation(m HullMutation) (HullChange, error) {
    if v.Status != VoyageStatusActive {
        return HullChange{}, ErrVoyageNotActive
    }
    if m.Delta == 0 {
        return HullChange{}, ErrZeroHullMutation
    }

    next, change := v.Hull.ApplyDelta(m.Delta)
    if change.EffectiveDelta == 0 {
        return HullChange{}, ErrHullMutationNoEffect
    }

    v.Hull = next
    if change.Destroyed {
        v.Status = VoyageStatusFailed
        now := time.Now().UTC() // Prefer passing clock into application service.
        v.CompletedAt = &now
    }

    return change, nil
}
```

The production implementation should avoid reading the system clock inside the aggregate. Pass the authoritative database or application clock time into the method when a terminal transition needs a timestamp.

Recommended signature:

```go
func (v *Voyage) ApplyHullMutation(
    mutation HullMutation,
    occurredAt time.Time,
) (HullChange, error)
```

### 7.3. Terminal behavior

When Hull reaches zero:

```text
Voyage status → FAILED
completed_at → authoritative transaction time
future daily progression → blocked
unclaimed reward behavior → defined by reward policy
```

The default MVP policy is:

- Persist the result that caused failure.
- Do not generate a normal continuation reward.
- Allow the frontend to show the final result and Voyage summary.
- Do not open the next preparation state.

---

## 8. Hull sources and reason taxonomy

Use a controlled enum for source categories.

```go
type HullSourceType string

const (
    HullSourceSettlement      HullSourceType = "SETTLEMENT"
    HullSourceDailyObjective  HullSourceType = "DAILY_OBJECTIVE"
    HullSourceReward          HullSourceType = "REWARD"
    HullSourceRelic           HullSourceType = "RELIC"
    HullSourceSynergy         HullSourceType = "SYNERGY"
    HullSourceBoss            HullSourceType = "BOSS"
    HullSourceEvent           HullSourceType = "EVENT"
    HullSourceVoyageStart     HullSourceType = "VOYAGE_START"
    HullSourceAdminRepair     HullSourceType = "ADMIN_REPAIR"
)
```

Ledger entry types should distinguish the effect direction:

```go
type LedgerEntryType string

const (
    LedgerEntryStartingHull  LedgerEntryType = "STARTING_HULL"
    LedgerEntryHullDamage    LedgerEntryType = "HULL_DAMAGE"
    LedgerEntryHullHeal      LedgerEntryType = "HULL_HEAL"
    LedgerEntryMaxHullChange LedgerEntryType = "MAX_HULL_CHANGE"
)
```

`reason_key` should be a stable localization or explanation key, for example:

```text
settlement.score_deficit_damage
synergy.harbor.objective_heal
reward.restore_hull
boss.red_leviathan.pressure
admin.hull_reconciliation
```

Avoid arbitrary player-facing text in domain rows.

---

## 9. Damage and healing policy

### 9.1. Settlement output contract

The settlement engine returns a requested Hull delta and detailed evidence. It does not directly update Voyage state.

```go
type HullEffectOutput struct {
    RequestedDelta int32
    PolicyVersion  string
    ReasonKey      string
    Inputs         HullEffectInputs
    Contributions  []HullContribution
}

type HullEffectInputs struct {
    DailyScoreMicros     int64
    WinThresholdMicros   int64
    SurvivalThresholdMicros int64
    ScoreDeficitMicros   int64
    DifficultyMultiplier int32
}

type HullContribution struct {
    SourceType     string
    SourceKey      string
    Amount         int32
    ExplanationKey string
    Sequence       int16
}
```

### 9.2. Baseline damage curve

The source gameplay rule is:

```text
If Daily Score is below the threshold:
Damage = distance to threshold × difficulty multiplier
```

The exact conversion and rounding must be versioned.

Recommended baseline content contract:

```json
{
  "policyKey": "linear-score-deficit-v1",
  "scoreUnitMicrosPerHull": 100000,
  "difficultyMultiplierBps": 10000,
  "minimumDamage": 1,
  "maximumDamagePerTide": 30,
  "winHealing": 0,
  "maximumHealingPerTide": 10,
  "roundingMode": "CEIL_DAMAGE_FLOOR_HEAL"
}
```

Example calculation:

```text
score_deficit = max(0, win_threshold - daily_score)
raw_damage = score_deficit / score_unit_per_hull
damage = ceil(raw_damage × difficulty_multiplier)
damage = clamp(damage, minimum_damage, maximum_damage_per_tide)
hull_delta = -damage
```

### 9.3. Healing

Healing may come from multiple rules. Aggregate all healing and damage contributions in deterministic rule order, then emit one net requested Hull delta for the Voyage transaction.

Example:

```text
Score deficit damage       -12
Harbor synergy healing      +3
Daily objective healing     +2
Boss pressure penalty       -4
--------------------------------
Requested Hull delta       -11
```

Persist each contribution for explainability and the net Hull delta for balance application.

### 9.4. Caps

Apply caps in two stages:

1. Rule-level caps inside the versioned settlement policy.
2. Aggregate-level bounds in the Hull value object.

The aggregate-level bounds are always authoritative.

---

## 10. Persistence model

### 10.1. Voyages table

Recommended canonical columns:

```sql
ALTER TABLE voyages
    ADD COLUMN current_hull integer,
    ADD COLUMN max_hull integer;
```

Final constraints:

```sql
ALTER TABLE voyages
    ALTER COLUMN current_hull SET NOT NULL,
    ALTER COLUMN max_hull SET NOT NULL,
    ADD CONSTRAINT voyages_max_hull_positive
        CHECK (max_hull > 0),
    ADD CONSTRAINT voyages_current_hull_bounds
        CHECK (current_hull >= 0 AND current_hull <= max_hull);
```

Recommended complete relevant shape:

```sql
CREATE TABLE voyages (
    id uuid PRIMARY KEY,
    player_id uuid NOT NULL REFERENCES players(id),
    status text NOT NULL,
    voyage_definition_version_id uuid NOT NULL,
    current_day_number smallint NOT NULL,
    current_hull integer NOT NULL,
    max_hull integer NOT NULL,
    supplies integer NOT NULL,
    score_micros bigint NOT NULL,
    started_at timestamptz NOT NULL,
    completed_at timestamptz,
    row_version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT voyages_max_hull_positive CHECK (max_hull > 0),
    CONSTRAINT voyages_current_hull_bounds
        CHECK (current_hull >= 0 AND current_hull <= max_hull),
    CONSTRAINT voyages_active_hull_check
        CHECK (status <> 'ACTIVE' OR current_hull > 0)
);
```

The final active-Hull constraint should be added only after all code paths atomically transition zero-Hull Voyages to `FAILED`.

### 10.2. Voyage ledger entries

```sql
CREATE TABLE voyage_ledger_entries (
    id uuid PRIMARY KEY,
    voyage_id uuid NOT NULL REFERENCES voyages(id),
    daily_tide_id uuid REFERENCES daily_tides(id),
    entry_type text NOT NULL,
    supplies_delta integer NOT NULL DEFAULT 0,
    hull_delta integer NOT NULL DEFAULT 0,
    max_hull_delta integer NOT NULL DEFAULT 0,
    score_delta_micros bigint NOT NULL DEFAULT 0,
    source_type text NOT NULL,
    source_id text NOT NULL,
    reason_key text NOT NULL,
    idempotency_key text NOT NULL,
    balance_hull_before integer,
    balance_hull_after integer,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT voyage_ledger_non_empty CHECK (
        supplies_delta <> 0 OR
        hull_delta <> 0 OR
        max_hull_delta <> 0 OR
        score_delta_micros <> 0 OR
        entry_type = 'STARTING_HULL'
    ),
    CONSTRAINT voyage_ledger_hull_balance_pair CHECK (
        (balance_hull_before IS NULL AND balance_hull_after IS NULL)
        OR
        (balance_hull_before IS NOT NULL AND balance_hull_after IS NOT NULL)
    ),
    UNIQUE (voyage_id, idempotency_key)
);
```

For Hull entries, require `balance_hull_before` and `balance_hull_after` at the application layer. A later migration may add a conditional database constraint based on `entry_type`.

### 10.3. Daily results

```sql
ALTER TABLE daily_results
    ADD COLUMN hull_before integer NOT NULL,
    ADD COLUMN requested_hull_delta integer NOT NULL,
    ADD COLUMN effective_hull_delta integer NOT NULL,
    ADD COLUMN hull_after integer NOT NULL,
    ADD COLUMN hull_policy_version text NOT NULL;
```

Constraints:

```sql
ALTER TABLE daily_results
    ADD CONSTRAINT daily_results_hull_non_negative
        CHECK (hull_before >= 0 AND hull_after >= 0),
    ADD CONSTRAINT daily_results_effective_delta_matches
        CHECK (hull_after - hull_before = effective_hull_delta);
```

Store requested and effective deltas separately so capped healing or lethal over-damage remains explainable.

### 10.4. Lineup snapshots

```sql
ALTER TABLE lineup_snapshots
    ADD COLUMN hull_before integer NOT NULL,
    ADD COLUMN max_hull_before integer NOT NULL;
```

These values are settlement inputs and replay evidence. They are not updated after lock.

### 10.5. Hull contribution rows

Use the general score breakdown system when possible. If Hull explanations require dedicated querying, add:

```sql
CREATE TABLE hull_effect_contributions (
    id uuid PRIMARY KEY,
    daily_result_id uuid NOT NULL REFERENCES daily_results(id),
    sequence smallint NOT NULL,
    source_type text NOT NULL,
    source_key text NOT NULL,
    requested_delta integer NOT NULL,
    explanation_key text NOT NULL,
    explanation_params jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (daily_result_id, sequence)
);
```

Prefer reusing a generic `score_breakdown_entries` table if it supports a `HULL` component and signed integer amount.

---

## 11. Repository interfaces

The Voyage repository must support row locking.

```go
type Repository interface {
    GetByID(ctx context.Context, id VoyageID) (Voyage, error)
    GetByIDForUpdate(ctx context.Context, tx Tx, id VoyageID) (Voyage, error)
    Update(ctx context.Context, tx Tx, voyage Voyage, expectedRowVersion int64) error
}
```

Ledger repository:

```go
type LedgerRepository interface {
    Insert(ctx context.Context, tx Tx, entry LedgerEntry) error
    ExistsByIdempotencyKey(
        ctx context.Context,
        tx Tx,
        voyageID VoyageID,
        key string,
    ) (bool, error)
}
```

Daily result repository:

```go
type DailyResultRepository interface {
    Insert(ctx context.Context, tx Tx, result DailyResult) error
    GetByVoyageAndTide(
        ctx context.Context,
        voyageID VoyageID,
        dailyTideID DailyTideID,
    ) (DailyResult, error)
}
```

Transactions should be orchestrated by an application service, not inside repository implementations.

---

## 12. Application service

Recommended package:

```text
internal/voyage/application
```

### 12.1. Generic Hull mutation command

```go
type ApplyHullCommand struct {
    VoyageID      voyage.VoyageID
    DailyTideID   *daily.TideID
    RequestedDelta int32
    SourceType    voyage.HullSourceType
    SourceID      string
    ReasonKey     string
    IdempotencyKey string
    OccurredAt    time.Time
    Metadata      map[string]any
}

type ApplyHullResult struct {
    VoyageID       voyage.VoyageID
    VoyageStatus   voyage.Status
    HullChange     voyage.HullChange
    RowVersion     int64
}
```

### 12.2. Service algorithm

```text
1. Validate command shape.
2. Start database transaction.
3. Check ledger idempotency key.
4. Load Voyage FOR UPDATE.
5. Verify Voyage is active for ordinary gameplay mutations.
6. Apply Hull mutation through aggregate method.
7. Insert immutable ledger entry using effective delta.
8. Update Voyage Hull, status, completion time and row version.
9. Commit.
10. Return canonical result.
```

Pseudocode:

```go
func (s *Service) ApplyHull(
    ctx context.Context,
    cmd ApplyHullCommand,
) (ApplyHullResult, error) {
    var result ApplyHullResult

    err := s.tx.Within(ctx, func(tx Tx) error {
        exists, err := s.ledger.ExistsByIdempotencyKey(
            ctx, tx, cmd.VoyageID, cmd.IdempotencyKey,
        )
        if err != nil {
            return err
        }
        if exists {
            stored, err := s.loadPriorHullResult(ctx, tx, cmd)
            if err != nil {
                return err
            }
            result = stored
            return nil
        }

        v, err := s.voyages.GetByIDForUpdate(ctx, tx, cmd.VoyageID)
        if err != nil {
            return err
        }

        expectedVersion := v.RowVersion
        change, err := v.ApplyHullMutation(
            voyage.HullMutation{
                Delta:      cmd.RequestedDelta,
                SourceType: cmd.SourceType,
                SourceID:   cmd.SourceID,
                ReasonKey:  cmd.ReasonKey,
            },
            cmd.OccurredAt,
        )
        if err != nil {
            return err
        }

        entry := NewHullLedgerEntry(v, cmd, change)
        if err := s.ledger.Insert(ctx, tx, entry); err != nil {
            return err
        }

        v.RowVersion++
        if err := s.voyages.Update(ctx, tx, v, expectedVersion); err != nil {
            return err
        }

        result = ApplyHullResult{
            VoyageID:     v.ID,
            VoyageStatus: v.Status,
            HullChange:   change,
            RowVersion:   v.RowVersion,
        }
        return nil
    })

    return result, err
}
```

### 12.3. No-op policy

A requested Hull mutation may become a no-op after clamping, such as healing at full Hull.

Recommended MVP behavior:

- Settlement may record a zero-effective contribution in result breakdowns.
- Do not insert a balance ledger entry when effective delta is zero.
- Do persist the daily result's requested and effective deltas.
- Reward claims that restore Hull at full Hull should be rejected before claim unless the reward explicitly permits wasting the effect.

---

## 13. Settlement integration

Hull application during daily settlement must occur in the same transaction as the daily result and other Voyage effects.

### 13.1. Transaction sequence

```text
1. Load immutable settlement inputs.
2. Run pure settlement engine outside the write transaction.
3. Start player settlement transaction.
4. Lock Voyage row FOR UPDATE.
5. Confirm daily result does not already exist.
6. Confirm snapshot Hull matches expected settlement input policy.
7. Apply requested Hull delta to Voyage aggregate.
8. Insert daily result with before/requested/effective/after values.
9. Insert score and Hull contribution rows.
10. Insert Hull, Supplies and score ledger entries.
11. Update Voyage materialized balances and status.
12. Generate reward set only when policy allows.
13. Update player daily phase.
14. Commit.
```

### 13.2. Snapshot consistency

At lock time, store:

```text
hull_before
max_hull_before
```

At settlement time, the authoritative Voyage Hull may differ from the lock snapshot only if the game permits between-lock effects.

MVP recommendation:

- Do not permit Hull mutations between Fleet lock and settlement.
- Assert `voyage.current_hull == snapshot.hull_before` before applying the daily result.
- If the values differ, mark settlement input invalid and require operator inspection or deterministic replay.

If future events can change Hull after lock, introduce an ordered ledger cutoff rather than silently relaxing this assertion.

### 13.3. Idempotency

Use a stable settlement idempotency key:

```text
settlement:{daily_tide_id}:{voyage_id}:hull
```

The unique daily result constraint and unique Voyage ledger idempotency key together prevent duplicate Hull effects.

### 13.4. Partial failure

If any of these writes fail, roll back all player settlement effects:

- Daily result.
- Hull contributions.
- Ledger entries.
- Voyage balance update.
- Voyage failure transition.
- Reward generation.
- Player daily phase update.

Never commit the Hull balance without its result and ledger evidence.

---

## 14. Reward integration

Hull restoration rewards must use the same aggregate mutation rules.

### 14.1. Reward option

```json
{
  "type": "RESTORE_HULL",
  "amount": 8,
  "preview": {
    "currentHull": 72,
    "maximumHull": 100,
    "effectiveRestore": 8,
    "resultingHull": 80
  }
}
```

The preview is informational. The claim transaction recalculates the effective change from authoritative state.

### 14.2. Claim transaction

```text
1. Lock reward set.
2. Verify unclaimed option.
3. Lock Voyage.
4. Apply healing to Voyage aggregate.
5. Insert reward claim.
6. Insert Hull ledger entry.
7. Update Voyage.
8. Advance daily state.
9. Commit.
```

### 14.3. Full-Hull policy

For MVP, do not generate a pure Hull-restoration reward when Hull is already full. If state changes between generation and claim and the reward becomes partially capped, apply the effective healing and persist both requested and effective amounts.

---

## 15. Voyage creation

Voyage creation initializes Hull transactionally.

### 15.1. Source of values

Starting and maximum Hull come from the immutable Voyage definition version.

```go
type VoyageDefinitionVersion struct {
    ID           VoyageDefinitionVersionID
    StartingHull int32
    MaximumHull  int32
}
```

Validation:

```text
maximum_hull > 0
0 < starting_hull <= maximum_hull
```

### 15.2. Creation transaction

```text
1. Resolve Voyage definition version.
2. Construct Hull value object.
3. Insert Voyage.
4. Insert STARTING_HULL ledger entry.
5. Grant starter content and create daily state.
6. Commit.
```

Starting ledger example:

```text
entry_type: STARTING_HULL
hull_delta: +100
balance_hull_before: 0
balance_hull_after: 100
source_type: VOYAGE_START
```

This entry establishes the reconciliation baseline.

---

## 16. API contracts

### 16.1. Daily context

Recommended response vocabulary:

```json
{
  "voyage": {
    "id": "voy_123",
    "status": "ACTIVE",
    "dayNumber": 3,
    "hull": {
      "current": 82,
      "maximum": 100,
      "state": "STABLE"
    },
    "supplies": 11,
    "score": 245
  }
}
```

Compatibility option during migration:

```json
{
  "fundHealth": 82,
  "maxFundHealth": 100
}
```

Do not indefinitely expose both shapes. Add new Hull fields, migrate the frontend, then deprecate legacy names through a documented API version.

### 16.2. Daily result

```json
{
  "outcome": "LOSS",
  "dailyScore": 184,
  "hull": {
    "before": 82,
    "requestedDelta": -14,
    "effectiveDelta": -14,
    "after": 68,
    "stateAfter": "DAMAGED",
    "contributions": [
      {
        "sourceType": "SCORE_DEFICIT",
        "sourceKey": "linear-score-deficit-v1",
        "delta": -12,
        "explanationKey": "settlement.score_deficit_damage"
      },
      {
        "sourceType": "BOSS",
        "sourceKey": "red_leviathan_pressure",
        "delta": -4,
        "explanationKey": "boss.red_leviathan.pressure"
      },
      {
        "sourceType": "SYNERGY",
        "sourceKey": "harbor_ii",
        "delta": 2,
        "explanationKey": "synergy.harbor.damage_reduction"
      }
    ]
  }
}
```

### 16.3. Mutation endpoints

There is no public generic `POST /hull` endpoint.

Hull changes occur through domain-specific commands:

- Daily settlement worker.
- Reward claim endpoint.
- Event choice endpoint when added.
- Admin repair command with explicit audit reason.

The client must never submit `hullDelta`, `currentHull` or `maximumHull` as an authoritative gameplay mutation.

### 16.4. Error codes

```text
VOYAGE_NOT_ACTIVE
HULL_MUTATION_INVALID
HULL_REWARD_NO_EFFECT
SETTLEMENT_HULL_MISMATCH
VERSION_CONFLICT
ALREADY_APPLIED
ADMIN_REASON_REQUIRED
```

Public responses should map internal operational errors to safe player-facing messages.

---

## 17. Frontend behavior contract

The frontend:

- Displays Hull returned by the backend.
- May derive visual meter percentage from current and maximum values.
- Must not predict authoritative post-settlement Hull.
- Shows `requestedDelta` and `effectiveDelta` when a cap matters.
- Displays critical status with text and icon, not color alone.
- Refreshes canonical daily context after any reward that affects Hull.
- Routes to Voyage failure summary when status becomes `FAILED`.

Suggested labels:

```text
Hull
Hull restored
Hull damaged
Critical Hull
Voyage lost
```

Avoid displaying the legacy term `Fund Health` after migration is complete.

---

## 18. Concurrency controls

### 18.1. Row lock

Every authoritative Hull mutation must load the Voyage row using:

```sql
SELECT ...
FROM voyages
WHERE id = $1
FOR UPDATE;
```

### 18.2. Optimistic version

The Voyage `row_version` protects updates that do not use the same row-locking transaction boundary.

```sql
UPDATE voyages
SET current_hull = $2,
    max_hull = $3,
    status = $4,
    completed_at = $5,
    row_version = row_version + 1,
    updated_at = now()
WHERE id = $1
  AND row_version = $6;
```

Zero affected rows returns `VERSION_CONFLICT`.

### 18.3. Lock order

Use a consistent lock order to avoid deadlocks:

```text
1. Player daily state or reward set, depending on command root.
2. Voyage.
3. Daily result uniqueness check or insert.
4. Ledger rows.
5. Reward rows.
```

Document and use the same order across settlement and reward claims.

### 18.4. Concurrent examples

#### Settlement and reward claim

Normally impossible because daily phase controls prevent claiming before result generation. Database phase checks and row locks still enforce this.

#### Two reward claims

One transaction claims the reward set. The other observes `CLAIMED` and returns the canonical claimed result without applying Hull twice.

#### Settlement retry

Unique result and ledger keys return the already persisted result rather than applying damage again.

---

## 19. Audit and replay

A Hull outcome must be reconstructable from stored evidence.

Required replay inputs:

- Voyage Hull and maximum Hull at lock.
- Daily score and threshold inputs.
- Difficulty and Hull policy version.
- Keeper, Synergy, Relic, Strategy, Modifier and Objective contributions.
- Requested Hull delta.
- Effective Hull delta.
- Ledger entry.
- Voyage state after settlement.

Replay validation:

```text
recomputed requested delta == stored requested delta
clamp(before + requested delta) == stored after
stored after - stored before == stored effective delta
ledger hull delta == stored effective delta
voyage balance after transaction == stored after
```

Replay mismatch must emit `SETTLEMENT_REPLAY_MISMATCH` and never silently repair production state.

---

## 20. Reconciliation

Provide an administrative read-only reconciliation command:

```text
tidekeepers-admin reconcile-hull --voyage-id <id>
```

It should calculate:

```text
expected_hull = starting_hull + sum(effective hull ledger deltas)
```

Then compare against `voyages.current_hull`.

Output:

```json
{
  "voyageId": "voy_123",
  "materializedHull": 68,
  "ledgerHull": 68,
  "matches": true,
  "lastLedgerEntryId": "vle_..."
}
```

A repair command must be separate:

```text
tidekeepers-admin repair-hull \
  --voyage-id <id> \
  --target <value> \
  --reason "operator explanation" \
  --dry-run
```

A repair:

- Requires operator identity.
- Requires a reason.
- Creates an `ADMIN_REPAIR` audit event.
- Creates a Hull ledger entry.
- Uses the aggregate and normal bounds.
- Never rewrites or deletes historical ledger entries.

---

## 21. Observability

### 21.1. Structured logs

Include:

```text
voyage_id
player_id
daily_tide_id
source_type
source_id
requested_hull_delta
effective_hull_delta
hull_before
hull_after
voyage_status
idempotency_key
request_id or job_id
```

Do not log player session credentials or raw provider secrets.

### 21.2. Metrics

```text
hull_damage_total
hull_healing_total
hull_mutation_total{source_type,result}
hull_mutation_capped_total{direction}
voyage_failed_total{cause}
hull_reconciliation_mismatch_total
settlement_hull_mismatch_total
```

Suggested histogram:

```text
hull_effective_delta_absolute
```

### 21.3. Alerts

Alert when:

- Hull reconciliation mismatch is greater than zero.
- Settlement snapshot Hull mismatches current Voyage Hull.
- Active Voyage rows have zero Hull.
- Failed Voyage rows have positive Hull after a Hull-caused failure transaction.
- Repeated idempotency conflicts indicate a worker bug.

---

## 22. Security and authorization

- Only authenticated Voyage owners may read their Hull state.
- Public APIs cannot directly mutate Hull.
- Worker service identity may apply settlement effects.
- Reward claims authorize Voyage ownership before locking rows.
- Admin repairs require privileged operator authorization and audit logging.
- All SQL uses parameters.
- Hull metadata JSON must be validated and size-limited.
- Reason keys are controlled values, not arbitrary HTML or executable content.

---

## 23. Migration from `fund_health`

The current backend specification uses `fund_health` and `max_fund_health`. Use an expand-contract migration.

### Phase 1 — Expand

Add canonical columns:

```sql
ALTER TABLE voyages
    ADD COLUMN current_hull integer,
    ADD COLUMN max_hull integer;
```

Backfill:

```sql
UPDATE voyages
SET current_hull = fund_health,
    max_hull = max_fund_health
WHERE current_hull IS NULL;
```

Add temporary consistency constraint or migration verification query.

### Phase 2 — Dual read compatibility

- New domain code reads canonical Hull fields.
- API may continue mapping them to legacy response names.
- Do not allow two independent write paths.

### Phase 3 — API migration

- Add `hull.current`, `hull.maximum` and `hull.state`.
- Update generated frontend client.
- Update UI terminology.
- Deprecate legacy fields.

### Phase 4 — Contract

After all deployed versions use canonical fields:

```sql
ALTER TABLE voyages
    DROP COLUMN fund_health,
    DROP COLUMN max_fund_health;
```

If database column renaming is preferred over duplicate columns, still use an expand-contract release sequence to keep old application versions compatible during rolling deployment.

---

## 24. Package structure

Recommended files:

```text
internal/voyage/
├── voyage.go
├── status.go
├── hull.go
├── hull_state.go
├── hull_source.go
├── errors.go
├── repository.go
└── application/
    ├── service.go
    ├── apply_hull.go
    └── reconcile_hull.go

internal/settlement/
├── engine/
│   ├── hull_effect.go
│   └── hull_policy.go
└── breakdown/
    └── hull_contribution.go

internal/database/query/
├── voyages.sql
├── voyage_ledger_entries.sql
└── daily_results.sql

db/migrations/
├── *_add_hull_columns.sql
├── *_add_hull_ledger_fields.sql
└── *_add_daily_result_hull_evidence.sql

test/
├── integration/hull_test.go
├── replay/hull_replay_test.go
└── fixtures/hull/
```

Keep formula policy in settlement and balance invariants in voyage. This prevents settlement rules from bypassing aggregate constraints.

---

## 25. Testing strategy

### 25.1. Unit tests for Hull value object

Required cases:

```text
construct valid Hull
reject maximum <= 0
reject current < 0
reject current > maximum
apply ordinary damage
apply lethal damage and clamp to zero
apply ordinary healing
apply healing above maximum and clamp
derive Stable state
derive Damaged state
derive Critical state
derive Destroyed state
requested delta differs from effective delta when capped
integer intermediate does not overflow
```

Example table-driven test:

```go
func TestHullApplyDelta(t *testing.T) {
    tests := []struct {
        name      string
        current   int32
        maximum   int32
        delta     int32
        wantAfter int32
        wantDelta int32
        wantCapped bool
    }{
        {"damage", 80, 100, -12, 68, -12, false},
        {"lethal damage", 8, 100, -12, 0, -8, true},
        {"healing", 70, 100, 10, 80, 10, false},
        {"capped healing", 95, 100, 10, 100, 5, true},
    }
    // Execute cases.
}
```

### 25.2. Aggregate tests

```text
active Voyage accepts damage
active Voyage accepts healing
zero Hull transitions Voyage to FAILED
terminal Voyage rejects ordinary mutations
zero requested delta is rejected
capped zero-effective healing follows no-op policy
failure completion timestamp is supplied by caller
```

### 25.3. Settlement engine tests

```text
score above threshold produces no baseline damage
score below threshold produces deterministic damage
minimum and maximum damage caps apply
healing and damage contributions follow rule order
identical inputs produce identical Hull output
unknown Hull policy fails settlement
rounding mode is exact
```

### 25.4. Integration tests

Run against PostgreSQL:

```text
Hull mutation inserts ledger and updates Voyage atomically
transaction rollback leaves neither balance nor ledger entry
same idempotency key applies once
concurrent mutations serialize through Voyage row lock
settlement retry does not duplicate damage
reward claim race heals once
lethal settlement persists result and FAILED status together
check constraints reject invalid direct SQL values
snapshot mismatch prevents settlement
ledger reconciliation matches materialized balance
```

### 25.5. Property tests

For arbitrary valid Hull and delta:

```text
0 <= after <= maximum
effective_delta == after - before
applying the same deterministic sequence produces the same final Hull
no sequence of domain mutations can produce negative Hull
no sequence of domain mutations can exceed maximum Hull
```

### 25.6. Replay fixtures

Minimum fixtures:

```text
ordinary loss
lethal loss
win with no Hull change
healing reward
healing capped at maximum
Harbor mitigation
boss pressure plus mitigation
admin repair
settlement retry
snapshot Hull mismatch
```

---

## 26. Failure handling

### Invalid settlement policy

- Do not apply Hull changes.
- Do not persist player result effects.
- Record `RULE_CONFIGURATION_INVALID`.
- Leave player daily state pending for repair.

### Snapshot mismatch

- Do not guess which Hull value is correct.
- Record `SETTLEMENT_HULL_MISMATCH`.
- Keep the player unsettled.
- Alert operators.

### Ledger insert failure

- Roll back Voyage update and all related effects.

### Voyage update conflict

- Roll back transaction.
- Retry only through bounded worker retry policy.
- Do not recalculate with a different rule version.

### Result already exists

- Load and return the existing result.
- Verify its ledger entry exists.
- Do not apply Hull again.

---

## 27. Implementation sequence

### Milestone 1 — Domain core

- Add `Hull` value object.
- Add Hull invariants and derived state.
- Embed Hull in `Voyage`.
- Add aggregate mutation and failure transition tests.

### Milestone 2 — Persistence

- Add or rename Hull columns.
- Add constraints.
- Add Hull ledger fields.
- Add daily-result and lineup-snapshot evidence fields.
- Generate `sqlc` queries and models.

### Milestone 3 — Application service

- Implement transactional Hull mutation service.
- Add idempotency behavior.
- Add reconciliation query.
- Add structured logs and metrics.

### Milestone 4 — Settlement

- Implement versioned Hull damage policy.
- Emit Hull contributions.
- Apply Hull inside settlement transaction.
- Add replay fixtures.

### Milestone 5 — Rewards and API

- Implement Hull restoration reward.
- Add canonical Hull response object.
- Migrate frontend terminology and types.
- Deprecate legacy `fundHealth` fields.

### Milestone 6 — Hardening

- Add concurrency tests.
- Add admin reconciliation and repair commands.
- Add production invariant checks and alerts.
- Complete expand-contract migration.

---

## 28. Acceptance criteria

Hull domain implementation is complete when:

1. Every Voyage has valid current and maximum Hull.
2. Hull is owned and mutated only through the Voyage aggregate.
3. Damage and healing are clamped to domain bounds.
4. Reaching zero Hull atomically changes Voyage status to `FAILED`.
5. Every effective Hull change creates exactly one immutable ledger entry.
6. Daily results persist Hull before, requested delta, effective delta and after values.
7. Fleet lock snapshots preserve Hull inputs for settlement replay.
8. Settlement applies Hull in the same transaction as result, ledger and Voyage progression.
9. Settlement retries cannot duplicate Hull damage or healing.
10. Reward claim races cannot apply healing twice.
11. The API exposes canonical Hull state without accepting authoritative Hull mutations from the client.
12. Materialized Hull can be reconciled against the ledger.
13. Any Hull result can be deterministically replayed from stored evidence.
14. Invalid policy, snapshot mismatch or persistence failure leaves player state unchanged.
15. Database constraints prevent Hull outside `[0, maximum]`.
16. Unit, integration, property and replay tests cover damage, healing, caps, failure and concurrency.

---

## 29. Decisions deferred

The following are intentionally deferred until required by published content:

- Hull armor or temporary shield as a separate resource.
- Multiple Hull sections or ship components.
- Permanent meta-progression increases to maximum Hull.
- Reviving a failed Voyage.
- Damage-over-time effects between daily settlements.
- Player-versus-player Hull comparison.
- Client-visible exact damage forecast before lock.
- Maximum Hull mutation by Relics or Tidekeeper classes.

Future features must preserve the same principles: Voyage ownership, immutable evidence, deterministic calculation, transactional application and replayability.

---

## 30. Final implementation decision

Use the following structure as the authoritative design:

```text
Voyage.Hull
├── Current              authoritative balance
├── Maximum              authoritative cap
└── State()              derived display value

LineupSnapshot
├── HullBefore           immutable lock evidence
└── MaxHullBefore        immutable lock evidence

SettlementOutput
├── RequestedHullDelta   pure engine output
└── Contributions        explainability evidence

DailyResult
├── HullBefore
├── RequestedHullDelta
├── EffectiveHullDelta
└── HullAfter

VoyageLedgerEntry
├── HullDelta            immutable applied effect
├── BalanceHullBefore
└── BalanceHullAfter
```

Hull is a **Voyage-owned value object**, not an independent aggregate and not a field owned by the daily state. This boundary is the basis for all implementation, transaction and API decisions in this specification.
