package keeper

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateRejectsInvalidCatalogContent(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CatalogRelease)
		want   string
	}{
		{
			name: "duplicate asset key",
			mutate: func(release *CatalogRelease) {
				release.Assets = append(release.Assets, release.Assets[0])
			},
			want: "duplicate asset key",
		},
		{
			name: "unknown basket asset",
			mutate: func(release *CatalogRelease) {
				release.Baskets[0].Components[0].AssetKey = "unknown_asset"
			},
			want: "unknown asset",
		},
		{
			name: "invalid basket total",
			mutate: func(release *CatalogRelease) {
				release.Baskets[0].Components[0].Weight--
			},
			want: "component weights total",
		},
		{
			name: "duplicate basket component",
			mutate: func(release *CatalogRelease) {
				release.Baskets[0].Components[1].AssetKey = release.Baskets[0].Components[0].AssetKey
			},
			want: "repeats asset",
		},
		{
			name: "unknown passive rule",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].PassiveRuleKey = "NOT_A_RULE"
			},
			want: "unknown passive rule key",
		},
		{
			name: "non object passive configuration",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].PassiveRuleConfig = json.RawMessage(`[]`)
			},
			want: "must be a JSON object",
		},
		{
			name: "unknown passive configuration parameter",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].PassiveRuleConfig = json.RawMessage(`{"schemaVersion":1,"parameters":{"minimumOutperformingComponents":3,"globalRelativeScoreBonusBps":2000,"broadDeclineDepthPenaltyReductionBps":1500,"executable":"DROP TABLE"}}`)
			},
			want: "unknown parameter",
		},
		{
			name: "empty Current name",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].CurrentName = ""
			},
			want: "invalid display Current data",
		},
		{
			name: "non positive expected turbulence",
			mutate: func(release *CatalogRelease) {
				value := 0
				release.Definitions[0].ExpectedTurbulenceBPS = &value
			},
			want: "invalid expected turbulence",
		},
		{
			name: "upgrade cycle",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].UpgradeTree = json.RawMessage(`{
					"rootNodeKey":"base",
					"nodes":[
						{"key":"base","name":"Base","next":["first"],"effects":{}},
						{"key":"first","name":"First","next":["base"],"effects":{}}
					]
				}`)
			},
			want: "upgrade root has an incoming edge",
		},
		{
			name: "disconnected upgrade node",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].UpgradeTree = json.RawMessage(`{
					"rootNodeKey":"base",
					"nodes":[
						{"key":"base","name":"Base","next":["first"],"effects":{}},
						{"key":"first","name":"First","next":[],"effects":{}},
						{"key":"second","name":"Second","next":[],"effects":{}}
					]
				}`)
			},
			want: "must have exactly one parent",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			release := CatalogV1()
			test.mutate(&release)
			err := Validate(release)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, test.want)
			}
		})
	}
}
