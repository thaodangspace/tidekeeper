package voyage

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

const (
	idempotencyScopeCreate  = "voyage_create"
	idempotencyScopeAbandon = "voyage_abandon"
	defaultDefinitionKey    = "standard"
	ledgerEntryHull         = "START_HULL"
	ledgerEntrySupplies     = "START_SUPPLIES"
	lifecycleEventCreated   = "CREATED"
	lifecycleEventAbandoned = "ABANDONED"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Create(ctx context.Context, playerID, idempotencyKey string) (*VoyageResponse, error) {
	if !validKeyFormat(idempotencyKey) {
		return nil, ErrInvalidIdempotencyKey
	}

	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	hashInput := sha256.Sum256([]byte(idempotencyScopeCreate))
	requestHash := hashInput[:]

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: begin tx: %w", ErrServiceUnavailable, err)
	}
	defer tx.Rollback(ctx)
	q := query.New(tx)

	result, err := s.claimKey(ctx, q, pid, idempotencyScopeCreate, idempotencyKey, requestHash)
	if err != nil {
		return nil, err
	}
	if !result.isNew {
		return replayVoyageResponse(result.key.ResponseJson, result.key.ResponseStatus)
	}

	_, err = q.LockPlayerRow(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("%w: lock player: %w", ErrServiceUnavailable, err)
	}
	_, checkErr := q.GetActiveVoyageForPlayer(ctx, pid)
	if checkErr == nil {
		return nil, ErrActiveVoyageExists
	}
	if !errors.Is(checkErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: check active voyage: %w", ErrInternal, checkErr)
	}

	def, err := q.GetPublishedVoyageDefinition(ctx, defaultDefinitionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitializationUnavailable, fmt.Errorf("get definition: %w", err))
	}

	today := pgtype.Date{Time: nowDate(), Valid: true}
	tide, err := q.GetOpenDailyTide(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInitializationUnavailable, fmt.Errorf("get daily tide: %w", err))
	}

	starterKeepers, err := q.GetVoyageDefinitionStarterKeepers(ctx, def.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: get starter keepers: %w", ErrInternal, err)
	}

	initialOffers, err := q.GetVoyageDefinitionInitialOffers(ctx, def.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: get initial offers: %w", ErrInternal, err)
	}

	voyageID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	voyagePublicID, err := newVoyagePublicID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	voyageRow, err := q.InsertVoyage(ctx, query.InsertVoyageParams{
		ID:                        voyageID,
		PublicID:                  voyagePublicID,
		PlayerID:                  pid,
		VoyageDefinitionVersionID: def.ID,
		FundHealth:                def.StartingHull,
		MaxFundHealth:             def.StartingHull,
		Capital:                   def.StartingSupplies,
	})
	if err != nil {
		if isActiveVoyageConflict(err) {
			return nil, ErrActiveVoyageExists
		}
		return nil, fmt.Errorf("%w: insert voyage: %w", ErrInternal, err)
	}

	if def.StartingHull > 0 {
		if err := insertLedgerEntry(ctx, q, voyageID, voyagePublicID, ledgerEntryHull, def.StartingHull); err != nil {
			return nil, err
		}
	}
	if def.StartingSupplies > 0 {
		if err := insertLedgerEntry(ctx, q, voyageID, voyagePublicID, ledgerEntrySupplies, def.StartingSupplies); err != nil {
			return nil, err
		}
	}

	instances := make([]keeperInstance, len(starterKeepers))
	for i, sk := range starterKeepers {
		kid, err := newUUID()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInternal, err)
		}
		kprPublicID, err := newKeeperPublicID()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInternal, err)
		}
		instances[i] = keeperInstance{
			id:       kid,
			publicID: kprPublicID,
			defID:    sk.KeeperDefinitionVersionID,
			key:      sk.KeeperKey,
			name:     sk.Name,
			sector:   sk.SectorKey,
			role:     sk.RoleKey,
			rarity:   sk.RarityKey,
		}
		if err := q.InsertKeeperInstance(ctx, query.InsertKeeperInstanceParams{
			ID:                        kid,
			PublicID:                  kprPublicID,
			PlayerID:                  pid,
			KeeperDefinitionVersionID: sk.KeeperDefinitionVersionID,
		}); err != nil {
			return nil, fmt.Errorf("%w: insert keeper instance %d: %w", ErrInternal, i, err)
		}
	}

	stateID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	projection, err := buildInitialProjection(def, instances, initialOffers)
	if err != nil {
		return nil, fmt.Errorf("%w: build projection: %w", ErrInternal, err)
	}
	if err := q.InsertPlayerDailyStateWithView(ctx, query.InsertPlayerDailyStateWithViewParams{
		Projection:  projection,
		StateID:     stateID,
		PlayerID:    pid,
		VoyageID:    voyageID,
		DailyTideID: tide.ID,
	}); err != nil {
		return nil, fmt.Errorf("%w: insert daily state: %w", ErrInternal, err)
	}

	eventID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	eventPublicID, err := newEventPublicID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if err := q.InsertVoyageLifecycleEvent(ctx, query.InsertVoyageLifecycleEventParams{
		ID:              eventID,
		PublicID:        eventPublicID,
		VoyageID:        voyageID,
		EventType:       lifecycleEventCreated,
		ResultingStatus: string(StatusActive),
	}); err != nil {
		return nil, fmt.Errorf("%w: insert lifecycle event: %w", ErrInternal, err)
	}

	if err := q.UpdatePlayerCurrentVoyage(ctx, query.UpdatePlayerCurrentVoyageParams{
		VoyageID: voyageID,
		PlayerID: pid,
	}); err != nil {
		return nil, fmt.Errorf("%w: update player: %w", ErrInternal, err)
	}

	resp := makeResponse(voyageRow.PublicID, voyageRow.Status, voyageRow.CurrentDayNumber, voyageRow.FundHealth, voyageRow.MaxFundHealth, voyageRow.Capital, voyageRow.Score, voyageRow.RowVersion, voyageRow.StartedAt, voyageRow.CompletedAt, def)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal response: %w", ErrInternal, err)
	}
	status := int16(201)

	if err := q.CompleteIdempotencyKey(ctx, query.CompleteIdempotencyKeyParams{
		ResultVoyageID: voyageID,
		ResponseStatus: &status,
		ResponseJson:   respJSON,
		PlayerID:       pid,
		Scope:          idempotencyScopeCreate,
		Key:            idempotencyKey,
	}); err != nil {
		return nil, fmt.Errorf("%w: complete idempotency key: %w", ErrInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: commit: %w", ErrServiceUnavailable, err)
	}

	return resp, nil
}

func (s *Service) GetCurrent(ctx context.Context, playerID string) (*VoyageResponse, error) {
	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	q := query.New(s.pool)
	voyageRow, err := q.GetActiveVoyageForPlayer(ctx, pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveVoyage
		}
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	def, err := q.GetVoyageDefinitionByID(ctx, voyageRow.VoyageDefinitionVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: get definition: %w", ErrInternal, err)
	}

	return makeResponse(voyageRow.PublicID, voyageRow.Status, voyageRow.CurrentDayNumber, voyageRow.FundHealth, voyageRow.MaxFundHealth, voyageRow.Capital, voyageRow.Score, voyageRow.RowVersion, voyageRow.StartedAt, voyageRow.CompletedAt, def), nil
}

func (s *Service) GetByID(ctx context.Context, playerID, voyageID string) (*VoyageResponse, error) {
	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	q := query.New(s.pool)
	voyageRow, err := q.GetVoyageByPublicIDAndPlayerID(ctx, query.GetVoyageByPublicIDAndPlayerIDParams{
		PublicID: voyageID,
		PlayerID: pid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVoyageNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	def, err := q.GetVoyageDefinitionByID(ctx, voyageRow.VoyageDefinitionVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: get definition: %w", ErrInternal, err)
	}

	return makeResponse(voyageRow.PublicID, voyageRow.Status, voyageRow.CurrentDayNumber, voyageRow.FundHealth, voyageRow.MaxFundHealth, voyageRow.Capital, voyageRow.Score, voyageRow.RowVersion, voyageRow.StartedAt, voyageRow.CompletedAt, def), nil
}

func (s *Service) Abandon(ctx context.Context, playerID, voyageID, idempotencyKey string) (*VoyageResponse, error) {
	if !validKeyFormat(idempotencyKey) {
		return nil, ErrInvalidIdempotencyKey
	}

	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	hashInput := sha256.Sum256([]byte(idempotencyScopeAbandon + ":" + voyageID))
	requestHash := hashInput[:]

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: begin tx: %w", ErrServiceUnavailable, err)
	}
	defer tx.Rollback(ctx)
	q := query.New(tx)

	result, err := s.claimKey(ctx, q, pid, idempotencyScopeAbandon, idempotencyKey, requestHash)
	if err != nil {
		return nil, err
	}
	if !result.isNew {
		return replayVoyageResponse(result.key.ResponseJson, result.key.ResponseStatus)
	}

	voyageRow, err := q.GetVoyageByPublicIDAndPlayerID(ctx, query.GetVoyageByPublicIDAndPlayerIDParams{
		PublicID: voyageID,
		PlayerID: pid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVoyageNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	def, err := q.GetVoyageDefinitionByID(ctx, voyageRow.VoyageDefinitionVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: get definition: %w", ErrInternal, err)
	}

	if voyageRow.Status == string(StatusAbandoned) {
		resp := makeResponse(voyageRow.PublicID, voyageRow.Status, voyageRow.CurrentDayNumber, voyageRow.FundHealth, voyageRow.MaxFundHealth, voyageRow.Capital, voyageRow.Score, voyageRow.RowVersion, voyageRow.StartedAt, voyageRow.CompletedAt, def)
		respJSON, err := json.Marshal(resp)
		if err != nil {
			return nil, fmt.Errorf("%w: marshal response: %w", ErrInternal, err)
		}
		status := int16(200)
		if err := q.CompleteIdempotencyKey(ctx, query.CompleteIdempotencyKeyParams{
			ResultVoyageID: voyageRow.ID,
			ResponseStatus: &status,
			ResponseJson:   respJSON,
			PlayerID:       pid,
			Scope:          idempotencyScopeAbandon,
			Key:            idempotencyKey,
		}); err != nil {
			return nil, fmt.Errorf("%w: complete idempotency key: %w", ErrInternal, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("%w: commit: %w", ErrServiceUnavailable, err)
		}
		return resp, nil
	}

	if voyageRow.Status != string(StatusActive) {
		return nil, ErrVoyageTerminal
	}

	updated, err := q.UpdateVoyageStatus(ctx, query.UpdateVoyageStatusParams{
		Status:       string(StatusAbandoned),
		SetCompleted: true,
		ID:           voyageRow.ID,
		RowVersion:   voyageRow.RowVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: update voyage status: %w", ErrInternal, err)
	}

	eventID, err := newUUID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	eventPublicID, err := newEventPublicID()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if err := q.InsertVoyageLifecycleEvent(ctx, query.InsertVoyageLifecycleEventParams{
		ID:              eventID,
		PublicID:        eventPublicID,
		VoyageID:        voyageRow.ID,
		EventType:       lifecycleEventAbandoned,
		ResultingStatus: string(StatusAbandoned),
	}); err != nil {
		return nil, fmt.Errorf("%w: insert lifecycle event: %w", ErrInternal, err)
	}

	resp := makeResponse(updated.PublicID, updated.Status, updated.CurrentDayNumber, updated.FundHealth, updated.MaxFundHealth, updated.Capital, updated.Score, updated.RowVersion, updated.StartedAt, updated.CompletedAt, def)
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal response: %w", ErrInternal, err)
	}
	status := int16(200)
	if err := q.CompleteIdempotencyKey(ctx, query.CompleteIdempotencyKeyParams{
		ResultVoyageID: voyageRow.ID,
		ResponseStatus: &status,
		ResponseJson:   respJSON,
		PlayerID:       pid,
		Scope:          idempotencyScopeAbandon,
		Key:            idempotencyKey,
	}); err != nil {
		return nil, fmt.Errorf("%w: complete idempotency key: %w", ErrInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: commit: %w", ErrServiceUnavailable, err)
	}

	return resp, nil
}

func (s *Service) GetHistory(ctx context.Context, playerID, voyageID string) (*VoyageHistoryResponse, error) {
	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	q := query.New(s.pool)
	voyageRow, err := q.GetVoyageByPublicIDAndPlayerID(ctx, query.GetVoyageByPublicIDAndPlayerIDParams{
		PublicID: voyageID,
		PlayerID: pid,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVoyageNotFound
		}
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	events, err := q.ListLifecycleEventsForVoyage(ctx, voyageRow.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: list events: %w", ErrInternal, err)
	}

	def, err := q.GetVoyageDefinitionByID(ctx, voyageRow.VoyageDefinitionVersionID)
	if err != nil {
		return nil, fmt.Errorf("%w: get definition: %w", ErrInternal, err)
	}

	resp := makeResponse(voyageRow.PublicID, voyageRow.Status, voyageRow.CurrentDayNumber, voyageRow.FundHealth, voyageRow.MaxFundHealth, voyageRow.Capital, voyageRow.Score, voyageRow.RowVersion, voyageRow.StartedAt, voyageRow.CompletedAt, def)
	lifecycleEvents := make([]LifecycleEvent, len(events))
	for i, e := range events {
		lifecycleEvents[i] = LifecycleEvent{
			PublicID:        e.PublicID,
			EventType:       e.EventType,
			ResultingStatus: e.ResultingStatus,
			OccurredAt:      e.OccurredAt.Time.UTC(),
		}
	}

	return &VoyageHistoryResponse{
		Voyage: *resp,
		Events: lifecycleEvents,
	}, nil
}

type idempotencyClaimResult struct {
	isNew bool
	key   query.IdempotencyKey
}

func (s *Service) claimKey(ctx context.Context, q *query.Queries, playerID pgtype.UUID, scope, key string, requestHash []byte) (*idempotencyClaimResult, error) {
	inserted, err := q.ClaimIdempotencyKey(ctx, query.ClaimIdempotencyKeyParams{
		PlayerID:    playerID,
		Scope:       scope,
		Key:         key,
		RequestHash: requestHash,
	})
	if err == nil {
		return &idempotencyClaimResult{isNew: true, key: inserted}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	existing, err := q.GetIdempotencyKeyForUpdate(ctx, query.GetIdempotencyKeyForUpdateParams{
		PlayerID: playerID,
		Scope:    scope,
		Key:      key,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}

	if existing.State != "COMPLETED" {
		return nil, fmt.Errorf("%w: stale in-progress key", ErrInternal)
	}

	if !bytes.Equal(existing.RequestHash, requestHash) {
		return nil, ErrIdempotencyKeyReused
	}

	return &idempotencyClaimResult{isNew: false, key: existing}, nil
}

func insertLedgerEntry(ctx context.Context, q *query.Queries, voyageID pgtype.UUID, voyagePublicID string, entryType string, amount int32) error {
	entryID, err := newUUID()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInternal, err)
	}
	if err := q.InsertVoyageLedgerEntry(ctx, query.InsertVoyageLedgerEntryParams{
		ID:           entryID,
		VoyageID:     voyageID,
		EntryType:    entryType,
		Amount:       amount,
		BalanceAfter: amount,
		SourceType:   "VOYAGE_START",
		SourceID:     voyagePublicID,
		ReasonKey:    "voyage.create." + entryType,
	}); err != nil {
		return fmt.Errorf("%w: insert %s entry: %w", ErrInternal, entryType, err)
	}
	return nil
}

func replayVoyageResponse(responseJSON []byte, responseStatus *int16) (*VoyageResponse, error) {
	if responseStatus == nil || (*responseStatus != 201 && *responseStatus != 200) {
		return nil, fmt.Errorf("%w: unexpected replayed status", ErrInternal)
	}
	var resp VoyageResponse
	if err := json.Unmarshal(responseJSON, &resp); err != nil {
		return nil, fmt.Errorf("%w: unmarshal replayed response: %w", ErrInternal, err)
	}
	return &resp, nil
}

func makeResponse(
	publicID string,
	status string,
	dayNumber int16,
	fundHealth int32,
	maxFundHealth int32,
	capital int32,
	score pgtype.Numeric,
	rowVersion int64,
	startedAt pgtype.Timestamptz,
	completedAt pgtype.Timestamptz,
	def query.VoyageDefinitionVersion,
) *VoyageResponse {
	resp := &VoyageResponse{
		PublicID:          publicID,
		DefinitionKey:     def.DefinitionKey,
		DefinitionVersion: def.Version,
		Status:            status,
		DayNumber:         int(dayNumber),
		FundHealth:        int(fundHealth),
		MaxFundHealth:     int(maxFundHealth),
		Capital:           int(capital),
		Score:             formatNumeric(score),
		RowVersion:        rowVersion,
		StartedAt:         startedAt.Time.UTC(),
		CompletedAt:       nil,
	}
	if completedAt.Valid {
		t := completedAt.Time.UTC()
		resp.CompletedAt = &t
	}
	return resp
}

func formatNumeric(n pgtype.Numeric) string {
	if !n.Valid || n.Int == nil || n.NaN {
		return "0.0000"
	}
	if n.InfinityModifier == pgtype.NegativeInfinity || n.InfinityModifier == pgtype.Infinity {
		return "0.0000"
	}

	val := new(big.Rat).SetFrac(n.Int, big.NewInt(1))
	pow := big.NewInt(1)
	if n.Exp < 0 {
		for i := n.Exp; i < 0; i++ {
			pow.Mul(pow, big.NewInt(10))
		}
		val = val.Quo(val, new(big.Rat).SetInt(pow))
	} else if n.Exp > 0 {
		for i := int32(0); i < n.Exp; i++ {
			pow.Mul(pow, big.NewInt(10))
		}
		val = val.Mul(val, new(big.Rat).SetInt(pow))
	}

	floatVal, _ := val.Float64()
	return fmt.Sprintf("%.4f", floatVal)
}

func validKeyFormat(key string) bool {
	if len(key) < 1 || len(key) > 255 {
		return false
	}
	for i, c := range key {
		if i == 0 {
			if !isAlphaNumeric(c) {
				return false
			}
		} else {
			if !isAlphaNumeric(c) && c != '.' && c != '_' && c != ':' && c != '-' {
				return false
			}
		}
	}
	return true
}

func isAlphaNumeric(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func newUUID() (pgtype.UUID, error) {
	var value pgtype.UUID
	if _, err := rand.Read(value.Bytes[:]); err != nil {
		return pgtype.UUID{}, fmt.Errorf("generate UUID: %w", err)
	}
	value.Bytes[6] = (value.Bytes[6] & 0x0f) | 0x40
	value.Bytes[8] = (value.Bytes[8] & 0x3f) | 0x80
	value.Valid = true
	return value, nil
}

func newVoyagePublicID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate voyage public ID: %w", err)
	}
	return "voy_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func newKeeperPublicID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate keeper public ID: %w", err)
	}
	return "kpr_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func newEventPublicID() (string, error) {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate event public ID: %w", err)
	}
	return "evt_" + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func nowDate() time.Time {
	return time.Now().UTC().Truncate(24 * time.Hour)
}

func isActiveVoyageConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.ConstraintName == "voyages_one_active_per_player_idx"
	}
	return false
}

type keeperInstance struct {
	id       pgtype.UUID
	publicID string
	defID    pgtype.UUID
	key      string
	name     string
	sector   string
	role     string
	rarity   string
}

func buildInitialProjection(
	def query.VoyageDefinitionVersion,
	instances []keeperInstance,
	offers []query.GetVoyageDefinitionInitialOffersRow,
) ([]byte, error) {
	slotCount := int(def.FleetSlotCount)
	slots := make([]map[string]any, slotCount)
	for i := 0; i < slotCount; i++ {
		slots[i] = map[string]any{
			"index":  i,
			"keeper": nil,
		}
	}

	warnings := make([]map[string]any, 0)
	if slotCount > 0 {
		warnings = append(warnings, map[string]any{
			"code":     "EMPTY_SLOT",
			"severity": "WARNING",
			"message":  "All slots are empty.",
		})
	}

	inventory := make([]map[string]any, len(instances))
	for i, inst := range instances {
		inventory[i] = makeKeeperMap(inst.publicID, inst.key, inst.name, inst.sector, inst.role, inst.rarity)
	}

	shopOffers := make([]map[string]any, len(offers))
	for i, o := range offers {
		shopOffers[i] = map[string]any{
			"id": o.KeeperKey + "_offer",
			"keeper": makeKeeperMap(
				o.KeeperKey+"_preview",
				o.KeeperKey,
				o.Name,
				o.SectorKey,
				o.RoleKey,
				o.RarityKey,
			),
			"cost":        o.Cost,
			"available":   true,
			"synergyHint": nil,
		}
	}

	proj := map[string]any{
		"modifier": map[string]any{
			"id":          "mod_default",
			"name":        "Standard Conditions",
			"description": "Default voyage conditions.",
		},
		"objective": map[string]any{
			"id":            "obj_default",
			"name":          "Navigate",
			"description":   "Complete the daily voyage.",
			"progressLabel": nil,
			"rewardLabel":   nil,
		},
		"signals": []map[string]any{},
		"lineup": map[string]any{
			"lockedAt":  nil,
			"maxSlots":  slotCount,
			"slots":     slots,
			"synergies": []map[string]any{},
			"warnings":  warnings,
		},
		"inventory": inventory,
		"shop": map[string]any{
			"offers":      shopOffers,
			"refreshAt":   nil,
			"rerollCost":  2,
			"rerollIndex": 0,
		},
		"strategies":         []map[string]any{},
		"selectedStrategyId": nil,
		"pendingRewardCount": 0,
	}

	return json.Marshal(proj)
}

func makeKeeperMap(id, definitionID, name, sector, role, rarity string) map[string]any {
	return map[string]any{
		"id":             id,
		"definitionId":   definitionID,
		"name":           name,
		"level":          1,
		"rarity":         rarity,
		"role":           role,
		"sector":         sector,
		"passiveSummary": defaultPassiveSummary(definitionID),
		"artworkUrl":     nil,
	}
}

func defaultPassiveSummary(keeperKey string) string {
	switch keeperKey {
	case "crest_sovereign":
		return "Commands the Crest sector with authority."
	case "harbor_warden":
		return "Guards the Harbor sector vigilantly."
	case "current_weaver":
		return "Weaves through Current sector currents."
	default:
		return "Adapts to changing market conditions."
	}
}
