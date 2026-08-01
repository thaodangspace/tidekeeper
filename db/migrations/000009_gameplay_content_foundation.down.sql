-- Safe rollback of the additive gameplay-content foundation. No authoritative
-- history is deleted: the six version tables are removed only when this
-- migration is applied before any exact reference depends on them, and the
-- legacy text-only selected Strategy column is preserved.

DROP TRIGGER IF EXISTS daily_tide_strategy_versions_published_guard_trigger
    ON daily_tide_strategy_versions;
DROP TRIGGER IF EXISTS daily_tides_exact_content_guard_trigger
    ON daily_tides;

DROP FUNCTION IF EXISTS gameplay_content_guard_strategy_availability();
DROP FUNCTION IF EXISTS gameplay_content_guard_tide_exact_content();
DROP FUNCTION IF EXISTS gameplay_content_require_published_schema_two(bigint);

ALTER TABLE player_daily_states
    DROP CONSTRAINT IF EXISTS player_daily_states_selected_strategy_availability_fk,
    DROP COLUMN IF EXISTS selected_strategy_definition_version_id;

DROP TABLE IF EXISTS daily_tide_strategy_versions;

ALTER TABLE daily_tides
    DROP CONSTRAINT IF EXISTS daily_tides_content_release_fk;

DROP INDEX IF EXISTS daily_tides_exact_content_idx;
DROP INDEX IF EXISTS daily_tides_id_content_version_idx;

ALTER TABLE daily_tides
    DROP COLUMN IF EXISTS modifier_definition_version_id,
    DROP COLUMN IF EXISTS objective_definition_version_id,
    DROP COLUMN IF EXISTS game_rule_set_version_id;

DROP TRIGGER IF EXISTS strategy_definition_versions_draft_only_trigger
    ON strategy_definition_versions;
DROP TRIGGER IF EXISTS relic_definition_versions_draft_only_trigger
    ON relic_definition_versions;
DROP TRIGGER IF EXISTS synergy_definition_versions_draft_only_trigger
    ON synergy_definition_versions;
DROP TRIGGER IF EXISTS daily_modifier_definition_versions_draft_only_trigger
    ON daily_modifier_definition_versions;
DROP TRIGGER IF EXISTS daily_objective_definition_versions_draft_only_trigger
    ON daily_objective_definition_versions;
DROP TRIGGER IF EXISTS game_rule_set_versions_draft_only_trigger
    ON game_rule_set_versions;

DROP TABLE IF EXISTS strategy_definition_versions;
DROP TABLE IF EXISTS relic_definition_versions;
DROP TABLE IF EXISTS synergy_definition_versions;
DROP TABLE IF EXISTS daily_modifier_definition_versions;
DROP TABLE IF EXISTS daily_objective_definition_versions;
DROP TABLE IF EXISTS game_rule_set_versions;

ALTER TABLE content_releases
    DROP CONSTRAINT IF EXISTS content_releases_checksum_schema_version_check,
    DROP COLUMN IF EXISTS checksum_schema_version;
