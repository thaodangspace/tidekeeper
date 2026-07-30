-- Tidekeepers foundation schema. All instants use timestamptz; PostgreSQL stores
-- these as absolute instants and clients must render public values in UTC (`Z`).

CREATE TABLE accounts (
    id uuid PRIMARY KEY,
    status text NOT NULL DEFAULT 'ACTIVE',
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT accounts_status_check
        CHECK (status IN ('ACTIVE', 'DISABLED')),
    CONSTRAINT accounts_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE TABLE players (
    id uuid PRIMARY KEY,
    account_id uuid NOT NULL,
    public_id text NOT NULL,
    onboarding_completed boolean NOT NULL DEFAULT false,
    locale text NOT NULL DEFAULT 'en',
    timezone text NOT NULL DEFAULT 'UTC',
    -- Added after voyages exists with an ownership-safe composite foreign key.
    current_voyage_id uuid,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT players_account_id_key UNIQUE (account_id),
    CONSTRAINT players_public_id_key UNIQUE (public_id),
    CONSTRAINT players_id_account_id_key UNIQUE (id, account_id),
    CONSTRAINT players_account_fk
        FOREIGN KEY (account_id) REFERENCES accounts (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT players_public_id_check
        CHECK (char_length(public_id) BETWEEN 5 AND 128
            AND public_id ~ '^plr_[A-Za-z0-9][A-Za-z0-9_-]*$'),
    CONSTRAINT players_locale_check
        CHECK (char_length(locale) BETWEEN 2 AND 35
            AND locale ~ '^[A-Za-z]{2,3}([_-][A-Za-z0-9]{2,8})*$'),
    CONSTRAINT players_timezone_check
        CHECK (char_length(timezone) BETWEEN 1 AND 255
            AND timezone ~ '^[A-Za-z0-9._+-]+(/[A-Za-z0-9._+-]+)*$'),
    CONSTRAINT players_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE TABLE sessions (
    id uuid PRIMARY KEY,
    account_id uuid NOT NULL,
    player_id uuid NOT NULL,
    -- SHA-256 digest of the opaque 32-byte session token. Raw tokens are never stored.
    token_digest bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    rotated_at timestamptz,
    replaced_by_session_id uuid,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT sessions_token_digest_key UNIQUE (token_digest),
    CONSTRAINT sessions_account_fk
        FOREIGN KEY (account_id) REFERENCES accounts (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sessions_player_account_fk
        FOREIGN KEY (player_id, account_id) REFERENCES players (id, account_id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sessions_replacement_fk
        FOREIGN KEY (replaced_by_session_id) REFERENCES sessions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT sessions_token_digest_length_check
        CHECK (octet_length(token_digest) = 32),
    CONSTRAINT sessions_expiry_check
        CHECK (expires_at > created_at),
    CONSTRAINT sessions_last_seen_check
        CHECK (last_seen_at >= created_at AND last_seen_at <= expires_at),
    CONSTRAINT sessions_rotation_pair_check
        CHECK ((rotated_at IS NULL) = (replaced_by_session_id IS NULL)),
    CONSTRAINT sessions_rotation_time_check
        CHECK (rotated_at IS NULL OR rotated_at >= created_at),
    CONSTRAINT sessions_revocation_time_check
        CHECK (revoked_at IS NULL OR revoked_at >= created_at),
    CONSTRAINT sessions_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE INDEX sessions_player_account_idx
    ON sessions (player_id, account_id);
CREATE INDEX sessions_expiry_cleanup_idx
    ON sessions (expires_at);
CREATE INDEX sessions_active_expiry_idx
    ON sessions (expires_at)
    WHERE revoked_at IS NULL AND replaced_by_session_id IS NULL;

CREATE TABLE voyages (
    id uuid PRIMARY KEY,
    public_id text NOT NULL,
    player_id uuid NOT NULL,
    status text NOT NULL,
    definition_version_key text NOT NULL,
    current_day_number smallint NOT NULL,
    fund_health integer NOT NULL,
    max_fund_health integer NOT NULL,
    capital integer NOT NULL,
    score numeric(20,4) NOT NULL DEFAULT 0.0000,
    row_version bigint NOT NULL DEFAULT 1,
    started_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT voyages_public_id_key UNIQUE (public_id),
    CONSTRAINT voyages_id_player_id_key UNIQUE (id, player_id),
    CONSTRAINT voyages_player_fk
        FOREIGN KEY (player_id) REFERENCES players (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyages_public_id_check
        CHECK (char_length(public_id) BETWEEN 5 AND 128
            AND public_id ~ '^voy_[A-Za-z0-9][A-Za-z0-9_-]*$'),
    CONSTRAINT voyages_status_check
        CHECK (status IN ('ACTIVE', 'COMPLETED', 'FAILED', 'ABANDONED')),
    CONSTRAINT voyages_definition_version_key_check
        CHECK (char_length(definition_version_key) BETWEEN 1 AND 128
            AND definition_version_key ~ '^[A-Za-z][A-Za-z0-9_.:-]*$'),
    CONSTRAINT voyages_day_number_check
        CHECK (current_day_number BETWEEN 1 AND 7),
    CONSTRAINT voyages_fund_health_check
        CHECK (fund_health >= 0 AND fund_health <= max_fund_health),
    CONSTRAINT voyages_max_fund_health_check
        CHECK (max_fund_health BETWEEN 1 AND 10000),
    CONSTRAINT voyages_capital_check
        CHECK (capital >= 0),
    CONSTRAINT voyages_score_check
        CHECK (score >= 0),
    CONSTRAINT voyages_row_version_check
        CHECK (row_version BETWEEN 1 AND 9007199254740991),
    CONSTRAINT voyages_started_time_check
        CHECK (started_at >= created_at),
    CONSTRAINT voyages_completion_time_check
        CHECK (completed_at IS NULL OR completed_at >= started_at),
    CONSTRAINT voyages_status_completion_check
        CHECK ((status = 'ACTIVE' AND completed_at IS NULL)
            OR (status IN ('COMPLETED', 'FAILED', 'ABANDONED') AND completed_at IS NOT NULL)),
    CONSTRAINT voyages_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX voyages_one_active_per_player_idx
    ON voyages (player_id)
    WHERE status = 'ACTIVE';
CREATE INDEX voyages_player_status_idx
    ON voyages (player_id, status);

ALTER TABLE players
    ADD CONSTRAINT players_current_voyage_owner_fk
    FOREIGN KEY (current_voyage_id, id)
    REFERENCES voyages (id, player_id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

CREATE INDEX players_current_voyage_owner_idx
    ON players (current_voyage_id, id)
    WHERE current_voyage_id IS NOT NULL;

CREATE TABLE daily_tides (
    id uuid PRIMARY KEY,
    day_key date NOT NULL,
    sequence_number bigint NOT NULL,
    phase text NOT NULL,
    lock_at timestamptz NOT NULL,
    settle_after timestamptz NOT NULL,
    content_version bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT daily_tides_day_key_key UNIQUE (day_key),
    CONSTRAINT daily_tides_sequence_number_key UNIQUE (sequence_number),
    CONSTRAINT daily_tides_sequence_number_check
        CHECK (sequence_number > 0),
    CONSTRAINT daily_tides_phase_check
        CHECK (char_length(phase) BETWEEN 1 AND 64
            AND phase ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT daily_tides_time_order_check
        CHECK (lock_at < settle_after),
    CONSTRAINT daily_tides_content_version_check
        CHECK (content_version > 0),
    CONSTRAINT daily_tides_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE INDEX daily_tides_timing_idx
    ON daily_tides (lock_at, settle_after);

CREATE TABLE player_daily_states (
    id uuid PRIMARY KEY,
    player_id uuid NOT NULL,
    voyage_id uuid NOT NULL,
    daily_tide_id uuid NOT NULL,
    day_number smallint NOT NULL,
    -- Intentionally not an enum/checklist: future uppercase phase tokens remain readable.
    phase text NOT NULL,
    version bigint NOT NULL,
    selected_strategy_id text,
    pending_reward_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT player_daily_states_voyage_day_key
        UNIQUE (voyage_id, day_number),
    CONSTRAINT player_daily_states_voyage_tide_key
        UNIQUE (voyage_id, daily_tide_id),
    CONSTRAINT player_daily_states_voyage_owner_fk
        FOREIGN KEY (voyage_id, player_id) REFERENCES voyages (id, player_id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT player_daily_states_tide_fk
        FOREIGN KEY (daily_tide_id) REFERENCES daily_tides (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT player_daily_states_day_number_check
        CHECK (day_number BETWEEN 1 AND 7),
    CONSTRAINT player_daily_states_phase_check
        CHECK (char_length(phase) BETWEEN 1 AND 64
            AND phase ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT player_daily_states_version_check
        CHECK (version BETWEEN 1 AND 9007199254740991),
    CONSTRAINT player_daily_states_selected_strategy_id_check
        CHECK (selected_strategy_id IS NULL
            OR (char_length(selected_strategy_id) BETWEEN 1 AND 128
                AND selected_strategy_id ~ '^[A-Za-z][A-Za-z0-9_-]*$')),
    CONSTRAINT player_daily_states_pending_reward_count_check
        CHECK (pending_reward_count >= 0),
    CONSTRAINT player_daily_states_timestamp_order_check
        CHECK (updated_at >= created_at)
);

CREATE INDEX player_daily_states_current_lookup_idx
    ON player_daily_states (player_id, voyage_id, day_number);
CREATE INDEX player_daily_states_tide_idx
    ON player_daily_states (daily_tide_id);

CREATE TABLE player_daily_context_views (
    player_daily_state_id uuid PRIMARY KEY,
    schema_version smallint NOT NULL DEFAULT 1,
    projection jsonb NOT NULL,
    validated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT player_daily_context_views_state_fk
        FOREIGN KEY (player_daily_state_id) REFERENCES player_daily_states (id)
        ON UPDATE RESTRICT ON DELETE CASCADE,
    CONSTRAINT player_daily_context_views_schema_version_check
        CHECK (schema_version > 0),
    CONSTRAINT player_daily_context_views_projection_object_check
        CHECK (jsonb_typeof(projection) = 'object'),
    CONSTRAINT player_daily_context_views_validation_time_check
        CHECK (validated_at >= created_at AND validated_at <= updated_at),
    CONSTRAINT player_daily_context_views_timestamp_order_check
        CHECK (updated_at >= created_at)
);
