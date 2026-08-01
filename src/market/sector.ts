/** Pure four-sector market calculation types and calculators (ported from market/sector). */

import * as precision from "./precision.ts";
import { canonicalChecksum } from "./canonical.ts";

export type SectorKey = "CREST" | "EMBER" | "CURRENT" | "HARBOR";

export const Crest: SectorKey = "CREST";
export const Ember: SectorKey = "EMBER";
export const Current: SectorKey = "CURRENT";
export const Harbor: SectorKey = "HARBOR";

/** Returns the four MVP sectors in canonical order. */
export function targetKeys(): SectorKey[] {
  return [Crest, Ember, Current, Harbor];
}

/** Reports whether key is part of the four-sector MVP. */
export function isCalculationTarget(key: string): boolean {
  return key === Crest || key === Ember || key === Current || key === Harbor;
}

export const EqualWeight = "EQUAL_WEIGHT";

export type Status = "READY" | "DATA_INCOMPLETE";

export const StatusReady: Status = "READY";
export const StatusDataIncomplete: Status = "DATA_INCOMPLETE";

export interface Definition {
  id: string;
  sector: SectorKey;
  benchmarkMethod: string;
  minimumEligibleBaskets: number;
  relativeScale: number;
  relativeBlendWeight: number;
  rankBlendWeight: number;
  scoreCap: number;
}

/** Confirms a definition has the phase-one invariant shape. */
export function validateDefinition(definition: Definition): void {
  if (!isCalculationTarget(definition.sector)) {
    throw new Error("unsupported calculation sector");
  }
  if (
    definition.benchmarkMethod !== EqualWeight ||
    definition.minimumEligibleBaskets < 2 ||
    definition.relativeScale <= 0 || definition.scoreCap <= 0 ||
    definition.relativeBlendWeight < 0 || definition.rankBlendWeight < 0
  ) {
    throw new Error("invalid sector definition");
  }
  const blend = precision.add(
    definition.relativeBlendWeight,
    definition.rankBlendWeight,
  );
  if (blend !== precision.NormalizedScale) {
    throw new Error("invalid sector blend weights");
  }
}

export interface BenchmarkBasket {
  basketMetricId: string;
  basketMappingVersionId: string;
  sector: SectorKey;
  benchmarkEligible: boolean;
  metricStatus: string;
  normalizedPerformance: number | null;
}

export interface BenchmarkInput {
  dailyTideId: string;
  definition: Definition;
  baskets: BenchmarkBasket[];
}

export interface Member {
  basketMetricId: string;
  basketMappingVersionId: string;
  benchmarkWeight: number;
  normalizedPerformance: number;
  contribution: number;
  rankIndex: number;
  percentile: number;
}

export interface Benchmark {
  sector: SectorKey;
  status: Status;
  eligibleBasketCount: number;
  value: number | null;
  members: Member[];
  inputChecksum: string;
}

export class BenchmarkCalculator {
  /** Computes equal-weight benchmarks from unique ready baskets. */
  async calculate(input: BenchmarkInput): Promise<Benchmark> {
    if (input.dailyTideId === "") {
      throw new Error("daily tide ID is required");
    }
    validateDefinition(input.definition);

    const eligible: BenchmarkBasket[] = [];
    const seen = new Set<string>();
    for (const basket of input.baskets) {
      if (
        basket.sector !== input.definition.sector ||
        !basket.benchmarkEligible || basket.metricStatus !== "READY"
      ) {
        continue;
      }
      if (
        basket.basketMappingVersionId === "" || basket.basketMetricId === "" ||
        basket.normalizedPerformance === null
      ) {
        throw new Error("invalid ready benchmark basket");
      }
      if (seen.has(basket.basketMappingVersionId)) {
        throw new Error(
          `duplicate basket mapping in sector benchmark: ${basket.basketMappingVersionId}`,
        );
      }
      seen.add(basket.basketMappingVersionId);
      eligible.push(basket);
    }
    eligible.sort((a, b) =>
      a.basketMappingVersionId < b.basketMappingVersionId
        ? -1
        : a.basketMappingVersionId > b.basketMappingVersionId
        ? 1
        : 0
    );

    const result: Benchmark = {
      sector: input.definition.sector,
      status: StatusDataIncomplete,
      eligibleBasketCount: eligible.length,
      value: null,
      members: [],
      inputChecksum: "",
    };
    if (eligible.length < input.definition.minimumEligibleBaskets) {
      result.inputChecksum = await benchmarkChecksum(input, result, eligible);
      return result;
    }

    let sum = 0;
    for (const basket of eligible) {
      sum = precision.add(sum, basket.normalizedPerformance!);
    }
    const value = precision.divide(sum, eligible.length);
    const members = rankedMembers(eligible);
    result.status = StatusReady;
    result.value = value;
    result.members = members;
    result.inputChecksum = await benchmarkChecksum(input, result, eligible);
    return result;
  }
}

function rankedMembers(baskets: BenchmarkBasket[]): Member[] {
  const members: Member[] = [];
  const weight = Math.floor(precision.WeightScale / baskets.length);
  const remainingWeight = precision.WeightScale - weight * baskets.length;
  for (let index = 0; index < baskets.length; index++) {
    const basket = baskets[index]!;
    let contribution = precision.multiplyDivide(
      basket.normalizedPerformance!,
      weight,
      precision.WeightScale,
    );
    let effectiveWeight = weight;
    if (index < remainingWeight) {
      effectiveWeight = weight + 1;
      contribution = precision.multiplyDivide(
        basket.normalizedPerformance!,
        effectiveWeight,
        precision.WeightScale,
      );
    }
    members.push({
      basketMetricId: basket.basketMetricId,
      basketMappingVersionId: basket.basketMappingVersionId,
      benchmarkWeight: effectiveWeight,
      normalizedPerformance: basket.normalizedPerformance!,
      contribution,
      rankIndex: 0,
      percentile: 0,
    });
  }

  const ranked = [...members].sort((a, b) => {
    if (a.normalizedPerformance !== b.normalizedPerformance) {
      return a.normalizedPerformance < b.normalizedPerformance ? -1 : 1;
    }
    return a.basketMappingVersionId < b.basketMappingVersionId ? -1 : 1;
  });
  for (let start = 0; start < ranked.length;) {
    let end = start;
    while (
      end + 1 < ranked.length &&
      ranked[end + 1]!.normalizedPerformance ===
        ranked[start]!.normalizedPerformance
    ) {
      end++;
    }
    const rankIndex = precision.multiplyDivide(
      start + end,
      precision.NormalizedScale,
      2,
    );
    const percentile = precision.divide(rankIndex, ranked.length - 1);
    for (let index = start; index <= end; index++) {
      ranked[index]!.rankIndex = rankIndex;
      ranked[index]!.percentile = percentile;
    }
    start = end + 1;
  }
  const byMapping = new Map<string, Member>();
  for (const member of ranked) {
    byMapping.set(member.basketMappingVersionId, member);
  }
  for (let index = 0; index < members.length; index++) {
    members[index] = byMapping.get(members[index]!.basketMappingVersionId)!;
  }
  return members;
}

async function benchmarkChecksum(
  input: BenchmarkInput,
  result: Benchmark,
  baskets: BenchmarkBasket[],
): Promise<string> {
  const parts = [
    input.dailyTideId,
    input.definition.id,
    input.definition.sector,
    input.definition.benchmarkMethod,
    String(input.definition.minimumEligibleBaskets),
    result.status,
    String(result.eligibleBasketCount),
  ];
  for (const basket of baskets) {
    parts.push(
      basket.basketMetricId,
      basket.basketMappingVersionId,
      String(basket.normalizedPerformance),
    );
  }
  return canonicalChecksum(parts);
}

export interface ScoreInput {
  definition: Definition;
  keeperNormalizedPerformance: number;
  sectorBenchmark: number;
  sectorPercentile: number;
}

export interface ScoreOutput {
  relativePerformance: number;
  relativeComponent: number;
  rankComponent: number;
  normalizedScore: number;
  scorePoints: number;
}

export class ScoreCalculator {
  /** Applies the versioned relative/rank score curve. */
  score(input: ScoreInput): ScoreOutput {
    validateDefinition(input.definition);
    if (
      input.sectorPercentile < 0 ||
      input.sectorPercentile > precision.NormalizedScale
    ) {
      throw new Error("sector percentile out of bounds");
    }
    const relative = precision.subtract(
      input.keeperNormalizedPerformance,
      input.sectorBenchmark,
    );
    let relativeComponent = precision.multiplyDivide(
      relative,
      precision.NormalizedScale,
      input.definition.relativeScale,
    );
    relativeComponent = precision.clamp(
      relativeComponent,
      -precision.NormalizedScale,
      precision.NormalizedScale,
    );
    const doubledPercentile = precision.multiplyDivide(
      input.sectorPercentile,
      2,
      1,
    );
    const rankComponent = precision.subtract(
      doubledPercentile,
      precision.NormalizedScale,
    );
    const relativeContribution = precision.multiplyDivide(
      relativeComponent,
      input.definition.relativeBlendWeight,
      precision.NormalizedScale,
    );
    const rankContribution = precision.multiplyDivide(
      rankComponent,
      input.definition.rankBlendWeight,
      precision.NormalizedScale,
    );
    let normalized = precision.add(relativeContribution, rankContribution);
    normalized = precision.clamp(
      normalized,
      -input.definition.scoreCap,
      input.definition.scoreCap,
    );
    const points = precision.multiplyDivide(
      normalized,
      100,
      precision.NormalizedScale,
    );
    return {
      relativePerformance: relative,
      relativeComponent,
      rankComponent,
      normalizedScore: normalized,
      scorePoints: points,
    };
  }
}
