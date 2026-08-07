/**
 * Pure basket-calculation inputs, evidence, and outputs (ported from
 * market/basket). No database, network, or clock dependency.
 */

import * as precision from "./precision.ts";
import { canonicalChecksum } from "./canonical.ts";

export type MetricStatus = "READY" | "DATA_INCOMPLETE";

export type ComponentStatus =
  | "VALID"
  | "MISSING_OPEN"
  | "MISSING_CLOSE"
  | "OUTSIDE_TOLERANCE"
  | "INVALID_PRICE"
  | "OUTLIER_FLAGGED"
  | "PROVIDER_REJECTED"
  | "DISABLED_BY_MAPPING";

export const MetricStatusReady: MetricStatus = "READY";
export const MetricStatusDataIncomplete: MetricStatus = "DATA_INCOMPLETE";

export const ComponentStatusValid: ComponentStatus = "VALID";
export const ComponentStatusMissingOpen: ComponentStatus = "MISSING_OPEN";
export const ComponentStatusMissingClose: ComponentStatus = "MISSING_CLOSE";
export const ComponentStatusOutsideTolerance: ComponentStatus =
  "OUTSIDE_TOLERANCE";
export const ComponentStatusInvalidPrice: ComponentStatus = "INVALID_PRICE";
export const ComponentStatusOutlierFlagged: ComponentStatus = "OUTLIER_FLAGGED";
export const ComponentStatusProviderRejected: ComponentStatus =
  "PROVIDER_REJECTED";
export const ComponentStatusDisabledByMapping: ComponentStatus =
  "DISABLED_BY_MAPPING";

export type ObservationStatus =
  | "VALID"
  | "INVALID_PRICE"
  | "OUTLIER_FLAGGED"
  | "PROVIDER_REJECTED";

export const ObservationStatusValid: ObservationStatus = "VALID";

export const TurbulencePolicyStaticContentValue = "STATIC_CONTENT_VALUE";

export interface ExpectedTurbulencePolicy {
  id: string;
  type: string;
  staticValue: number;
  floor: number;
}

export interface MappingComponent {
  marketAssetId: string;
  targetWeight: number;
  enabled: boolean;
  sequence: number;
}

export interface MappingVersion {
  id: string;
  key: string;
  sector: string;
  minimumCoveredWeight: number;
  normalizationCap: number;
  benchmarkEligible: boolean;
  expectedTurbulence: ExpectedTurbulencePolicy;
  components: MappingComponent[];
}

export interface Observation {
  id: string;
  providerKey: string;
  marketAssetId: string;
  observedAt: string;
  priceUnits: number;
  status: ObservationStatus;
}

export interface CalculationWindow {
  id: string;
  dailyTideId: string;
  providerKey: string;
  start: string;
  end: string;
  toleranceMs: number;
}

export interface CalculationInput {
  mapping: MappingVersion;
  window: CalculationWindow;
  observations: Observation[];
}

export interface ComponentMetric {
  marketAssetId: string;
  status: ComponentStatus;
  targetWeight: number;
  effectiveWeight: number;
  openObservationId: string;
  closeObservationId: string;
  componentReturn: number | null;
  contribution: number | null;
  qualityFlags: string[];
}

export interface Metric {
  basketMappingVersionId: string;
  sector: string;
  status: MetricStatus;
  coveredWeight: number;
  rawReturn: number | null;
  expectedTurbulence: number | null;
  normalizedPerformance: number | null;
  components: ComponentMetric[];
  inputChecksum: string;
}

/** Confirms a mapping weight uses the shared catalog scale. */
export function validWeight(value: number): boolean {
  return value > 0 && value <= precision.WeightScale;
}

export class Calculator {
  /** Derives a basket metric from immutable observations deterministically. */
  async calculate(input: CalculationInput): Promise<Metric> {
    validateInput(input);

    const components = [...input.mapping.components].sort((a, b) =>
      a.marketAssetId < b.marketAssetId
        ? -1
        : a.marketAssetId > b.marketAssetId
        ? 1
        : 0
    );
    const metrics: ComponentMetric[] = [];
    const validIndexes: number[] = [];
    let coveredWeight = 0;

    for (const component of components) {
      const metric: ComponentMetric = {
        marketAssetId: component.marketAssetId,
        status: ComponentStatusDisabledByMapping,
        targetWeight: component.targetWeight,
        effectiveWeight: 0,
        openObservationId: "",
        closeObservationId: "",
        componentReturn: null,
        contribution: null,
        qualityFlags: [],
      };
      if (!component.enabled) {
        metric.status = ComponentStatusDisabledByMapping;
        metrics.push(metric);
        continue;
      }
      const open = selectObservation(
        input.observations,
        input.window,
        component.marketAssetId,
        input.window.start,
      );
      const close = selectObservation(
        input.observations,
        input.window,
        component.marketAssetId,
        input.window.end,
      );
      if (
        open.status !== ComponentStatusValid || open.observation === null
      ) {
        metric.status = open.status;
        metrics.push(metric);
        continue;
      }
      if (
        close.status !== ComponentStatusValid || close.observation === null
      ) {
        metric.status = close.status === ComponentStatusMissingOpen
          ? ComponentStatusMissingClose
          : close.status;
        metrics.push(metric);
        continue;
      }
      const priceDelta = precision.subtract(
        close.observation.priceUnits,
        open.observation.priceUnits,
      );
      const componentReturn = precision.multiplyDivide(
        priceDelta,
        precision.ReturnScale,
        open.observation.priceUnits,
      );
      metric.status = ComponentStatusValid;
      metric.openObservationId = open.observation.id;
      metric.closeObservationId = close.observation.id;
      metric.componentReturn = componentReturn;
      metrics.push(metric);
      validIndexes.push(metrics.length - 1);
      coveredWeight = precision.add(coveredWeight, component.targetWeight);
    }

    const result: Metric = {
      basketMappingVersionId: input.mapping.id,
      sector: input.mapping.sector,
      status: MetricStatusDataIncomplete,
      coveredWeight,
      rawReturn: null,
      expectedTurbulence: null,
      normalizedPerformance: null,
      components: metrics,
      inputChecksum: "",
    };
    if (coveredWeight < input.mapping.minimumCoveredWeight) {
      result.inputChecksum = await checksum(input, result);
      return result;
    }
    assignEffectiveWeights(result.components, validIndexes, coveredWeight);
    let basketReturn = 0;
    for (const component of result.components) {
      if (component.status !== ComponentStatusValid) {
        continue;
      }
      const contribution = precision.multiplyDivide(
        component.componentReturn!,
        component.effectiveWeight,
        precision.WeightScale,
      );
      component.contribution = contribution;
      basketReturn = precision.add(basketReturn, contribution);
    }
    let effectiveTurbulence = input.mapping.expectedTurbulence.staticValue;
    if (effectiveTurbulence < input.mapping.expectedTurbulence.floor) {
      effectiveTurbulence = input.mapping.expectedTurbulence.floor;
    }
    let normalized = precision.multiplyDivide(
      basketReturn,
      precision.NormalizedScale,
      effectiveTurbulence,
    );
    normalized = precision.clamp(
      normalized,
      -input.mapping.normalizationCap,
      input.mapping.normalizationCap,
    );
    result.status = MetricStatusReady;
    result.rawReturn = basketReturn;
    result.expectedTurbulence = effectiveTurbulence;
    result.normalizedPerformance = normalized;
    result.inputChecksum = await checksum(input, result);
    return result;
  }
}

function validateInput(input: CalculationInput): void {
  const mapping = input.mapping;
  const validSector = ["CREST", "EMBER", "CURRENT", "HARBOR"].includes(
    mapping.sector,
  );
  if (
    mapping.id === "" || mapping.key === "" || !validSector ||
    mapping.minimumCoveredWeight <= 0 ||
    mapping.minimumCoveredWeight > precision.WeightScale ||
    mapping.normalizationCap <= 0 ||
    mapping.expectedTurbulence.type !== TurbulencePolicyStaticContentValue ||
    mapping.expectedTurbulence.staticValue <= 0 ||
    mapping.expectedTurbulence.floor <= 0 || mapping.components.length === 0
  ) {
    throw new Error("invalid basket mapping");
  }
  if (
    input.window.dailyTideId === "" || input.window.providerKey === "" ||
    !(new Date(input.window.start).getTime() <
      new Date(input.window.end).getTime()) ||
    input.window.toleranceMs < 0
  ) {
    throw new Error("invalid calculation window");
  }
  const seen = new Set<string>();
  let totalWeight = 0;
  for (const component of mapping.components) {
    if (
      component.marketAssetId === "" || !validWeight(component.targetWeight)
    ) {
      throw new Error("invalid basket mapping");
    }
    if (seen.has(component.marketAssetId)) {
      throw new Error("invalid basket mapping");
    }
    seen.add(component.marketAssetId);
    totalWeight = precision.add(totalWeight, component.targetWeight);
  }
  if (totalWeight !== precision.WeightScale) {
    throw new Error("invalid basket mapping");
  }
}

function selectObservation(
  observations: Observation[],
  window: CalculationWindow,
  assetId: string,
  target: string,
): { observation: Observation | null; status: ComponentStatus } {
  const targetMs = new Date(target).getTime();
  let nearest: Observation | null = null;
  let nearestDistance = Number.POSITIVE_INFINITY;
  let seenMatchingAsset = false;
  let seenWithinTolerance = false;
  let rejected: ComponentStatus | null = null;
  for (const observation of observations) {
    if (
      observation.marketAssetId !== assetId ||
      observation.providerKey !== window.providerKey
    ) {
      continue;
    }
    seenMatchingAsset = true;
    const distance = Math.abs(
      new Date(observation.observedAt).getTime() - targetMs,
    );
    if (distance > window.toleranceMs) {
      continue;
    }
    seenWithinTolerance = true;
    if (
      observation.status !== ObservationStatusValid ||
      observation.priceUnits <= 0
    ) {
      rejected = observationComponentStatus(observation.status);
      continue;
    }
    if (
      nearest === null || distance < nearestDistance ||
      (distance === nearestDistance && observation.id < nearest.id)
    ) {
      nearest = observation;
      nearestDistance = distance;
    }
  }
  if (nearest !== null) {
    return { observation: nearest, status: ComponentStatusValid };
  }
  if (!seenMatchingAsset) {
    return { observation: null, status: ComponentStatusMissingOpen };
  }
  if (!seenWithinTolerance) {
    return { observation: null, status: ComponentStatusOutsideTolerance };
  }
  if (rejected !== null) {
    return { observation: null, status: rejected };
  }
  return { observation: null, status: ComponentStatusMissingOpen };
}

function observationComponentStatus(
  status: ObservationStatus,
): ComponentStatus {
  switch (status) {
    case "OUTLIER_FLAGGED":
      return ComponentStatusOutlierFlagged;
    case "PROVIDER_REJECTED":
      return ComponentStatusProviderRejected;
    default:
      return ComponentStatusInvalidPrice;
  }
}

function assignEffectiveWeights(
  metrics: ComponentMetric[],
  validIndexes: number[],
  coveredWeight: number,
): void {
  const remainders: { index: number; value: bigint }[] = [];
  let assigned = 0;
  for (const index of validIndexes) {
    const numerator = BigInt(metrics[index]!.targetWeight) *
      BigInt(precision.WeightScale);
    const quotient = numerator / BigInt(coveredWeight);
    const remainderValue = numerator % BigInt(coveredWeight);
    metrics[index]!.effectiveWeight = Number(quotient);
    assigned = precision.add(assigned, metrics[index]!.effectiveWeight);
    remainders.push({ index, value: remainderValue });
  }
  let remaining = precision.subtract(precision.WeightScale, assigned);
  remainders.sort((a, b) => {
    if (a.value !== b.value) {
      return a.value > b.value ? -1 : 1;
    }
    return metrics[a.index]!.marketAssetId < metrics[b.index]!.marketAssetId
      ? -1
      : 1;
  });
  let cursor = 0;
  while (remaining > 0) {
    metrics[remainders[cursor]!.index]!.effectiveWeight++;
    cursor++;
    remaining--;
  }
}

async function checksum(
  input: CalculationInput,
  metric: Metric,
): Promise<string> {
  const parts = [
    input.window.dailyTideId,
    input.mapping.id,
    input.mapping.sector,
    String(input.mapping.minimumCoveredWeight),
    String(input.mapping.normalizationCap),
    input.mapping.expectedTurbulence.id,
    input.mapping.expectedTurbulence.type,
    String(input.mapping.expectedTurbulence.staticValue),
    String(input.mapping.expectedTurbulence.floor),
    metric.status,
    String(metric.coveredWeight),
  ];
  for (const component of metric.components) {
    parts.push(
      component.marketAssetId,
      component.status,
      String(component.targetWeight),
      String(component.effectiveWeight),
      component.openObservationId,
      component.closeObservationId,
    );
    if (component.componentReturn !== null) {
      parts.push(String(component.componentReturn));
    }
    if (component.contribution !== null) {
      parts.push(String(component.contribution));
    }
  }
  return canonicalChecksum(parts);
}
