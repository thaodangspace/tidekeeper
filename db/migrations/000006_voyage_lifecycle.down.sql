DROP TRIGGER IF EXISTS voyages_terminal_immutability_trigger ON voyages;
DROP FUNCTION IF EXISTS voyage_guard_terminal_immutability();

DROP TABLE IF EXISTS voyage_lifecycle_events;
DROP TABLE IF EXISTS voyage_ledger_entries;

DROP INDEX IF EXISTS idempotency_keys_stale_idx;
DROP TABLE IF EXISTS idempotency_keys;

ALTER TABLE voyages
    ADD COLUMN definition_version_key text,
    ADD CONSTRAINT voyages_definition_version_key_check
        CHECK (char_length(definition_version_key) BETWEEN 1 AND 128
            AND definition_version_key ~ '^[A-Za-z][A-Za-z0-9_.:-]*$');

UPDATE voyages SET definition_version_key = 'legacy' WHERE definition_version_key IS NULL;

ALTER TABLE voyages ALTER COLUMN definition_version_key SET NOT NULL;

ALTER TABLE voyages
    DROP CONSTRAINT IF EXISTS voyages_definition_version_fk,
    DROP COLUMN IF EXISTS voyage_definition_version_id;

DROP TRIGGER IF EXISTS voyage_definition_versions_published_immutable_trigger ON voyage_definition_versions;
DROP FUNCTION IF EXISTS voyage_definition_guard_published_immutability();

DROP TRIGGER IF EXISTS voyage_definition_starter_keepers_draft_only_trigger ON voyage_definition_starter_keepers;
DROP TRIGGER IF EXISTS voyage_definition_initial_shop_offers_draft_only_trigger ON voyage_definition_initial_shop_offers;
DROP FUNCTION IF EXISTS voyage_definition_guard_child_mutation();
DROP FUNCTION IF EXISTS voyage_definition_require_draft_status();

DROP TABLE IF EXISTS voyage_definition_initial_shop_offers;
DROP TABLE IF EXISTS voyage_definition_starter_keepers;
DROP TABLE IF EXISTS voyage_definition_versions;

DROP INDEX IF EXISTS voyage_definition_versions_published_idx;
