-- Gameplay-content foundation: additive version tables for Strategies, Relics,
-- Synergies, Daily Modifiers, Daily Objectives, and game-rule sets, plus exact
-- (nullable) Daily Tide and Strategy-selection references. Nothing here deletes
-- legacy text state; exact references are enforced in migration 000010.

-- ── Checksum schema on the aggregate release ────────────────────────────────
-- Historical rows (releases 1 and 2) keep their schema-1 meaning; new complete
-- gameplay releases use schema 2. This does not alter any stored checksum.

ALTER TABLE content_releases
    ADD COLUMN checksum_schema_version smallint NOT NULL DEFAULT 1;

ALTER TABLE content_releases
    ADD CONSTRAINT content_releases_checksum_schema_version_check
        CHECK (checksum_schema_version > 0);

-- ── Versioned Strategy definitions ──────────────────────────────────────────

CREATE TABLE strategy_definition_versions (
    id uuid PRIMARY KEY,
    strategy_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    upside text NOT NULL,
    downside text NOT NULL,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT strategy_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT strategy_definition_versions_key_content_key
        UNIQUE (strategy_key, content_version),
    CONSTRAINT strategy_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT strategy_definition_versions_key_check
        CHECK (strategy_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT strategy_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT strategy_definition_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT strategy_definition_versions_upside_check
        CHECK (char_length(upside) BETWEEN 1 AND 256),
    CONSTRAINT strategy_definition_versions_downside_check
        CHECK (char_length(downside) BETWEEN 1 AND 256),
    CONSTRAINT strategy_definition_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT strategy_definition_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX strategy_definition_versions_content_idx
    ON strategy_definition_versions (content_version, strategy_key);

-- ── Versioned Relic definitions ─────────────────────────────────────────────

CREATE TABLE relic_definition_versions (
    id uuid PRIMARY KEY,
    relic_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT relic_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT relic_definition_versions_key_content_key
        UNIQUE (relic_key, content_version),
    CONSTRAINT relic_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT relic_definition_versions_key_check
        CHECK (relic_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT relic_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT relic_definition_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT relic_definition_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT relic_definition_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX relic_definition_versions_content_idx
    ON relic_definition_versions (content_version, relic_key);

-- ── Versioned Synergy definitions ───────────────────────────────────────────

CREATE TABLE synergy_definition_versions (
    id uuid PRIMARY KEY,
    synergy_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    required_count smallint NOT NULL,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT synergy_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT synergy_definition_versions_key_content_key
        UNIQUE (synergy_key, content_version),
    CONSTRAINT synergy_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT synergy_definition_versions_key_check
        CHECK (synergy_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT synergy_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT synergy_definition_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT synergy_definition_versions_required_count_check
        CHECK (required_count BETWEEN 1 AND 64),
    CONSTRAINT synergy_definition_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT synergy_definition_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX synergy_definition_versions_content_idx
    ON synergy_definition_versions (content_version, synergy_key);

-- ── Versioned Daily Modifier definitions ────────────────────────────────────

CREATE TABLE daily_modifier_definition_versions (
    id uuid PRIMARY KEY,
    modifier_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT daily_modifier_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT daily_modifier_definition_versions_key_content_key
        UNIQUE (modifier_key, content_version),
    CONSTRAINT daily_modifier_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT daily_modifier_definition_versions_key_check
        CHECK (modifier_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT daily_modifier_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT daily_modifier_definition_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT daily_modifier_definition_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT daily_modifier_definition_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX daily_modifier_definition_versions_content_idx
    ON daily_modifier_definition_versions (content_version, modifier_key);

-- ── Versioned Daily Objective definitions ───────────────────────────────────

CREATE TABLE daily_objective_definition_versions (
    id uuid PRIMARY KEY,
    objective_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    progress_label text,
    reward_label text,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT daily_objective_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT daily_objective_definition_versions_key_content_key
        UNIQUE (objective_key, content_version),
    CONSTRAINT daily_objective_definition_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT daily_objective_definition_versions_key_check
        CHECK (objective_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT daily_objective_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT daily_objective_definition_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT daily_objective_definition_versions_progress_label_check
        CHECK (progress_label IS NULL OR char_length(progress_label) BETWEEN 1 AND 512),
    CONSTRAINT daily_objective_definition_versions_reward_label_check
        CHECK (reward_label IS NULL OR char_length(reward_label) BETWEEN 1 AND 512),
    CONSTRAINT daily_objective_definition_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT daily_objective_definition_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX daily_objective_definition_versions_content_idx
    ON daily_objective_definition_versions (content_version, objective_key);

-- ── Versioned game-rule-set definitions ─────────────────────────────────────

CREATE TABLE game_rule_set_versions (
    id uuid PRIMARY KEY,
    rule_set_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    rule_key text NOT NULL,
    rule_config jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT game_rule_set_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT game_rule_set_versions_key_content_key
        UNIQUE (rule_set_key, content_version),
    CONSTRAINT game_rule_set_versions_id_content_key
        UNIQUE (id, content_version),
    CONSTRAINT game_rule_set_versions_key_check
        CHECK (rule_set_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT game_rule_set_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT game_rule_set_versions_description_check
        CHECK (char_length(description) BETWEEN 1 AND 512),
    CONSTRAINT game_rule_set_versions_rule_key_check
        CHECK (rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT game_rule_set_versions_rule_config_check
        CHECK (jsonb_typeof(rule_config) = 'object')
);

CREATE INDEX game_rule_set_versions_content_idx
    ON game_rule_set_versions (content_version, rule_set_key);

-- ── Draft-only mutation guards for every new definition table ───────────────
-- Reuse the existing catalog guard so child rows may only be assembled while
-- their content release is DRAFT.

CREATE TRIGGER strategy_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON strategy_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER relic_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON relic_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER synergy_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON synergy_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER daily_modifier_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON daily_modifier_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER daily_objective_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON daily_objective_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER game_rule_set_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON game_rule_set_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

-- ── Exact Daily Tide references (additive, nullable, deploy-safe) ───────────
-- Composite same-release foreign keys ensure all three exact references belong
-- to daily_tides.content_version. NOT VALID keeps the migration safe on
-- populated databases; migration 000010 validates them after cutover.

CREATE UNIQUE INDEX daily_tides_id_content_version_idx
    ON daily_tides (id, content_version);

ALTER TABLE daily_tides
    ADD COLUMN modifier_definition_version_id uuid,
    ADD COLUMN objective_definition_version_id uuid,
    ADD COLUMN game_rule_set_version_id uuid,
    ADD CONSTRAINT daily_tides_content_release_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT
        NOT VALID,
    ADD CONSTRAINT daily_tides_modifier_same_release_fk
        FOREIGN KEY (modifier_definition_version_id, content_version)
        REFERENCES daily_modifier_definition_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT
        NOT VALID,
    ADD CONSTRAINT daily_tides_objective_same_release_fk
        FOREIGN KEY (objective_definition_version_id, content_version)
        REFERENCES daily_objective_definition_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT
        NOT VALID,
    ADD CONSTRAINT daily_tides_game_rule_set_same_release_fk
        FOREIGN KEY (game_rule_set_version_id, content_version)
        REFERENCES game_rule_set_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT
        NOT VALID;

CREATE INDEX daily_tides_exact_content_idx
    ON daily_tides (content_version, modifier_definition_version_id, objective_definition_version_id, game_rule_set_version_id);

-- ── Exact Strategy availability per Daily Tide ──────────────────────────────
-- Both parents share one content_version so an availability entry can never
-- cross releases.

CREATE TABLE daily_tide_strategy_versions (
    daily_tide_id uuid NOT NULL,
    strategy_definition_version_id uuid NOT NULL,
    content_version bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT daily_tide_strategy_versions_pkey
        PRIMARY KEY (daily_tide_id, strategy_definition_version_id),
    CONSTRAINT daily_tide_strategy_versions_tide_same_release_fk
        FOREIGN KEY (daily_tide_id, content_version)
        REFERENCES daily_tides (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT daily_tide_strategy_versions_strategy_same_release_fk
        FOREIGN KEY (strategy_definition_version_id, content_version)
        REFERENCES strategy_definition_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX daily_tide_strategy_versions_strategy_idx
    ON daily_tide_strategy_versions (strategy_definition_version_id);

-- ── Exact Strategy selection (nullable, additive) ───────────────────────────
-- A non-null selection must be one of that Tide's available exact Strategy
-- versions; clearing selection remains represented by NULL.

ALTER TABLE player_daily_states
    ADD COLUMN selected_strategy_definition_version_id uuid,
    ADD CONSTRAINT player_daily_states_selected_strategy_availability_fk
        FOREIGN KEY (daily_tide_id, selected_strategy_definition_version_id)
        REFERENCES daily_tide_strategy_versions (daily_tide_id, strategy_definition_version_id)
        ON UPDATE RESTRICT ON DELETE RESTRICT;

-- ── Authoritative reference guards ──────────────────────────────────────────
-- New exact Tide references and Strategy availability rows require a PUBLISHED
-- checksum-schema-2 release. Pre-existing legacy rows keep their NULL exact
-- columns and survive until migration 000010 cutover.

CREATE FUNCTION gameplay_content_require_published_schema_two(required_version bigint)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    release_status text;
    release_schema smallint;
BEGIN
    SELECT status, checksum_schema_version
    INTO release_status, release_schema
    FROM content_releases
    WHERE version = required_version;

    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '23503',
            MESSAGE = format('content release %s does not exist', required_version);
    END IF;

    IF release_status <> 'PUBLISHED' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('content release %s is not published', required_version);
    END IF;

    IF release_schema <> 2 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('content release %s does not use checksum schema 2', required_version);
    END IF;
END;
$$;

CREATE FUNCTION gameplay_content_guard_tide_exact_content()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.modifier_definition_version_id IS NOT NULL
       OR NEW.objective_definition_version_id IS NOT NULL
       OR NEW.game_rule_set_version_id IS NOT NULL THEN
        PERFORM gameplay_content_require_published_schema_two(NEW.content_version);
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER daily_tides_exact_content_guard_trigger
    BEFORE INSERT OR UPDATE ON daily_tides
    FOR EACH ROW EXECUTE FUNCTION gameplay_content_guard_tide_exact_content();

CREATE FUNCTION gameplay_content_guard_strategy_availability()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM gameplay_content_require_published_schema_two(NEW.content_version);
    RETURN NEW;
END;
$$;

CREATE TRIGGER daily_tide_strategy_versions_published_guard_trigger
    BEFORE INSERT OR UPDATE ON daily_tide_strategy_versions
    FOR EACH ROW EXECUTE FUNCTION gameplay_content_guard_strategy_availability();
