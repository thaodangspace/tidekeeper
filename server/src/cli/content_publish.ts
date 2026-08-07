/** Deno replacement for cmd/content-publish/main.go. */

import {
  BaselineV1Version,
  BaselineV2Version,
  completeReleaseV1,
  completeReleaseV2,
} from "../content/releases.ts";
import { ContentPublisher, type PublishResult } from "../content/publisher.ts";
import { Store } from "../repositories/kv.ts";

export const releaseNames = [
  "keepers-v1",
  "sectors-v2",
  "baseline-v1",
  "baseline-v2",
] as const;

export type ReleaseName = typeof releaseNames[number];

export function releaseForName(name: ReleaseName) {
  switch (name) {
    case "keepers-v1":
      return completeReleaseV1(1);
    case "sectors-v2":
      return completeReleaseV2(2);
    case "baseline-v1":
      return completeReleaseV1(BaselineV1Version);
    case "baseline-v2":
      return completeReleaseV2(BaselineV2Version);
  }
}

export function parseReleaseName(args: string[]): ReleaseName | null {
  if (args.length !== 2 || args[0] !== "--release") return null;
  const name = args[1];
  return releaseNames.includes(name as ReleaseName)
    ? name as ReleaseName
    : null;
}

export function formatPublishResult(result: PublishResult): string {
  return [
    `content release ${result.version} published (idempotent=${result.idempotent} checksum=${result.checksum} modifiers=${result.modifierCount} objectives=${result.objectiveCount} game_rule_sets=${result.gameRuleSetCount})`,
  ].join("\n");
}

export async function runContentPublish(
  args: string[],
  kvPath?: string,
  output: (line: string) => void = console.log,
): Promise<number> {
  const name = parseReleaseName(args);
  if (!name) {
    output(
      "--release=keepers-v1, sectors-v2, baseline-v1, or baseline-v2 is required",
    );
    return 2;
  }
  const kv = await Deno.openKv(kvPath);
  try {
    const result = await new ContentPublisher(new Store(kv)).publish(
      releaseForName(name),
    );
    output(formatPublishResult(result));
    return 0;
  } catch (error) {
    output(
      `content publication failed: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
    return 1;
  } finally {
    kv.close();
  }
}

if (import.meta.main) {
  Deno.exit(await runContentPublish(Deno.args));
}
