/** KV-backed execution wrapper for the pure content cutover planner. */

import {
  BaselineModifierKey,
  BaselineObjectiveKey,
  BaselineRuleSetKey,
  sourceCatalogForVersion,
  SourceCatalogV2,
} from "./releases.ts";
import { catalogV1 } from "./keeper_catalog_v1.ts";
import { catalogV2 } from "./keeper_catalog_v2.ts";
import { keeperChecksum } from "./keeper_checksum.ts";
import {
  buildCutoverPlan,
  CutoverError,
  type CutoverParams,
  type CutoverPlan,
  type CutoverResult,
} from "./cutover.ts";
import type { ContentRepository } from "../repositories/content_repository.ts";
import type { CutoverPersistence } from "../repositories/cutover_repository.ts";

export interface ExecutedCutoverResult extends CutoverResult {
  targetChecksum: string;
  applied: boolean;
}

export class ContentCutoverService {
  #content: ContentRepository;
  #cutover: CutoverPersistence;

  constructor(content: ContentRepository, cutover: CutoverPersistence) {
    this.#content = content;
    this.#cutover = cutover;
  }

  async run(
    params: CutoverParams & { apply?: boolean },
  ): Promise<ExecutedCutoverResult> {
    if (
      params.sourceVersion <= 0 || params.targetVersion <= 0 ||
      params.sourceVersion === params.targetVersion
    ) {
      throw new CutoverError(
        "invalid-parameters",
        "source and target versions must be positive and differ",
      );
    }
    const target = await this.#content.getRecord(params.targetVersion);
    if (!target || target.status !== "PUBLISHED") {
      throw new CutoverError(
        "missing-target",
        `content release ${params.targetVersion} does not exist or is not published`,
      );
    }
    if (target.checksumSchemaVersion !== 2) {
      throw new CutoverError(
        "target-not-schema-two",
        `content release ${params.targetVersion} does not use checksum schema 2`,
      );
    }
    const sourceCatalog = sourceCatalogForVersion(params.sourceVersion);
    if (!sourceCatalog) {
      throw new CutoverError(
        "unknown-source-version",
        `source version ${params.sourceVersion} is not a recognized legacy source`,
      );
    }
    const expectedCatalog = sourceCatalog === SourceCatalogV2
      ? catalogV2()
      : catalogV1();
    expectedCatalog.version = params.targetVersion;
    if (
      await keeperChecksum(expectedCatalog) !==
        await keeperChecksum(target.release.keeper)
    ) {
      throw new CutoverError(
        "target-mismatch",
        `content release ${params.targetVersion} does not embed the approved ${sourceCatalog} keeper catalog`,
      );
    }
    const strategyByKey = new Map(
      target.release.strategies.map((strategy) => [strategy.key, strategy.key]),
    );
    const snapshot = await this.#cutover.loadSnapshot();
    const plan = buildCutoverPlan({
      ...params,
      sourceCatalog,
      expectedModifierId: BaselineModifierKey,
      expectedObjectiveId: BaselineObjectiveKey,
      expectedRulesetId: BaselineRuleSetKey,
      strategyByKey,
      tides: snapshot.tides,
      projections: snapshot.projections,
    });
    const apply = params.apply === true;
    if (apply && !plan.result.idempotent) {
      await this.#cutover.apply(
        plan.tideUpdates,
        plan.availability,
        plan.selections,
      );
    }
    return {
      ...plan.result,
      targetChecksum: target.checksum,
      applied: apply && !plan.result.idempotent,
    };
  }
}

export function planWasApplied(
  plan: CutoverPlan,
  apply: boolean,
): boolean {
  return apply && !plan.result.idempotent;
}
