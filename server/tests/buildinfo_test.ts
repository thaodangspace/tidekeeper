import { assertEquals } from "@std/assert";
import { buildInfo, buildInfoString } from "../src/buildinfo.ts";

Deno.test("buildinfo: exposes stable metadata", () => {
  const info = buildInfo((name) =>
    ({
      SERVICE_VERSION: "  v1 ",
      GIT_COMMIT: "abc123",
      BUILD_TIME: "2026-08-01T00:00:00Z",
    } as Record<string, string>)[name]
  );
  assertEquals(info, {
    version: "v1",
    commit: "abc123",
    buildTime: "2026-08-01T00:00:00Z",
  });
  assertEquals(
    buildInfoString(info),
    "tidekeepers-api version=v1 commit=abc123 built=2026-08-01T00:00:00Z",
  );
});
