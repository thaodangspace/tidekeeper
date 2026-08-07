/** Rule registry and descriptor-driven validation (ported from content/rules.go). */

import type { Kind } from "./model.ts";
import { ContentWeightScale, ruleKeyPattern } from "./model.ts";

/** Baseline rule keys registered in the production registry. */
export const RuleStandardConditions = "STANDARD_CONDITIONS";
export const RuleNavigate = "NAVIGATE";
export const RuleStandardRules = "STANDARD_RULES";

/** Strict schema-versioned configuration envelope required by every rule. */
export interface ConfigEnvelope {
  schemaVersion: number;
  parameters: unknown;
}

/** Parameter JSON value types. */
export type ParameterType =
  | "integer"
  | "boolean"
  | "string"
  | "stringList"
  | "reference";

export interface FixedPoint {
  scale: number;
  min: number;
  max: number;
}

export interface ParameterSpec {
  name: string;
  type: ParameterType;
  required: boolean;
  fixed?: FixedPoint;
  refKind?: Kind;
}

export interface TotalSpec {
  fields: string[];
  total: number;
}

/** Descriptor-driven rule specification. */
export interface RuleSpec {
  kind: Kind;
  ruleKey: string;
  schemaVersion: number;
  parameters: ParameterSpec[];
  mutuallyExclusive: string[][];
  total: TotalSpec | null;
}

/** Release context exposing candidate-release indexes for reference checks. */
export class ReleaseContext {
  readonly indexes: Map<Kind, Set<string>>;

  constructor(indexes: Map<Kind, Set<string>>) {
    this.indexes = indexes;
  }

  contains(kind: Kind, key: string): boolean {
    return this.indexes.get(kind)?.has(key) ?? false;
  }
}

interface RuleValidator {
  kind: Kind;
  ruleKey: string;
  schemaVersion: number;
  validateParameters(
    ctx: ReleaseContext,
    key: string,
    parameters: string,
  ): void;
}

/** Descriptor-driven rule validator (ported from SpecValidator). */
export class SpecValidator implements RuleValidator {
  readonly spec: RuleSpec;

  constructor(spec: RuleSpec) {
    this.spec = spec;
  }

  get kind(): Kind {
    return this.spec.kind;
  }

  get ruleKey(): string {
    return this.spec.ruleKey;
  }

  get schemaVersion(): number {
    return this.spec.schemaVersion;
  }

  validateParameters(
    ctx: ReleaseContext,
    _key: string,
    parameters: string,
  ): void {
    if (!isJsonObject(parameters)) {
      throw new Error("parameters must be a JSON object");
    }
    const rawFields: Record<string, unknown> = JSON.parse(parameters);
    const declared = new Map<string, ParameterSpec>();
    for (const spec of this.spec.parameters) {
      declared.set(spec.name, spec);
    }
    for (const name of Object.keys(rawFields)) {
      if (!declared.has(name)) {
        throw new Error(`unknown parameter ${name}`);
      }
    }
    for (const spec of this.spec.parameters) {
      if (!(spec.name in rawFields)) {
        if (spec.required) {
          throw new Error(`missing required parameter ${spec.name}`);
        }
        continue;
      }
      this.validateParameter(ctx, spec, JSON.stringify(rawFields[spec.name]));
    }
    for (const group of this.spec.mutuallyExclusive) {
      let present = 0;
      for (const name of group) {
        if (name in rawFields) {
          present++;
        }
      }
      if (present > 1) {
        throw new Error(
          `parameters ${JSON.stringify(group)} are mutually exclusive`,
        );
      }
    }
    if (this.spec.total !== null) {
      this.validateTotal(rawFields);
    }
  }

  private validateParameter(
    ctx: ReleaseContext,
    spec: ParameterSpec,
    raw: string,
  ): void {
    switch (spec.type) {
      case "integer": {
        const fixed = spec.fixed ??
          { scale: ContentWeightScale, min: 0, max: ContentWeightScale };
        const number = decodeInteger(raw);
        if (number < fixed.min || number > fixed.max) {
          throw new Error(
            `parameter ${spec.name} is outside the range [${fixed.min}, ${fixed.max}]`,
          );
        }
        return;
      }
      case "boolean": {
        const value = JSON.parse(raw);
        if (typeof value !== "boolean") {
          throw new Error(`parameter ${spec.name} must be a boolean`);
        }
        return;
      }
      case "string": {
        const value = JSON.parse(raw);
        if (typeof value !== "string") {
          throw new Error(`parameter ${spec.name} must be a string`);
        }
        return;
      }
      case "stringList": {
        const value = JSON.parse(raw);
        if (
          !Array.isArray(value) ||
          value.some((entry) => typeof entry !== "string")
        ) {
          throw new Error(`parameter ${spec.name} must be a list of strings`);
        }
        return;
      }
      case "reference": {
        const value = JSON.parse(raw);
        if (typeof value !== "string") {
          throw new Error(`parameter ${spec.name} must be a string`);
        }
        if (!ctx.contains(spec.refKind!, value)) {
          throw new Error(
            `parameter ${spec.name} references unknown ${spec.refKind} ${value}`,
          );
        }
        return;
      }
      default:
        throw new Error(`parameter ${spec.name} has an unsupported type`);
    }
  }

  private validateTotal(rawFields: Record<string, unknown>): void {
    let sum = 0;
    for (const field of this.spec.total!.fields) {
      if (!(field in rawFields)) {
        throw new Error(`total requires parameter ${field}`);
      }
      const value = decodeInteger(JSON.stringify(rawFields[field]));
      if (value > Number.MAX_SAFE_INTEGER - sum) {
        throw new Error(
          `total for parameters ${
            JSON.stringify(this.spec.total!.fields)
          } overflows the fixed-point scale`,
        );
      }
      sum += value;
    }
    if (sum !== this.spec.total!.total) {
      throw new Error(
        `parameters ${
          JSON.stringify(this.spec.total!.fields)
        } total ${sum}, want ${this.spec.total!.total}`,
      );
    }
  }
}

/** Immutable registry of rule validators keyed by kind and rule key. */
export class Registry {
  #validators = new Map<string, RuleValidator>();

  constructor(validators: RuleValidator[]) {
    for (const validator of validators) {
      if (validator === null) {
        continue;
      }
      this.#validators.set(`${validator.kind}:${validator.ruleKey}`, validator);
    }
  }

  validateRuleBinding(
    ctx: ReleaseContext,
    kind: Kind,
    key: string,
    ruleKey: string,
    config: string,
  ): void {
    if (!ruleKeyPattern.test(ruleKey)) {
      throw new Error(`invalid rule key ${ruleKey}`);
    }
    const validator = this.#validators.get(`${kind}:${ruleKey}`);
    if (!validator) {
      throw new Error(`unknown rule key ${ruleKey}`);
    }
    if (!isJsonObject(config)) {
      throw new Error("rule configuration must be a JSON object");
    }
    let envelope: ConfigEnvelope;
    try {
      envelope = JSON.parse(config);
    } catch {
      throw new Error("invalid rule configuration envelope");
    }
    if (
      typeof envelope.schemaVersion !== "number" ||
      envelope.schemaVersion !== validator.schemaVersion
    ) {
      throw new Error(
        `unsupported rule schema version ${envelope.schemaVersion} (supported ${validator.schemaVersion})`,
      );
    }
    if (
      envelope.parameters === null || typeof envelope.parameters !== "object" ||
      Array.isArray(envelope.parameters)
    ) {
      throw new Error("rule parameters must be a JSON object");
    }
    validator.validateParameters(ctx, key, JSON.stringify(envelope.parameters));
  }
}

/** Returns the immutable production baseline rule registry. */
export function standardRegistry(): Registry {
  return new Registry([
    new SpecValidator({
      kind: "modifier",
      ruleKey: RuleStandardConditions,
      schemaVersion: 1,
      parameters: [],
      mutuallyExclusive: [],
      total: null,
    }),
    new SpecValidator({
      kind: "objective",
      ruleKey: RuleNavigate,
      schemaVersion: 1,
      parameters: [],
      mutuallyExclusive: [],
      total: null,
    }),
    new SpecValidator({
      kind: "game_rule_set",
      ruleKey: RuleStandardRules,
      schemaVersion: 1,
      parameters: [],
      mutuallyExclusive: [],
      total: null,
    }),
  ]);
}

function decodeInteger(raw: string): number {
  const value = JSON.parse(raw);
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw new Error("must be an integer");
  }
  return value;
}

function isJsonObject(text: string): boolean {
  try {
    const value = JSON.parse(text);
    return value !== null && typeof value === "object" && !Array.isArray(value);
  } catch {
    return false;
  }
}
