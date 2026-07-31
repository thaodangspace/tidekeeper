package keeper

import (
	"encoding/json"
	"testing"
)

func TestCatalogV1IsCompleteAndValid(t *testing.T) {
	release := CatalogV1()
	if err := Validate(release); err != nil {
		t.Fatalf("Validate(CatalogV1()): %v", err)
	}
	if len(release.Definitions) != 10 {
		t.Fatalf("definition count = %d, want 10", len(release.Definitions))
	}
	if len(release.Baskets) != 10 {
		t.Fatalf("basket count = %d, want 10", len(release.Baskets))
	}
	if len(release.Assets) != 47 {
		t.Fatalf("asset count = %d, want 47", len(release.Assets))
	}

	wantBaskets := map[string]map[string]int64{
		"crest_large_cap":       {"bitcoin": 35_000_000, "ethereum": 25_000_000, "binance_coin": 15_000_000, "solana": 15_000_000, "xrp": 10_000_000},
		"harbor_stability":      {"tether": 30_000_000, "usdc": 30_000_000, "usds": 15_000_000, "dai": 15_000_000, "ethena_usde": 10_000_000},
		"forge_commerce":        {"binance_coin": 30_000_000, "whitebit": 20_000_000, "leo_token": 20_000_000, "crypto_com": 15_000_000, "okb": 15_000_000},
		"exchange_flow":         {"hyperliquid": 30_000_000, "uniswap": 25_000_000, "jupiter": 15_000_000, "cake": 15_000_000, "crv": 15_000_000},
		"oracle_signal":         {"chainlink": 45_000_000, "pyth": 20_000_000, "redstone": 15_000_000, "tellor": 10_000_000, "api3": 10_000_000},
		"veiled_reversal":       {"zcash": 30_000_000, "monero": 30_000_000, "decred": 15_000_000, "arrr": 15_000_000, "verge": 10_000_000},
		"lending_growth":        {"aave": 30_000_000, "morpho": 25_000_000, "syrup": 15_000_000, "compound": 15_000_000, "kamino": 15_000_000},
		"wild_momentum":         {"dogecoin": 30_000_000, "shiba_inu": 20_000_000, "pepe": 20_000_000, "pump": 15_000_000, "bonk": 15_000_000},
		"engine_infrastructure": {"bittensor": 30_000_000, "render": 25_000_000, "filecoin": 20_000_000, "grass": 15_000_000, "graph": 10_000_000},
		"second_current":        {"mantle": 40_000_000, "polygon": 35_000_000, "arbitrum": 25_000_000},
	}
	for _, basket := range release.Baskets {
		want, exists := wantBaskets[basket.Key]
		if !exists {
			t.Fatalf("unexpected basket %q", basket.Key)
		}
		got := make(map[string]int64, len(basket.Components))
		for _, component := range basket.Components {
			got[component.AssetKey] = component.Weight
		}
		if !sameWeights(got, want) {
			t.Errorf("basket %q = %#v, want %#v", basket.Key, got, want)
		}
	}

	for _, definition := range release.Definitions {
		if definition.ExpectedTurbulenceBPS != nil {
			t.Errorf("Keeper %q expected turbulence = %d, want nil until calibration", definition.Key, *definition.ExpectedTurbulenceBPS)
		}
		var tree struct {
			RootNodeKey string `json:"rootNodeKey"`
			Nodes       []struct {
				Key string `json:"key"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(definition.UpgradeTree, &tree); err != nil {
			t.Fatalf("unmarshal upgrade tree for %q: %v", definition.Key, err)
		}
		if tree.RootNodeKey != "base" || len(tree.Nodes) != 3 {
			t.Errorf("Keeper %q upgrade tree = root %q with %d nodes, want base with 3 nodes", definition.Key, tree.RootNodeKey, len(tree.Nodes))
		}
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			t.Fatalf("parse upgrade tree for %q: %v", definition.Key, err)
		}
		if len(nodes) != 3 {
			t.Errorf("Keeper %q normalized node count = %d, want 3", definition.Key, len(nodes))
		}
	}
	totalNodes := 0
	for _, definition := range release.Definitions {
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			t.Fatalf("parse upgrade tree for %q: %v", definition.Key, err)
		}
		totalNodes += len(nodes)
	}
	if totalNodes != 30 {
		t.Fatalf("CatalogV1 total upgrade nodes = %d, want 30", totalNodes)
	}
}

func TestCatalogV2ConfiguresFourEligibleSectors(t *testing.T) {
	t.Parallel()

	release := CatalogV2()
	if err := Validate(release); err != nil {
		t.Fatalf("Validate(CatalogV2()): %v", err)
	}
	if release.Version != 2 || len(release.Definitions) != 13 || len(release.Baskets) != 13 {
		t.Fatalf("CatalogV2 shape = version:%d definitions:%d baskets:%d", release.Version, len(release.Definitions), len(release.Baskets))
	}
	totalNodes := 0
	for _, definition := range release.Definitions {
		nodes, err := parseUpgradeTree(definition.UpgradeTree)
		if err != nil {
			t.Fatalf("parse upgrade tree for %q: %v", definition.Key, err)
		}
		totalNodes += len(nodes)
	}
	if totalNodes != 39 {
		t.Fatalf("CatalogV2 total upgrade nodes = %d, want 39", totalNodes)
	}
	counts := map[string]int{}
	for _, basket := range release.Baskets {
		if basket.BenchmarkEligible {
			counts[basket.Sector]++
		}
	}
	for _, key := range []string{"CREST", "EMBER", "CURRENT", "HARBOR"} {
		if counts[key] < 2 {
			t.Errorf("%s eligible basket count = %d, want at least 2", key, counts[key])
		}
	}
}

func sameWeights(got, want map[string]int64) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantWeight := range want {
		if got[key] != wantWeight {
			return false
		}
	}
	return true
}
