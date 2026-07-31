-- Separate permanent Keeper archetype unlocks from Voyage-owned instances.
--
-- Meta progression lives in `player_keeper_unlocks`; each gameplay Keeper
-- instance belongs to exactly one Voyage, stores its exact immutable definition
-- version, its authoritative upgrade node, and acquisition provenance. Legacy
-- development rows are converted deterministically: player-only records become
-- one stable unlock per archetype, and Voyage-linked records are validated
-- against the owning Voyage before being retained as day-one STARTER grants.

-- ── Stable-key composite identity for unlock provenance ─────────────────────
-- Enables the composite FK guaranteeing a stored keeper_key matches the exact
-- historical definition version referenced by an unlock.

CREATE UNIQUE INDEX keeper_definition_versions_id_keeper_key_idx
    ON keeper_definition_versions (id, keeper_key);

-- ── Permanent Keeper archetype unlocks ──────────────────────────────────────

CREATE TABLE player_keeper_unlocks (
    player_id uuid NOT NULL,
    keeper_key text NOT NULL,
    unlocked_definition_version_id uuid NOT NULL,
    unlock_source text NOT NULL,
    unlocked_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT player_keeper_unlocks_pkey
        PRIMARY KEY (player_id, keeper_key),
    CONSTRAINT player_keeper_unlocks_player_fk
        FOREIGN KEY (player_id) REFERENCES players (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT player_keeper_unlocks_stable_key_fk
        FOREIGN KEY (unlocked_definition_version_id, keeper_key)
        REFERENCES keeper_definition_versions (id, keeper_key)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT player_keeper_unlocks_keeper_key_check
        CHECK (keeper_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT player_keeper_unlocks_source_check
        CHECK (unlock_source IN ('SHOP', 'STARTER', 'REWARD', 'EVENT', 'ADMIN', 'LEGACY_MIGRATION')),
    CONSTRAINT player_keeper_unlocks_timestamp_order_check
        CHECK (unlocked_at <= created_at AND updated_at >= created_at)
);

CREATE INDEX player_keeper_unlocks_player_idx
    ON player_keeper_unlocks (player_id, unlocked_at, keeper_key);

-- ── Migrate legacy player-only records into stable unlocks ──────────────────
-- One unlock per (player_id, keeper_key) using the earliest deterministic
-- (acquired_at, id) record and its exact definition version. Player-only
-- gameplay copies are removed before Voyage ownership becomes non-null.

INSERT INTO player_keeper_unlocks (player_id, keeper_key, unlocked_definition_version_id, unlock_source, unlocked_at)
SELECT DISTINCT ON (i.player_id, d.keeper_key)
    i.player_id,
    d.keeper_key,
    i.keeper_definition_version_id,
    'LEGACY_MIGRATION',
    i.acquired_at
FROM keeper_instances AS i
JOIN keeper_definition_versions AS d ON d.id = i.keeper_definition_version_id
WHERE i.voyage_id IS NULL
ORDER BY i.player_id, d.keeper_key, i.acquired_at ASC, i.id ASC;

DELETE FROM keeper_instances WHERE voyage_id IS NULL;

-- ── Validate retained Voyage ownership ──────────────────────────────────────
-- A Voyage-linked instance must be owned by the same player as its Voyage.
-- Mismatches and dangling references abort the migration with evidence instead
-- of silently reassigning run history.

DO $$
DECLARE
    mismatched bigint;
    missing bigint;
BEGIN
    SELECT count(*) INTO mismatched
    FROM keeper_instances AS i
    JOIN voyages AS v ON v.id = i.voyage_id
    WHERE i.player_id IS DISTINCT FROM v.player_id;

    IF mismatched > 0 THEN
        RAISE EXCEPTION
            'keeper ownership migration aborted: % instance(s) reference a voyage owned by a different player',
            mismatched;
    END IF;

    SELECT count(*) INTO missing
    FROM keeper_instances AS i
    LEFT JOIN voyages AS v ON v.id = i.voyage_id
    WHERE v.id IS NULL;

    IF missing > 0 THEN
        RAISE EXCEPTION
            'keeper ownership migration aborted: % instance(s) reference a missing voyage',
            missing;
    END IF;
END;
$$;

-- ── Retained instances: add issue-#3 state columns ──────────────────────────

ALTER TABLE keeper_instances
    ADD COLUMN upgrade_node_key text,
    ADD COLUMN acquired_day smallint,
    ADD COLUMN acquired_source text,
    ADD COLUMN created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT transaction_timestamp();

-- Every retained Voyage-linked instance was created by the pre-issue-#3 Voyage
-- producer and is therefore a day-one STARTER grant at its definition's root.

UPDATE keeper_instances AS i
SET upgrade_node_key = n.node_key,
    acquired_day = 1,
    acquired_source = 'STARTER',
    created_at = i.acquired_at,
    updated_at = i.acquired_at
FROM keeper_definition_upgrade_nodes AS n
WHERE n.keeper_definition_version_id = i.keeper_definition_version_id
  AND n.is_root;

DO $$
DECLARE
    bad_count bigint;
BEGIN
    SELECT count(*) INTO bad_count
    FROM keeper_instances
    WHERE upgrade_node_key IS NULL;

    IF bad_count > 0 THEN
        RAISE EXCEPTION
            'keeper instance backfill aborted: % instance(s) have no root upgrade node in their exact definition',
            bad_count;
    END IF;
END;
$$;

-- ── Final constraints: remove duplicated ownership and level ────────────────

DROP INDEX IF EXISTS keeper_instances_player_acquired_idx;
DROP INDEX IF EXISTS keeper_instances_voyage_idx;

ALTER TABLE keeper_instances
    DROP CONSTRAINT IF EXISTS keeper_instances_player_fk;

ALTER TABLE keeper_instances
    ALTER COLUMN voyage_id SET NOT NULL,
    ALTER COLUMN upgrade_node_key SET NOT NULL,
    ALTER COLUMN acquired_day SET NOT NULL,
    ALTER COLUMN acquired_source SET NOT NULL,
    DROP COLUMN player_id,
    DROP COLUMN level;

ALTER TABLE keeper_instances
    ADD CONSTRAINT keeper_instances_upgrade_node_fk
        FOREIGN KEY (keeper_definition_version_id, upgrade_node_key)
        REFERENCES keeper_definition_upgrade_nodes (keeper_definition_version_id, node_key)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT keeper_instances_acquired_day_check
        CHECK (acquired_day BETWEEN 1 AND 365),
    ADD CONSTRAINT keeper_instances_acquired_source_check
        CHECK (acquired_source IN ('SHOP', 'STARTER', 'REWARD', 'EVENT', 'ADMIN')),
    ADD CONSTRAINT keeper_instances_timestamp_order_check
        CHECK (updated_at >= created_at AND acquired_at <= created_at);

CREATE INDEX keeper_instances_voyage_inventory_idx
    ON keeper_instances (voyage_id, acquired_day, acquired_at, id);
