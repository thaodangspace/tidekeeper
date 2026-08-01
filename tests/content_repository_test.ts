import { assertEquals, assertRejects } from "@std/assert";
import { completeReleaseV1 } from "../src/content/releases.ts";
import { ContentPublisher } from "../src/content/publisher.ts";
import { ContentRepository } from "../src/repositories/content_repository.ts";
import { Store } from "../src/repositories/kv.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

Deno.test("content repository: reads published releases and stable definitions", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const release = completeReleaseV1(10);
    await new ContentPublisher(store).publish(release);
    const repository = new ContentRepository(store);
    const loaded = await repository.loadRelease(10);
    assertEquals(loaded.version, 10);
    assertEquals(
      (await repository.getModifier(10, "standard_conditions"))?.key,
      "standard_conditions",
    );
    assertEquals(
      (await repository.getKeeperDefinition(10, "crest_sovereign"))?.key,
      "crest_sovereign",
    );
    assertEquals(await repository.getStrategy(10, "missing"), null);
  } finally {
    kv.close();
  }
});

Deno.test("content repository: missing release is not found", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const repository = new ContentRepository(new Store(kv));
    const error = await assertRejects<TidekeepersError>(
      () => repository.loadRelease(99),
      TidekeepersError,
    );
    assertEquals(error.code, "CONTENT_RELEASE_NOT_FOUND");
  } finally {
    kv.close();
  }
});
