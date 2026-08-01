package content

import (
	"encoding/json"
	"regexp"

	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// Checksum schemas identify the canonical serialization used to produce a
// release checksum. Schema 1 is the historical Keeper catalog digest; schema 2
// covers a complete gameplay release and MUST NOT change schema-1 meaning.
const (
	ChecksumSchemaV1 int = 1
	ChecksumSchemaV2 int = 2
)

// ContentWeightScale is the fixed decimal scale for normalized gameplay-content
// weights and ratios (1 unit = 1e-6).
const ContentWeightScale int64 = 1_000_000

var (
	contentKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	ruleKeyPattern    = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)
)

// Kind is a stable gameplay-content kind within a release.
type Kind string

// Gameplay content kinds owned by a complete release.
const (
	KindStrategy    Kind = "strategy"
	KindRelic       Kind = "relic"
	KindSynergy     Kind = "synergy"
	KindModifier    Kind = "modifier"
	KindObjective   Kind = "objective"
	KindGameRuleSet Kind = "game_rule_set"
)

// Release is a complete candidate gameplay content release. It composes the
// existing Keeper catalog with the versioned definition kinds. Strategy, Relic,
// and Synergy collections may be empty until approved production content exists.
type Release struct {
	Version      int64
	Keeper       keeper.CatalogRelease
	Strategies   []Strategy
	Relics       []Relic
	Synergies    []Synergy
	Modifiers    []Modifier
	Objectives   []Objective
	GameRuleSets []GameRuleSet
}

// Strategy is one immutable versioned Strategy definition.
type Strategy struct {
	Key         string
	Name        string
	Description string
	Upside      string
	Downside    string
	RuleKey     string
	RuleConfig  json.RawMessage
}

// Relic is one immutable versioned Relic definition.
type Relic struct {
	Key         string
	Name        string
	Description string
	RuleKey     string
	RuleConfig  json.RawMessage
}

// Synergy is one immutable versioned Synergy definition.
type Synergy struct {
	Key           string
	Name          string
	Description   string
	RequiredCount int
	RuleKey       string
	RuleConfig    json.RawMessage
}

// Modifier is one immutable versioned Daily Modifier definition.
type Modifier struct {
	Key         string
	Name        string
	Description string
	RuleKey     string
	RuleConfig  json.RawMessage
}

// Objective is one immutable versioned Daily Objective definition.
type Objective struct {
	Key           string
	Name          string
	Description   string
	ProgressLabel *string
	RewardLabel   *string
	RuleKey       string
	RuleConfig    json.RawMessage
}

// GameRuleSet is one immutable versioned game-rule-set definition.
type GameRuleSet struct {
	Key         string
	Name        string
	Description string
	RuleKey     string
	RuleConfig  json.RawMessage
}

// keyedDefinition is satisfied by every versioned definition kind and exposes
// its stable business key.
type keyedDefinition interface {
	stableKey() string
}

func (s Strategy) stableKey() string    { return s.Key }
func (r Relic) stableKey() string       { return r.Key }
func (s Synergy) stableKey() string     { return s.Key }
func (m Modifier) stableKey() string    { return m.Key }
func (o Objective) stableKey() string   { return o.Key }
func (g GameRuleSet) stableKey() string { return g.Key }
