# Gameplay-Content Cutover

Phase 4 ships the operational bridge from legacy schema-1 Daily Tide projections
to exact schema-2 references. Operators publish an approved complete baseline and
run `content-cutover` to migrate only recognized legacy Tides in one transaction.
Unknown or incompatible state aborts with offending identifiers and leaves every
row unchanged.

## Deployment gate

Do not tighten exact references until every production row has been cut over.

1. Apply `000009_gameplay_content_foundation.up.sql`. It adds the additive,
   nullable exact-reference columns and versioned definition tables. Legacy rows
   keep their text projections and `NULL` exact columns.
2. Publish the approved complete baseline targets:

   ```sh
   content-publish --release=baseline-v1   # creates schema-2 release 3
   content-publish --release=baseline-v2   # creates schema-2 release 4
   ```

   Release 3 embeds the approved CatalogV1 keeper catalog plus Standard
   Conditions (`standard_conditions`), Navigate (`navigate`), and the standard
   game-rule set (`standard_rules`). Release 4 embeds CatalogV2 the same way.
   Production Strategy/Relic/Synergy collections stay empty until approved
   definitions are supplied; targets may carry additional approved Strategies
   without breaking cutover compatibility.
3. Run the dry-run cutover for each recognized source:

   ```sh
   content-cutover --source-version=1 --target-version=3   # CatalogV1 sources
   content-cutover --source-version=2 --target-version=4   # CatalogV2 sources
   ```

   The report lists source/target versions, recognized fingerprints, the source
   catalog, the target checksum, and Tide/state/strategy counts. No row changes.
4. Approve the report, then execute with `--apply`.
5. Re-run the same command to confirm an idempotent result with zero changed
   rows, then apply the tightening migration 000010.

## Recognized legacy sources

| Source version | Catalog | Recognized projection identity       | Fingerprint               |
| -------------- | ------- | ------------------------------------ | ------------------------- |
| 1              | v1      | `mod_default` / `obj_default`        | `migration-seeded-development` |
| 1              | v1      | `standard_conditions` / `navigate`   | `keepers-v1`              |
| 2              | v2      | `standard_conditions` / `navigate`   | `sectors-v2`              |

Any other source version or projection identity aborts with
`unknown-source-version` or `unknown-source-fingerprint`.

## What cutover does

`Cutover.Run` validates before writing:

- the source version maps to a recognized legacy catalog;
- the target exists, is `PUBLISHED`, uses checksum schema 2, embeds the approved
  source catalog content, and defines the baseline modifier/objective/ruleset;
- every candidate Tide is either fully legacy or fully mapped to the target;
- every projection attached to a source Tide decodes to one consistent
  recognized Modifier/Objective identity and an identical available Strategy set.

Then, in one transaction, it creates exact Strategy availability rows before
mapping selections, sets exact Tide references, and moves each Tide to the target
release. Every `CutoverError` category returns the offending Tide/State/Strategy
identifiers and rolls back so all rows stay unchanged.

## CLI

```sh
content-cutover --source-version=N --target-version=M [--apply]
```

- Dry-run by default; `--apply` writes.
- Exit code 2 with usage when source/target versions are missing or not positive.
- Exit code 1 with a JSON error log (`category` set to the `CutoverError`
  category, otherwise `cutover_failed`) on failure.
- Structured reports never include rule configurations or the database URL.

## Verification

```sh
go test ./cmd/content-publish ./cmd/content-cutover ./content
DATABASE_URL=... go test -tags integration ./test/integration -run 'TestGameplayContentCutover'
```

`TestGameplayContentCutover` covers migration-seeded mapping in one transaction,
strategy availability-before-selection, unknown Strategy/contradictory
projection/unknown fingerprint/target mismatch/partial-state rejection, and
idempotent re-execution.
