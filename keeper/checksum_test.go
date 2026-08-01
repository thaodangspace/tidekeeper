package keeper

import (
	"bytes"
	"crypto/sha256"
	"slices"
	"testing"
)

func TestChecksumIsIndependentOfCollectionAndJSONObjectOrdering(t *testing.T) {
	baseline := CatalogV1()
	want, err := Checksum(baseline)
	if err != nil {
		t.Fatalf("Checksum(baseline): %v", err)
	}

	reordered := CatalogV1()
	slices.Reverse(reordered.Assets)
	slices.Reverse(reordered.Baskets)
	slices.Reverse(reordered.Definitions)
	for index := range reordered.Baskets {
		slices.Reverse(reordered.Baskets[index].Components)
	}
	// The equivalent JSON object deliberately uses a different key order.
	for index := range reordered.Definitions {
		if reordered.Definitions[index].Key == "crest_sovereign" {
			reordered.Definitions[index].PassiveRuleConfig = []byte(`{"parameters":{"broadDeclineDepthPenaltyReductionBps":1500,"globalRelativeScoreBonusBps":2000,"minimumOutperformingComponents":3},"schemaVersion":1}`)
		}
	}

	got, err := Checksum(reordered)
	if err != nil {
		t.Fatalf("Checksum(reordered): %v", err)
	}
	if !bytes.Equal(got[:], want[:]) {
		t.Fatalf("checksum = %x, want %x", got, want)
	}
}

func TestChecksumRejectsInvalidRelease(t *testing.T) {
	release := CatalogV1()
	release.Baskets[0].Components[0].Weight--
	if _, err := Checksum(release); err == nil {
		t.Fatal("Checksum(invalid release) succeeded")
	}
}

func TestCanonicalJSONPreservesSchemaOneChecksumMeaning(t *testing.T) {
	for _, release := range []CatalogRelease{CatalogV1(), CatalogV2()} {
		want, err := Checksum(release)
		if err != nil {
			t.Fatalf("Checksum(): %v", err)
		}
		encoded, err := CanonicalJSON(release)
		if err != nil {
			t.Fatalf("CanonicalJSON(): %v", err)
		}
		digest := sha256.Sum256(encoded)
		if !bytes.Equal(digest[:], want[:]) {
			t.Fatalf("CanonicalJSON digest = %x, want Checksum %x", digest, want)
		}
	}
}
