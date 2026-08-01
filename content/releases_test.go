package content

import (
	"bytes"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func TestCompleteBaselineV1ValidatesAndChecksums(t *testing.T) {
	release := CompleteReleaseV1(BaselineV1Version)
	if err := Validate(*release); err != nil {
		t.Fatalf("Validate(CompleteReleaseV1) error = %v", err)
	}
	checksum, err := Checksum(*release)
	if err != nil {
		t.Fatalf("Checksum(CompleteReleaseV1) error = %v", err)
	}
	repeat, err := Checksum(*release)
	if err != nil {
		t.Fatalf("Checksum(CompleteReleaseV1) repeat error = %v", err)
	}
	if !bytes.Equal(checksum[:], repeat[:]) {
		t.Fatalf("checksum is not deterministic")
	}
	if got := len(release.Modifiers); got != 1 {
		t.Fatalf("modifiers = %d, want 1", got)
	}
	if got := len(release.Objectives); got != 1 {
		t.Fatalf("objectives = %d, want 1", got)
	}
	if got := len(release.GameRuleSets); got != 1 {
		t.Fatalf("game rule sets = %d, want 1", got)
	}
	if got := len(release.Strategies) + len(release.Relics) + len(release.Synergies); got != 0 {
		t.Fatalf("strategy/relic/synergy collections must be empty, got %d", got)
	}
	if release.Modifiers[0].Key != BaselineModifierKey {
		t.Fatalf("modifier key = %q, want %q", release.Modifiers[0].Key, BaselineModifierKey)
	}
	if release.Objectives[0].Key != BaselineObjectiveKey {
		t.Fatalf("objective key = %q, want %q", release.Objectives[0].Key, BaselineObjectiveKey)
	}
	if release.Keeper.Version != BaselineV1Version {
		t.Fatalf("keeper version = %d, want %d", release.Keeper.Version, BaselineV1Version)
	}
}

func TestCompleteBaselineV2ValidatesAndChecksums(t *testing.T) {
	release := CompleteReleaseV2(BaselineV2Version)
	if err := Validate(*release); err != nil {
		t.Fatalf("Validate(CompleteReleaseV2) error = %v", err)
	}
	if _, err := Checksum(*release); err != nil {
		t.Fatalf("Checksum(CompleteReleaseV2) error = %v", err)
	}
	if release.Keeper.Version != BaselineV2Version {
		t.Fatalf("keeper version = %d, want %d", release.Keeper.Version, BaselineV2Version)
	}
}

func TestCompleteBaselineChecksumsDifferByCatalogAndVersion(t *testing.T) {
	v1checksum, err := Checksum(*CompleteReleaseV1(BaselineV1Version))
	if err != nil {
		t.Fatalf("checksum v1: %v", err)
	}
	v2checksum, err := Checksum(*CompleteReleaseV2(BaselineV2Version))
	if err != nil {
		t.Fatalf("checksum v2: %v", err)
	}
	if bytes.Equal(v1checksum[:], v2checksum[:]) {
		t.Fatalf("baseline v1 and v2 must checksum differently")
	}
	v1AtOtherVersion, err := Checksum(*CompleteReleaseV1(9))
	if err != nil {
		t.Fatalf("checksum v1@9: %v", err)
	}
	if bytes.Equal(v1checksum[:], v1AtOtherVersion[:]) {
		t.Fatalf("baseline versions must checksum differently")
	}
}

func TestCompleteReleaseForSource(t *testing.T) {
	if CompleteReleaseForSource(SourceCatalogV1, 3).Keeper.Version != 3 {
		t.Fatalf("v1 source must embed the approved v1 catalog at the release version")
	}
	if CompleteReleaseForSource(SourceCatalogV2, 4).Keeper.Version != 4 {
		t.Fatalf("v2 source must embed the approved v2 catalog at the release version")
	}
}

func TestRecognizeSourceFingerprint(t *testing.T) {
	cases := []struct {
		name       string
		version    int64
		modifierID string
		objective  string
		label      string
		ok         bool
	}{
		{name: "migration seeded dev", version: 1, modifierID: "mod_default", objective: "obj_default", label: "migration-seeded-development", ok: true},
		{name: "keepers v1", version: 1, modifierID: BaselineModifierKey, objective: BaselineObjectiveKey, label: "keepers-v1", ok: true},
		{name: "sectors v2", version: 2, modifierID: BaselineModifierKey, objective: BaselineObjectiveKey, label: "sectors-v2", ok: true},
		{name: "unknown identity", version: 1, modifierID: "mystery_mod", objective: "obj_default", ok: false},
		{name: "unknown version", version: 7, modifierID: BaselineModifierKey, objective: BaselineObjectiveKey, ok: false},
		{name: "mixed identity", version: 1, modifierID: "mod_default", objective: "navigate", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fingerprint, ok := RecognizeSourceFingerprint(tc.version, daily.ProjectionIdentity{
				ModifierID:  tc.modifierID,
				ObjectiveID: tc.objective,
			})
			if ok != tc.ok {
				t.Fatalf("ok = %t, want %t", ok, tc.ok)
			}
			if ok && fingerprint.Label != tc.label {
				t.Fatalf("label = %q, want %q", fingerprint.Label, tc.label)
			}
		})
	}
}

func TestSourceCatalogForVersion(t *testing.T) {
	if catalog, ok := SourceCatalogForVersion(1); !ok || catalog != SourceCatalogV1 {
		t.Fatalf("version 1 catalog = %q, %t", catalog, ok)
	}
	if catalog, ok := SourceCatalogForVersion(2); !ok || catalog != SourceCatalogV2 {
		t.Fatalf("version 2 catalog = %q, %t", catalog, ok)
	}
	if _, ok := SourceCatalogForVersion(3); ok {
		t.Fatalf("version 3 must not be a recognized legacy source")
	}
}

func TestFingerprintTargetKeysMapToBaseline(t *testing.T) {
	for _, fingerprint := range recognizedSourceFingerprints {
		if fingerprint.ModifierKey != BaselineModifierKey || fingerprint.ObjectiveKey != BaselineObjectiveKey {
			t.Fatalf("fingerprint %q does not map to baseline keys", fingerprint.Label)
		}
	}
}

func TestCompleteBaselineEmbedsApprovedSourceCatalog(t *testing.T) {
	for _, tc := range []struct {
		source   string
		baseline func(int64) *Release
		catalog  keeper.CatalogRelease
	}{
		{source: SourceCatalogV1, baseline: CompleteReleaseV1, catalog: keeper.CatalogV1()},
		{source: SourceCatalogV2, baseline: CompleteReleaseV2, catalog: keeper.CatalogV2()},
	} {
		release := tc.baseline(BaselineV1Version)
		tc.catalog.Version = BaselineV1Version
		embedded, err := keeper.Checksum(release.Keeper)
		if err != nil {
			t.Fatalf("keeper checksum embedded %s: %v", tc.source, err)
		}
		expected, err := keeper.Checksum(tc.catalog)
		if err != nil {
			t.Fatalf("keeper checksum source %s: %v", tc.source, err)
		}
		if !bytes.Equal(embedded[:], expected[:]) {
			t.Fatalf("baseline %s must embed the approved source catalog", tc.source)
		}
	}
}

func TestKeeperCompatibleNormalizesVersion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source keeper.CatalogRelease
		target keeper.CatalogRelease
		want   bool
	}{
		{
			name:   "v1 source embeds v1 catalog at new version",
			source: keeper.CatalogV1(),
			target: keeper.CatalogV1(),
			want:   true,
		},
		{
			name:   "v2 source embeds v2 catalog at new version",
			source: keeper.CatalogV2(),
			target: keeper.CatalogV2(),
			want:   true,
		},
		{
			name:   "v1 source rejects v2 catalog",
			source: keeper.CatalogV1(),
			target: keeper.CatalogV2(),
			want:   false,
		},
		{
			name:   "v2 source rejects v1 catalog",
			source: keeper.CatalogV2(),
			target: keeper.CatalogV1(),
			want:   false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.target.Version = tc.source.Version + 10
			got, err := keeperCompatible(tc.source, tc.target)
			if err != nil {
				t.Fatalf("keeperCompatible() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("keeperCompatible() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestKeeperCompatibleRejectsChangedKeeperContent(t *testing.T) {
	source := keeper.CatalogV1()
	target := keeper.CatalogV1()
	target.Version = 13
	target.Definitions[0].Name = "Renamed Keeper"
	if got, err := keeperCompatible(source, target); err != nil {
		t.Fatalf("keeperCompatible() error = %v", err)
	} else if got {
		t.Fatalf("keeperCompatible() must reject changed keeper content")
	}
}
