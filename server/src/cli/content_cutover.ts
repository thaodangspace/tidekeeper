/** Deno replacement for cmd/content-cutover/main.go. */

import { ContentCutoverService } from "../content/cutover_service.ts";
import { ContentRepository } from "../repositories/content_repository.ts";
import { KvCutoverRepository } from "../repositories/cutover_repository.ts";
import { Store } from "../repositories/kv.ts";
import type { ExecutedCutoverResult } from "../content/cutover_service.ts";

export interface CutoverCliArgs {
  sourceVersion: number;
  targetVersion: number;
  apply: boolean;
}

export function parseCutoverArgs(args: string[]): CutoverCliArgs | null {
  let sourceVersion: number | null = null;
  let targetVersion: number | null = null;
  let apply = false;
  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    if (arg === "--apply") {
      apply = true;
      continue;
    }
    if (arg === "--source-version" || arg === "--target-version") {
      const raw = args[++index];
      const value = Number(raw);
      if (!Number.isInteger(value) || value <= 0) return null;
      if (arg === "--source-version") sourceVersion = value;
      else targetVersion = value;
      continue;
    }
    return null;
  }
  if (sourceVersion === null || targetVersion === null) return null;
  return { sourceVersion, targetVersion, apply };
}

export function formatCutoverReport(result: ExecutedCutoverResult): string {
  const lines = [
    `source version: ${result.sourceVersion}`,
    `target version: ${result.targetVersion}`,
    `fingerprint: ${
      result.fingerprintLabels.length
        ? result.fingerprintLabels.join(", ")
        : "-"
    }`,
    `source catalog: ${result.sourceCatalog}`,
    `target checksum: ${result.targetChecksum}`,
  ];
  if (result.idempotent) {
    lines.push("already complete: true", "no changes required");
    return lines.join("\n");
  }
  lines.push(
    `tides: ${result.tideCount}`,
    `states: ${result.stateCount}`,
    `strategies: ${result.strategyCount}`,
    `availability: ${result.availabilityCount}`,
    `tides to change: ${result.changedTideCount}`,
    result.applied
      ? `tides changed: ${result.changedTideCount}`
      : "dry-run: no changes applied",
  );
  return lines.join("\n");
}

export async function runContentCutover(
  args: string[],
  kvPath?: string,
  output: (line: string) => void = console.log,
): Promise<number> {
  const parsed = parseCutoverArgs(args);
  if (!parsed) {
    output(
      "--source-version and --target-version are required positive integers",
    );
    return 2;
  }
  const kv = await Deno.openKv(kvPath);
  try {
    const store = new Store(kv);
    const result = await new ContentCutoverService(
      new ContentRepository(store),
      new KvCutoverRepository(store),
    ).run(parsed);
    output(formatCutoverReport(result));
    return 0;
  } catch (error) {
    output(
      `content cutover failed: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
    return 1;
  } finally {
    kv.close();
  }
}

if (import.meta.main) {
  Deno.exit(await runContentCutover(Deno.args));
}
