-- Keeper instances are permanent player-owned inventory. Each instance is bound
-- to the exact versioned definition it was acquired from, preserving its content
-- identity across future catalog releases.
CREATE TABLE keeper_instances (
    id uuid PRIMARY KEY,
    public_id text NOT NULL,
    player_id uuid NOT NULL,
    keeper_definition_version_id uuid NOT NULL,
    level smallint NOT NULL DEFAULT 1,
    acquired_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT keeper_instances_public_id_key UNIQUE (public_id),
    CONSTRAINT keeper_instances_player_fk
        FOREIGN KEY (player_id) REFERENCES players (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_instances_definition_fk
        FOREIGN KEY (keeper_definition_version_id) REFERENCES keeper_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_instances_public_id_check
        CHECK (char_length(public_id) BETWEEN 5 AND 128
            AND public_id ~ '^kpr_[A-Za-z0-9][A-Za-z0-9_-]*$'),
    CONSTRAINT keeper_instances_level_check
        CHECK (level BETWEEN 1 AND 100)
);

CREATE INDEX keeper_instances_player_acquired_idx
    ON keeper_instances (player_id, acquired_at, id);
