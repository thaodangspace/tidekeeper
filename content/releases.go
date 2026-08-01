package content

import (
	"encoding/json"

	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// Baseline definition keys carried by every approved complete baseline release.
// The legacy projection identities mod_default/obj_default map to these exact
// keys during cutover.
const (
	BaselineModifierKey  = "standard_conditions"
	BaselineObjectiveKey = "navigate"
	BaselineRuleSetKey   = "standard_rules"
)

// Baseline release versions. Versions 1 and 2 already identify historical
// schema-1 Keeper catalog releases (migration 000006 seeds release 1), so
// approved complete baselines begin at version 3.
const (
	BaselineV1Version int64 = 3
	BaselineV2Version int64 = 4
)

// Approved legacy source catalogs embedded by complete baselines.
const (
	SourceCatalogV1 = "v1"
	SourceCatalogV2 = "v2"
)

// CompleteReleaseV1 returns the approved complete baseline at version, composed
// from the approved CatalogV1 Keeper/market catalog plus Standard Conditions,
// Navigate, and the standard game-rule set. Production Strategy, Relic, and
// Synergy collections stay empty until approved definitions are supplied.
func CompleteReleaseV1(version int64) *Release {
	return completeBaseline(version, keeper.CatalogV1())
}

// CompleteReleaseV2 returns the approved complete baseline at version, composed
// from the approved CatalogV2 Keeper/market catalog plus the standard gameplay
// rules. Strategy, Relic, and Synergy collections stay empty.
func CompleteReleaseV2(version int64) *Release {
	return completeBaseline(version, keeper.CatalogV2())
}

// CompleteReleaseForSource returns the approved complete baseline that embeds
// the given recognized source catalog at version.
func CompleteReleaseForSource(sourceCatalog string, version int64) *Release {
	if sourceCatalog == SourceCatalogV2 {
		return CompleteReleaseV2(version)
	}
	return CompleteReleaseV1(version)
}

// sourceCatalogRelease returns the approved source catalog (with its original
// schema-1 version) for a recognized source catalog label.
func sourceCatalogRelease(sourceCatalog string) keeper.CatalogRelease {
	if sourceCatalog == SourceCatalogV2 {
		return keeper.CatalogV2()
	}
	return keeper.CatalogV1()
}

// keeperCompatible reports whether a target release embeds the same approved
// Keeper catalog content as the recognized source catalog. The owning release
// version differs by construction (the target embeds the source catalog at a
// new version), so the version field is normalized before comparing checksums.
// Extra approved Strategy/Relic/Synergy definitions do not affect this check.
func keeperCompatible(source, target keeper.CatalogRelease) (bool, error) {
	target.Version = source.Version
	targetChecksum, err := keeper.Checksum(target)
	if err != nil {
		return false, err
	}
	sourceChecksum, err := keeper.Checksum(source)
	if err != nil {
		return false, err
	}
	return targetChecksum == sourceChecksum, nil
}

func completeBaseline(version int64, catalog keeper.CatalogRelease) *Release {
	// The Keeper section is persisted at the owning release version, matching
	// how Repository.LoadRelease reconstructs releases (Keeper.Version ==
	// release.Version). The approved catalog content is preserved unchanged.
	catalog.Version = version
	return &Release{
		Version: version,
		Keeper:  catalog,
		Modifiers: []Modifier{{
			Key:         BaselineModifierKey,
			Name:        "Standard Conditions",
			Description: "Standard market conditions for this Daily Tide.",
			RuleKey:     RuleStandardConditions,
			RuleConfig:  standardRuleConfig(),
		}},
		Objectives: []Objective{{
			Key:         BaselineObjectiveKey,
			Name:        "Navigate",
			Description: "Hold a balanced position and outperform the sector benchmark.",
			RuleKey:     RuleNavigate,
			RuleConfig:  standardRuleConfig(),
		}},
		GameRuleSets: []GameRuleSet{{
			Key:         BaselineRuleSetKey,
			Name:        "Standard Rules",
			Description: "The standard Tidekeepers game-rule set.",
			RuleKey:     RuleStandardRules,
			RuleConfig:  standardRuleConfig(),
		}},
	}
}

func standardRuleConfig() json.RawMessage {
	return json.RawMessage(`{"schemaVersion":1,"parameters":{}}`)
}

// SourceFingerprint recognizes one approved legacy projection source. Matching
// pins the source content version plus the projection Modifier/Objective
// identity and names the exact keys to resolve inside a complete target release.
type SourceFingerprint struct {
	Label         string
	SourceVersion int64
	ModifierID    string
	ObjectiveID   string
	ModifierKey   string
	ObjectiveKey  string
	SourceCatalog string
}

// recognizedSourceFingerprints are the only approved legacy sources. The
// migration-seeded development case is explicit rather than a generic guess:
// its identity is mod_default/obj_default at source release 1. Approved
// CatalogV1/V2 sources project the baseline keys directly.
var recognizedSourceFingerprints = []SourceFingerprint{
	{
		Label:         "migration-seeded-development",
		SourceVersion: 1,
		ModifierID:    "mod_default",
		ObjectiveID:   "obj_default",
		ModifierKey:   BaselineModifierKey,
		ObjectiveKey:  BaselineObjectiveKey,
		SourceCatalog: SourceCatalogV1,
	},
	{
		Label:         "keepers-v1",
		SourceVersion: 1,
		ModifierID:    BaselineModifierKey,
		ObjectiveID:   BaselineObjectiveKey,
		ModifierKey:   BaselineModifierKey,
		ObjectiveKey:  BaselineObjectiveKey,
		SourceCatalog: SourceCatalogV1,
	},
	{
		Label:         "sectors-v2",
		SourceVersion: 2,
		ModifierID:    BaselineModifierKey,
		ObjectiveID:   BaselineObjectiveKey,
		ModifierKey:   BaselineModifierKey,
		ObjectiveKey:  BaselineObjectiveKey,
		SourceCatalog: SourceCatalogV2,
	},
}

// RecognizeSourceFingerprint returns the approved fingerprint matching a legacy
// source content version and projection identity. The boolean result reports
// whether the source is recognized at all; every unknown fingerprint is rejected.
func RecognizeSourceFingerprint(version int64, identity daily.ProjectionIdentity) (SourceFingerprint, bool) {
	for _, fingerprint := range recognizedSourceFingerprints {
		if fingerprint.SourceVersion == version &&
			fingerprint.ModifierID == identity.ModifierID &&
			fingerprint.ObjectiveID == identity.ObjectiveID {
			return fingerprint, true
		}
	}
	return SourceFingerprint{}, false
}

// SourceCatalogForVersion maps a legacy source content version to the approved
// source catalog that produced it.
func SourceCatalogForVersion(version int64) (string, bool) {
	switch version {
	case 1:
		return SourceCatalogV1, true
	case 2:
		return SourceCatalogV2, true
	}
	return "", false
}
