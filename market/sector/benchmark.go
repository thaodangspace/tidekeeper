package sector

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/thaodangspace/tidekeepers-server/market/canonical"
	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

var ErrDuplicateBasketMapping = errors.New("duplicate basket mapping in sector benchmark")

// BenchmarkBasket is the immutable basket result considered for one benchmark.
type BenchmarkBasket struct {
	BasketMetricID         string
	BasketMappingVersionID string
	Sector                 Key
	BenchmarkEligible      bool
	MetricStatus           string
	NormalizedPerformance  *int64
}

// BenchmarkInput contains all persisted inputs needed by the pure benchmark calculator.
type BenchmarkInput struct {
	DailyTideID string
	Definition  Definition
	Baskets     []BenchmarkBasket
}

// Member records one eligible basket's benchmark contribution and rank.
type Member struct {
	BasketMetricID         string
	BasketMappingVersionID string
	BenchmarkWeight        int64
	NormalizedPerformance  int64
	Contribution           int64
	RankIndex              int64
	Percentile             int64
}

// Benchmark is the deterministic result for one Sector on one Daily Tide.
type Benchmark struct {
	Sector              Key
	Status              Status
	EligibleBasketCount int
	Value               *int64
	Members             []Member
	InputChecksum       [32]byte
}

// BenchmarkCalculator computes equal-weight benchmarks from unique ready baskets.
type BenchmarkCalculator struct{}

func (BenchmarkCalculator) Calculate(input BenchmarkInput) (Benchmark, error) {
	if input.DailyTideID == "" {
		return Benchmark{}, errors.New("daily tide ID is required")
	}
	if err := input.Definition.Validate(); err != nil {
		return Benchmark{}, err
	}

	eligible := make([]BenchmarkBasket, 0, len(input.Baskets))
	seen := make(map[string]struct{}, len(input.Baskets))
	for _, basket := range input.Baskets {
		if basket.Sector != input.Definition.Sector || !basket.BenchmarkEligible || basket.MetricStatus != "READY" {
			continue
		}
		if basket.BasketMappingVersionID == "" || basket.BasketMetricID == "" || basket.NormalizedPerformance == nil {
			return Benchmark{}, errors.New("invalid ready benchmark basket")
		}
		if _, exists := seen[basket.BasketMappingVersionID]; exists {
			return Benchmark{}, fmt.Errorf("%w: %s", ErrDuplicateBasketMapping, basket.BasketMappingVersionID)
		}
		seen[basket.BasketMappingVersionID] = struct{}{}
		eligible = append(eligible, basket)
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].BasketMappingVersionID < eligible[j].BasketMappingVersionID })

	result := Benchmark{Sector: input.Definition.Sector, EligibleBasketCount: len(eligible)}
	if len(eligible) < input.Definition.MinimumEligibleBaskets {
		result.Status = StatusDataIncomplete
		result.InputChecksum = benchmarkChecksum(input, result, eligible)
		return result, nil
	}

	var sum int64
	for _, basket := range eligible {
		var err error
		sum, err = precision.Add(sum, *basket.NormalizedPerformance)
		if err != nil {
			return Benchmark{}, fmt.Errorf("sum normalized performance: %w", err)
		}
	}
	value, err := precision.Divide(sum, int64(len(eligible)))
	if err != nil {
		return Benchmark{}, fmt.Errorf("average normalized performance: %w", err)
	}
	members, err := rankedMembers(eligible)
	if err != nil {
		return Benchmark{}, err
	}
	result.Status = StatusReady
	result.Value = int64Value(value)
	result.Members = members
	result.InputChecksum = benchmarkChecksum(input, result, eligible)
	return result, nil
}

func rankedMembers(baskets []BenchmarkBasket) ([]Member, error) {
	members := make([]Member, len(baskets))
	// Allocate the equal-weight remainder by canonical mapping ID. Integer division
	// deliberately rounds down before the exact remainder is distributed.
	weight := precision.WeightScale / int64(len(baskets))
	remainingWeight := precision.WeightScale - weight*int64(len(baskets))
	for index, basket := range baskets {
		contribution, err := precision.MultiplyDivide(*basket.NormalizedPerformance, weight, precision.WeightScale)
		if err != nil {
			return nil, err
		}
		members[index] = Member{
			BasketMetricID: basket.BasketMetricID, BasketMappingVersionID: basket.BasketMappingVersionID,
			BenchmarkWeight: weight, NormalizedPerformance: *basket.NormalizedPerformance, Contribution: contribution,
		}
		if int64(index) < remainingWeight {
			members[index].BenchmarkWeight++
			contribution, err = precision.MultiplyDivide(*basket.NormalizedPerformance, members[index].BenchmarkWeight, precision.WeightScale)
			if err != nil {
				return nil, err
			}
			members[index].Contribution = contribution
		}
	}

	ranked := append([]Member(nil), members...)
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].NormalizedPerformance != ranked[j].NormalizedPerformance {
			return ranked[i].NormalizedPerformance < ranked[j].NormalizedPerformance
		}
		return ranked[i].BasketMappingVersionID < ranked[j].BasketMappingVersionID
	})
	for start := 0; start < len(ranked); {
		end := start
		for end+1 < len(ranked) && ranked[end+1].NormalizedPerformance == ranked[start].NormalizedPerformance {
			end++
		}
		rankIndex, err := precision.MultiplyDivide(int64(start+end), precision.NormalizedScale, 2)
		if err != nil {
			return nil, err
		}
		percentile, err := precision.Divide(rankIndex, int64(len(ranked)-1))
		if err != nil {
			return nil, err
		}
		for index := start; index <= end; index++ {
			ranked[index].RankIndex = rankIndex
			ranked[index].Percentile = percentile
		}
		start = end + 1
	}
	byMapping := make(map[string]Member, len(ranked))
	for _, member := range ranked {
		byMapping[member.BasketMappingVersionID] = member
	}
	for index := range members {
		members[index] = byMapping[members[index].BasketMappingVersionID]
	}
	return members, nil
}

func benchmarkChecksum(input BenchmarkInput, result Benchmark, baskets []BenchmarkBasket) [32]byte {
	parts := []string{input.DailyTideID, input.Definition.ID, string(input.Definition.Sector), string(input.Definition.BenchmarkMethod),
		strconv.Itoa(input.Definition.MinimumEligibleBaskets), string(result.Status), strconv.Itoa(result.EligibleBasketCount)}
	for _, basket := range baskets {
		parts = append(parts, basket.BasketMetricID, basket.BasketMappingVersionID, strconv.FormatInt(*basket.NormalizedPerformance, 10))
	}
	return canonical.Checksum(parts...)
}

func int64Value(value int64) *int64 { return &value }
