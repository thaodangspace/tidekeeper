import { assertEquals } from "@std/assert";
import {
  formatPublishResult,
  parseReleaseName,
  releaseForName,
  runContentPublish,
} from "../src/cli/content_publish.ts";

Deno.test("content publish CLI: selects the four approved release names", () => {
  assertEquals(parseReleaseName(["--release", "keepers-v1"]), "keepers-v1");
  assertEquals(parseReleaseName(["--release", "sectors-v2"]), "sectors-v2");
  assertEquals(parseReleaseName(["--release", "baseline-v1"]), "baseline-v1");
  assertEquals(parseReleaseName(["--release", "baseline-v2"]), "baseline-v2");
  assertEquals(parseReleaseName(["--release", "unknown"]), null);
  assertEquals(parseReleaseName([]), null);
  assertEquals(releaseForName("keepers-v1").version, 1);
  assertEquals(releaseForName("sectors-v2").version, 2);
});

Deno.test("content publish CLI: publishes idempotently to KV", async () => {
  const output: string[] = [];
  const kvPath = await Deno.makeTempFile({ prefix: "tidekeepers-content-" });
  try {
    assertEquals(
      await runContentPublish(
        ["--release", "baseline-v1"],
        kvPath,
        (line) => output.push(line),
      ),
      0,
    );
    assertEquals(
      await runContentPublish(
        ["--release", "baseline-v1"],
        kvPath,
        (line) => output.push(line),
      ),
      0,
    );
    assertEquals(output.length, 2);
    if (!output[1]?.includes("idempotent=true")) {
      throw new Error("second publication was not idempotent");
    }
  } finally {
    await Deno.remove(kvPath);
  }
});

Deno.test("content publish CLI: formats publication output", () => {
  assertEquals(
    formatPublishResult({
      version: 4,
      checksum: "abc",
      basketMappingCount: 1,
      keeperDefinitionCount: 1,
      componentCount: 1,
      nodeCount: 1,
      strategyCount: 0,
      relicCount: 0,
      synergyCount: 0,
      modifierCount: 1,
      objectiveCount: 1,
      gameRuleSetCount: 1,
      idempotent: false,
    }),
    "content release 4 published (idempotent=false checksum=abc modifiers=1 objectives=1 game_rule_sets=1)",
  );
});
