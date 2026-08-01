# Gameplay-content cutover

The Deno KV cutover tool recognizes legacy schema-1 Daily Tide projections and
maps them to immutable schema-2 content references. It is a dry run by default;
`--apply` performs the optimistic, atomic KV update.

## Publish targets

```sh
deno task content:publish -- --release baseline-v1 # release 3
deno task content:publish -- --release baseline-v2 # release 4
```

## Plan and apply

```sh
deno task content:cutover -- --source-version 1 --target-version 3
deno task content:cutover -- --source-version 2 --target-version 4
deno task content:cutover -- --source-version 1 --target-version 3 --apply
```

The report includes source and target versions, recognized fingerprints, source
catalog, target checksum, and Tide/state/strategy counts. Re-running a completed
cutover is idempotent. Concurrent changes fail with `CUTOVER_CONFLICT` rather
than overwriting newer state.

## Recognized legacy sources

| Source version | Catalog | Projection identity | Fingerprint |
| --- | --- | --- | --- |
| 1 | v1 | `mod_default` / `obj_default` | `migration-seeded-development` |
| 1 | v1 | `standard_conditions` / `navigate` | `keepers-v1` |
| 2 | v2 | `standard_conditions` / `navigate` | `sectors-v2` |

Unknown source versions, malformed projections, contradictory projections,
partial mappings, and unknown strategies fail closed before mutation.

## Verification

```sh
deno task fmt:check
deno task lint
deno task check
deno task test
```

Cutover planner, KV persistence, idempotency, concurrency checks, CLI parsing,
and report formatting are covered by `tests/cutover*_test.ts` and
`tests/content_cutover_cli_test.ts`.
