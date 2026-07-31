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

func TestParseUpgradeTreeReturnsDeterministicProjections(t *testing.T) {
	for _, definition := range CatalogV1().Definitions {
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			t.Fatalf("parseUpgradeTree(%q): %v", definition.Key, err)
		}
		if len(nodes) != 3 {
			t.Fatalf("%q node count = %d, want 3", definition.Key, len(nodes))
		}
		rootCount := 0
		depths := make(map[string]int16, len(nodes))
		for i, node := range nodes {
			if node.IsRoot {
				rootCount++
				if node.Depth != 0 {
					t.Errorf("%q root node %q has depth %d, want 0", definition.Key, node.Key, node.Depth)
				}
			}
			depths[node.Key] = node.Depth
			if i > 0 && nodes[i-1].Key >= node.Key {
				t.Errorf("%q nodes are not sorted: %q then %q", definition.Key, nodes[i-1].Key, node.Key)
			}
		}
		if rootCount != 1 {
			t.Errorf("%q root count = %d, want 1", definition.Key, rootCount)
		}
		if len(depths) != 3 {
			t.Errorf("%q unique node keys = %d, want 3", definition.Key, len(depths))
		}
	}
}

func TestParseUpgradeTreeRejectsInvalidGraphs(t *testing.T) {
	tests := []struct {
		name string
		tree string
		want string
	}{
		{
			name: "cycle",
			tree: `{
				"rootNodeKey":"base",
				"nodes":[
					{"key":"base","name":"Base","next":["first"],"effects":{}},
					{"key":"first","name":"First","next":["base"],"effects":{}}
				]
			}`,
			want: "upgrade root has an incoming edge",
		},
		{
			name: "disconnected",
			tree: `{
				"rootNodeKey":"base",
				"nodes":[
					{"key":"base","name":"Base","next":["first"],"effects":{}},
					{"key":"first","name":"First","next":[],"effects":{}},
					{"key":"second","name":"Second","next":[],"effects":{}}
				]
			}`,
			want: "must have exactly one parent",
		},
		{
			name: "unknown child",
			tree: `{
				"rootNodeKey":"base",
				"nodes":[
					{"key":"base","name":"Base","next":["missing"],"effects":{}}
				]
			}`,
			want: "references unknown node",
		},
		{
			name: "root not a node",
			tree: `{
				"rootNodeKey":"absent",
				"nodes":[
					{"key":"base","name":"Base","next":[],"effects":{}}
				]
			}`,
			want: "is not a node",
		},
		{
			name: "multiple parents",
			tree: `{
				"rootNodeKey":"base",
				"nodes":[
					{"key":"base","name":"Base","next":["first"],"effects":{}},
					{"key":"first","name":"First","next":[],"effects":{}}
				]
			}`,
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodes, err := parseUpgradeTree(json.RawMessage(test.tree))
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("parseUpgradeTree() error = %v, want substring %q", err, test.want)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseUpgradeTree() error = %v, want nil", err)
			}
			if len(nodes) != 2 {
				t.Fatalf("node count = %d, want 2", len(nodes))
			}
			if !nodes[0].IsRoot || nodes[0].Key != "base" || nodes[0].Depth != 0 {
				t.Errorf("first node = %+v, want root base at depth 0", nodes[0])
			}
			if nodes[1].IsRoot || nodes[1].Key != "first" || nodes[1].Depth != 1 {
				t.Errorf("second node = %+v, want first at depth 1", nodes[1])
			}
		})
	}
}
