import { assertEquals } from "@std/assert";
import {
  formatCutoverReport,
  parseCutoverArgs,
} from "../src/cli/content_cutover.ts";

Deno.test("content cutover CLI: parses dry-run and apply flags", () => {
  assertEquals(
    parseCutoverArgs(["--source-version", "1", "--target-version", "3"]),
    {
      sourceVersion: 1,
      targetVersion: 3,
      apply: false,
    },
  );
  assertEquals(
    parseCutoverArgs([
      "--apply",
      "--target-version",
      "3",
      "--source-version",
      "1",
    ]),
    {
      sourceVersion: 1,
      targetVersion: 3,
      apply: true,
    },
  );
  assertEquals(
    parseCutoverArgs(["--source-version", "0", "--target-version", "3"]),
    null,
  );
  assertEquals(parseCutoverArgs(["--source-version", "1"]), null);
  assertEquals(parseCutoverArgs(["--unknown", "1"]), null);
});

Deno.test("content cutover CLI: formats idempotent and dry-run reports", () => {
  const base = {
    sourceVersion: 1,
    targetVersion: 3,
    fingerprintLabels: ["migration-seeded-development"],
    sourceCatalog: "v1",
    targetChecksum: "abc",
    applied: false,
    tideCount: 1,
    stateCount: 2,
    strategyCount: 0,
    availabilityCount: 0,
    changedTideCount: 1,
  };
  const dryRun = formatCutoverReport({ ...base, idempotent: false });
  if (!dryRun.includes("dry-run: no changes applied")) {
    throw new Error("missing dry-run output");
  }
  const complete = formatCutoverReport({ ...base, idempotent: true });
  if (!complete.includes("already complete: true")) {
    throw new Error("missing idempotent output");
  }
});
