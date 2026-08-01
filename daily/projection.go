package daily

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const supportedSchemaVersion = 1

type v1Modifier struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type v1Objective struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	ProgressLabel *string `json:"progressLabel"`
	RewardLabel   *string `json:"rewardLabel"`
}

type v1Signal struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Direction    string    `json:"direction"`
	Strength     string    `json:"strength"`
	ObservedFrom time.Time `json:"observedFrom"`
	ObservedTo   time.Time `json:"observedTo"`
}

type v1Lineup struct {
	LockedAt  *time.Time       `json:"lockedAt"`
	MaxSlots  int32            `json:"maxSlots"`
	Slots     []v1FleetSlot    `json:"slots"`
	Synergies []v1Synergy      `json:"synergies"`
	Warnings  []v1FleetWarning `json:"warnings"`
}

type v1FleetSlot struct {
	Index  int32     `json:"index"`
	Keeper *v1Keeper `json:"keeper"`
}

type v1FleetWarning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type v1Synergy struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	State         string `json:"state"`
	CurrentCount  int32  `json:"currentCount"`
	RequiredCount int32  `json:"requiredCount"`
}

type v1Keeper struct {
	ID             string  `json:"id"`
	DefinitionID   string  `json:"definitionId"`
	Name           string  `json:"name"`
	Level          int32   `json:"level"`
	Rarity         string  `json:"rarity"`
	Role           string  `json:"role"`
	Sector         string  `json:"sector"`
	PassiveSummary string  `json:"passiveSummary"`
	ArtworkURL     *string `json:"artworkUrl"`
}

type v1Shop struct {
	Offers      []v1ShopOffer `json:"offers"`
	RefreshAt   *time.Time    `json:"refreshAt"`
	RerollCost  int32         `json:"rerollCost"`
	RerollIndex int32         `json:"rerollIndex"`
}

type v1ShopOffer struct {
	ID          string   `json:"id"`
	Keeper      v1Keeper `json:"keeper"`
	Cost        int32    `json:"cost"`
	Available   bool     `json:"available"`
	SynergyHint *string  `json:"synergyHint"`
}

type v1Strategy struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Upside      string `json:"upside"`
	Downside    string `json:"downside"`
	Available   bool   `json:"available"`
}

type v1Projection struct {
	Modifier           v1Modifier   `json:"modifier"`
	Objective          v1Objective  `json:"objective"`
	Signals            []v1Signal   `json:"signals"`
	Lineup             v1Lineup     `json:"lineup"`
	Inventory          []v1Keeper   `json:"inventory"`
	Shop               v1Shop       `json:"shop"`
	Strategies         []v1Strategy `json:"strategies"`
	SelectedStrategyID *string      `json:"selectedStrategyId"`
	PendingRewardCount int32        `json:"pendingRewardCount"`
}

// ProjectionIdentity captures the authoritative identity fields of a legacy
// schema-1 projection: the Modifier and Objective, the available Strategy set,
// and the per-player selected Strategy. Cutover uses it to recognize a
// projection's source and resolve exact definition keys without depending on
// presentation text.
type ProjectionIdentity struct {
	ModifierID         string
	ModifierName       string
	ObjectiveID        string
	ObjectiveName      string
	StrategyIDs        []string
	SelectedStrategyID *string
}

// DecodeProjectionIdentity decodes and validates a legacy schema-1 projection
// and returns its authoritative identity. Unsupported schemas, empty data, or
// invalid projections fail closed so cutover never guesses from malformed text.
func DecodeProjectionIdentity(schemaVersion int16, data []byte) (ProjectionIdentity, error) {
	proj, err := decodeAndValidateSchemaV1Projection(schemaVersion, data)
	if err != nil {
		return ProjectionIdentity{}, err
	}
	strategyIDs := make([]string, 0, len(proj.Strategies))
	for _, strategy := range proj.Strategies {
		strategyIDs = append(strategyIDs, strategy.ID)
	}
	sort.Strings(strategyIDs)
	var selectedStrategyID *string
	if proj.SelectedStrategyID != nil && *proj.SelectedStrategyID != "" {
		value := *proj.SelectedStrategyID
		selectedStrategyID = &value
	}
	return ProjectionIdentity{
		ModifierID:         proj.Modifier.ID,
		ModifierName:       proj.Modifier.Name,
		ObjectiveID:        proj.Objective.ID,
		ObjectiveName:      proj.Objective.Name,
		StrategyIDs:        strategyIDs,
		SelectedStrategyID: selectedStrategyID,
	}, nil
}

// Compatible reports whether two projections share the same authoritative
// Modifier, Objective, and available Strategy set. The per-player selection is
// deliberately ignored: different players on the same Tide may select different
// available Strategies without contradicting one another.
func (p ProjectionIdentity) Compatible(other ProjectionIdentity) bool {
	if p.ModifierID != other.ModifierID || p.ObjectiveID != other.ObjectiveID {
		return false
	}
	if len(p.StrategyIDs) != len(other.StrategyIDs) {
		return false
	}
	for index := range p.StrategyIDs {
		if p.StrategyIDs[index] != other.StrategyIDs[index] {
			return false
		}
	}
	return true
}

func decodeAndValidateSchemaV1Projection(schemaVersion int16, data []byte) (v1Projection, error) {
	if schemaVersion != supportedSchemaVersion {
		return v1Projection{}, fmt.Errorf("unsupported projection schema version %d", schemaVersion)
	}
	if len(data) == 0 {
		return v1Projection{}, fmt.Errorf("empty projection data")
	}

	var proj v1Projection
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proj); err != nil {
		return v1Projection{}, fmt.Errorf("decode projection: %w", err)
	}

	if err := validateV1Projection(proj); err != nil {
		return v1Projection{}, fmt.Errorf("validate projection: %w", err)
	}

	return proj, nil
}

func validateV1Projection(p v1Projection) error {
	if p.Modifier.ID == "" {
		return fmt.Errorf("modifier.id is required")
	}
	if p.Modifier.Name == "" {
		return fmt.Errorf("modifier.name is required")
	}
	if p.Modifier.Description == "" {
		return fmt.Errorf("modifier.description is required")
	}

	if p.Objective.ID == "" {
		return fmt.Errorf("objective.id is required")
	}
	if p.Objective.Name == "" {
		return fmt.Errorf("objective.name is required")
	}
	if p.Objective.Description == "" {
		return fmt.Errorf("objective.description is required")
	}

	if p.Signals == nil {
		return fmt.Errorf("signals must not be null")
	}
	for i, s := range p.Signals {
		if s.ID == "" {
			return fmt.Errorf("signals[%d].id is required", i)
		}
		if s.Name == "" {
			return fmt.Errorf("signals[%d].name is required", i)
		}
		if s.Description == "" {
			return fmt.Errorf("signals[%d].description is required", i)
		}
		if s.Direction == "" {
			return fmt.Errorf("signals[%d].direction is required", i)
		}
		if s.Strength == "" {
			return fmt.Errorf("signals[%d].strength is required", i)
		}
		if s.ObservedFrom.IsZero() {
			return fmt.Errorf("signals[%d].observedFrom is required", i)
		}
		if s.ObservedTo.IsZero() {
			return fmt.Errorf("signals[%d].observedTo is required", i)
		}
		if s.ObservedTo.Before(s.ObservedFrom) {
			return fmt.Errorf("signals[%d].observedTo must not be before observedFrom", i)
		}
	}

	if p.Lineup.Slots == nil {
		return fmt.Errorf("lineup.slots must not be null")
	}
	if p.Lineup.MaxSlots <= 0 {
		return fmt.Errorf("lineup.maxSlots must be positive")
	}
	if int32(len(p.Lineup.Slots)) != p.Lineup.MaxSlots {
		return fmt.Errorf("lineup.slots length %d does not match maxSlots %d", len(p.Lineup.Slots), p.Lineup.MaxSlots)
	}
	for i, slot := range p.Lineup.Slots {
		if slot.Index != int32(i) {
			return fmt.Errorf("lineup.slots[%d].index is %d, want %d", i, slot.Index, i)
		}
		if slot.Keeper != nil {
			if err := validateV1Keeper(*slot.Keeper, fmt.Sprintf("lineup.slots[%d].keeper", i)); err != nil {
				return err
			}
		}
	}
	if p.Lineup.Synergies == nil {
		return fmt.Errorf("lineup.synergies must not be null")
	}
	for i, s := range p.Lineup.Synergies {
		if s.ID == "" {
			return fmt.Errorf("lineup.synergies[%d].id is required", i)
		}
		if s.Name == "" {
			return fmt.Errorf("lineup.synergies[%d].name is required", i)
		}
		if s.Description == "" {
			return fmt.Errorf("lineup.synergies[%d].description is required", i)
		}
		if s.State == "" {
			return fmt.Errorf("lineup.synergies[%d].state is required", i)
		}
		if s.CurrentCount < 0 {
			return fmt.Errorf("lineup.synergies[%d].currentCount must be non-negative", i)
		}
		if s.RequiredCount <= 0 {
			return fmt.Errorf("lineup.synergies[%d].requiredCount must be positive", i)
		}
	}
	if p.Lineup.Warnings == nil {
		return fmt.Errorf("lineup.warnings must not be null")
	}
	for i, w := range p.Lineup.Warnings {
		if w.Code == "" {
			return fmt.Errorf("lineup.warnings[%d].code is required", i)
		}
		if w.Message == "" {
			return fmt.Errorf("lineup.warnings[%d].message is required", i)
		}
		if w.Severity == "" {
			return fmt.Errorf("lineup.warnings[%d].severity is required", i)
		}
	}

	if p.Inventory == nil {
		return fmt.Errorf("inventory must not be null")
	}
	for i, k := range p.Inventory {
		if err := validateV1Keeper(k, fmt.Sprintf("inventory[%d]", i)); err != nil {
			return err
		}
	}

	if p.Shop.Offers == nil {
		return fmt.Errorf("shop.offers must not be null")
	}
	for i, o := range p.Shop.Offers {
		if o.ID == "" {
			return fmt.Errorf("shop.offers[%d].id is required", i)
		}
		if o.Cost < 0 {
			return fmt.Errorf("shop.offers[%d].cost must be non-negative", i)
		}
		if err := validateV1Keeper(o.Keeper, fmt.Sprintf("shop.offers[%d].keeper", i)); err != nil {
			return err
		}
	}
	if p.Shop.RerollCost < 0 {
		return fmt.Errorf("shop.rerollCost must be non-negative")
	}
	if p.Shop.RerollIndex < 0 {
		return fmt.Errorf("shop.rerollIndex must be non-negative")
	}

	if p.Strategies == nil {
		return fmt.Errorf("strategies must not be null")
	}
	for i, s := range p.Strategies {
		if s.ID == "" {
			return fmt.Errorf("strategies[%d].id is required", i)
		}
		if s.Name == "" {
			return fmt.Errorf("strategies[%d].name is required", i)
		}
		if s.Description == "" {
			return fmt.Errorf("strategies[%d].description is required", i)
		}
		if s.Upside == "" {
			return fmt.Errorf("strategies[%d].upside is required", i)
		}
		if s.Downside == "" {
			return fmt.Errorf("strategies[%d].downside is required", i)
		}
	}

	if p.SelectedStrategyID != nil && *p.SelectedStrategyID != "" {
		found := false
		for _, s := range p.Strategies {
			if s.ID == *p.SelectedStrategyID && s.Available {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("selectedStrategyId %q does not match an available strategy", *p.SelectedStrategyID)
		}
	}

	if p.PendingRewardCount < 0 {
		return fmt.Errorf("pendingRewardCount must be non-negative")
	}

	return nil
}

func validateV1Keeper(k v1Keeper, prefix string) error {
	if k.ID == "" {
		return fmt.Errorf("%s.id is required", prefix)
	}
	if k.DefinitionID == "" {
		return fmt.Errorf("%s.definitionId is required", prefix)
	}
	if k.Name == "" {
		return fmt.Errorf("%s.name is required", prefix)
	}
	if k.Level < 1 {
		return fmt.Errorf("%s.level must be at least 1", prefix)
	}
	if k.Rarity == "" {
		return fmt.Errorf("%s.rarity is required", prefix)
	}
	if k.Role == "" {
		return fmt.Errorf("%s.role is required", prefix)
	}
	if k.Sector == "" {
		return fmt.Errorf("%s.sector is required", prefix)
	}
	if k.PassiveSummary == "" {
		return fmt.Errorf("%s.passiveSummary is required", prefix)
	}
	return nil
}

func mapV1ToDaily(p v1Projection) (Modifier, Objective, []Signal, Lineup, []Keeper, Shop, []Strategy) {
	modifier := Modifier{
		ID:          p.Modifier.ID,
		Name:        p.Modifier.Name,
		Description: p.Modifier.Description,
	}

	objective := Objective{
		ID:            p.Objective.ID,
		Name:          p.Objective.Name,
		Description:   p.Objective.Description,
		ProgressLabel: p.Objective.ProgressLabel,
		RewardLabel:   p.Objective.RewardLabel,
	}

	signals := make([]Signal, len(p.Signals))
	for i, s := range p.Signals {
		signals[i] = Signal{
			ID:           s.ID,
			Name:         s.Name,
			Description:  s.Description,
			Direction:    s.Direction,
			Strength:     s.Strength,
			ObservedFrom: s.ObservedFrom.UTC(),
			ObservedTo:   s.ObservedTo.UTC(),
		}
	}

	var lockedAt *time.Time
	if p.Lineup.LockedAt != nil {
		t := p.Lineup.LockedAt.UTC()
		lockedAt = &t
	}
	lineup := Lineup{
		LockedAt:  lockedAt,
		MaxSlots:  p.Lineup.MaxSlots,
		Slots:     mapV1FleetSlots(p.Lineup.Slots),
		Synergies: mapV1Synergies(p.Lineup.Synergies),
		Warnings:  mapV1FleetWarnings(p.Lineup.Warnings),
	}

	inventory := make([]Keeper, len(p.Inventory))
	for i, k := range p.Inventory {
		inventory[i] = mapV1Keeper(k)
	}

	var refreshAt *time.Time
	if p.Shop.RefreshAt != nil {
		t := p.Shop.RefreshAt.UTC()
		refreshAt = &t
	}
	shop := Shop{
		Offers:      mapV1ShopOffers(p.Shop.Offers),
		RefreshAt:   refreshAt,
		RerollCost:  p.Shop.RerollCost,
		RerollIndex: p.Shop.RerollIndex,
	}

	strategies := make([]Strategy, len(p.Strategies))
	for i, s := range p.Strategies {
		strategies[i] = Strategy(s)
	}

	return modifier, objective, signals, lineup, inventory, shop, strategies
}

func mapV1FleetSlots(slots []v1FleetSlot) []FleetSlot {
	result := make([]FleetSlot, len(slots))
	for i, s := range slots {
		var keeper *Keeper
		if s.Keeper != nil {
			k := mapV1Keeper(*s.Keeper)
			keeper = &k
		}
		result[i] = FleetSlot{
			Index:  s.Index,
			Keeper: keeper,
		}
	}
	return result
}

func mapV1Synergies(synergies []v1Synergy) []Synergy {
	result := make([]Synergy, len(synergies))
	for i, s := range synergies {
		result[i] = Synergy(s)
	}
	return result
}

func mapV1FleetWarnings(warnings []v1FleetWarning) []FleetWarning {
	result := make([]FleetWarning, len(warnings))
	for i, w := range warnings {
		result[i] = FleetWarning(w)
	}
	return result
}

func mapV1Keeper(k v1Keeper) Keeper {
	return Keeper(k)
}

func mapV1ShopOffers(offers []v1ShopOffer) []ShopOffer {
	result := make([]ShopOffer, len(offers))
	for i, o := range offers {
		result[i] = ShopOffer{
			ID:          o.ID,
			Keeper:      mapV1Keeper(o.Keeper),
			Cost:        o.Cost,
			Available:   o.Available,
			SynergyHint: o.SynergyHint,
		}
	}
	return result
}
