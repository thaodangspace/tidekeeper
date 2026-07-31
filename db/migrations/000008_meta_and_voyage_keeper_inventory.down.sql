-- Revert issue-#3 keeper ownership to the post-000006 legacy shape.
--
-- player_id is reconstructed from the owning Voyage and level from the
-- normalized upgrade-node depth plus one, so retained instances roll back
-- deterministically. Discarded player-only gameplay rows converted into stable
-- unlocks cannot be recreated by rollback; their canonical legacy unlock is
-- preserved in player_keeper_unlocks until it is dropped.

-- Recreate legacy ownership columns and backfill deterministically.

ALTER TABLE keeper_instances
    ADD COLUMN player_id uuid,
    ADD COLUMN level smallint;

UPDATE keeper_instances AS i
SET player_id = v.player_id,
    level = (
        SELECT (n.depth + 1)::smallint
        FROM keeper_definition_upgrade_nodes AS n
        WHERE n.keeper_definition_version_id = i.keeper_definition_version_id
          AND n.node_key = i.upgrade_node_key
    )
FROM voyages AS v
WHERE v.id = i.voyage_id;

ALTER TABLE keeper_instances
    ALTER COLUMN player_id SET NOT NULL,
    ALTER COLUMN level SET NOT NULL,
    ADD CONSTRAINT keeper_instances_player_fk
        FOREIGN KEY (player_id) REFERENCES players (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT keeper_instances_level_check
        CHECK (level BETWEEN 1 AND 100);

CREATE INDEX keeper_instances_player_acquired_idx
    ON keeper_instances (player_id, acquired_at, id);

-- Remove issue-#3 objects, returning keeper_instances to its post-000006 shape.

DROP INDEX IF EXISTS keeper_instances_voyage_inventory_idx;

ALTER TABLE keeper_instances
    DROP CONSTRAINT IF EXISTS keeper_instances_upgrade_node_fk,
    DROP CONSTRAINT IF EXISTS keeper_instances_acquired_day_check,
    DROP CONSTRAINT IF EXISTS keeper_instances_acquired_source_check,
    DROP CONSTRAINT IF EXISTS keeper_instances_timestamp_order_check,
    DROP COLUMN upgrade_node_key,
    DROP COLUMN acquired_day,
    DROP COLUMN acquired_source,
    DROP COLUMN created_at,
    DROP COLUMN updated_at,
    ALTER COLUMN voyage_id DROP NOT NULL;

CREATE INDEX keeper_instances_voyage_idx
    ON keeper_instances (voyage_id);

-- Remove permanent unlock storage and its stable-key identity index.

DROP TABLE IF EXISTS player_keeper_unlocks;
DROP INDEX IF EXISTS keeper_definition_versions_id_keeper_key_idx;
