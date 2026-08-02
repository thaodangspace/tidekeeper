import { assertEquals, assertRejects } from "@std/assert";
import { completeReleaseV1 } from "../src/content/releases.ts";
import { ContentPublisher } from "../src/content/publisher.ts";
import { Store } from "../src/repositories/kv.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

Deno.test("content publisher: publishes immutable releases idempotently", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const publisher = new ContentPublisher(new Store(kv));
    const release = completeReleaseV1(30);
    const first = await publisher.publish(
      release,
      new Date("2026-08-01T00:00:00Z"),
    );
    assertEquals(first.idempotent, false);
    assertEquals(first.version, 30);
    assertEquals(first.checksum.length, 64);

    const second = await publisher.publish(
      completeReleaseV1(30),
      new Date("2026-08-02T00:00:00Z"),
    );
    assertEquals(second.idempotent, true);
    assertEquals(second.checksum, first.checksum);
    assertEquals(
      (await publisher.get(30))?.publishedAt,
      "2026-08-01T00:00:00.000Z",
    );
  } finally {
    kv.close();
  }
});

Deno.test("content publisher: rejects a different release at the same version", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const publisher = new ContentPublisher(new Store(kv));
    await publisher.publish(completeReleaseV1(31));
    const different = completeReleaseV1(31);
    different.objectives[0]!.name = "Changed objective";
    const error = await assertRejects<TidekeepersError>(
      () => publisher.publish(different),
      TidekeepersError,
    );
    assertEquals(error.code, "CONTENT_RELEASE_CONFLICT");
  } finally {
    kv.close();
  }
});
