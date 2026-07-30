-- Voyage lifecycle: definition publication, idempotent creation, terminal protection,
-- and lifecycle history.

-- ── Immutable published voyage definitions ──────────────────────────────────

CREATE TABLE voyage_definition_versions (
    id uuid PRIMARY KEY,
    definition_key text NOT NULL,
    version bigint NOT NULL,
    status text NOT NULL,
    duration_days smallint NOT NULL,
    starting_hull integer NOT NULL,
    starting_supplies integer NOT NULL,
    fleet_slot_count smallint NOT NULL,
    initial_phase text NOT NULL,
    launch_presentation jsonb NOT NULL,
    checksum bytea NOT NULL,
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT voyage_definition_versions_key_version_key
        UNIQUE (definition_key, version),
    CONSTRAINT voyage_definition_versions_status_check
        CHECK (status IN ('DRAFT', 'PUBLISHED')),
    CONSTRAINT voyage_definition_versions_duration_check
        CHECK (duration_days BETWEEN 1 AND 365),
    CONSTRAINT voyage_definition_versions_hull_check
        CHECK (starting_hull BETWEEN 1 AND 10000),
    CONSTRAINT voyage_definition_versions_supplies_check
        CHECK (starting_supplies >= 0),
    CONSTRAINT voyage_definition_versions_slots_check
        CHECK (fleet_slot_count BETWEEN 1 AND 20),
    CONSTRAINT voyage_definition_versions_phase_check
        CHECK (char_length(initial_phase) BETWEEN 1 AND 64
            AND initial_phase ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT voyage_definition_versions_presentation_check
        CHECK (jsonb_typeof(launch_presentation) = 'object'),
    CONSTRAINT voyage_definition_versions_checksum_length_check
        CHECK (octet_length(checksum) = 32),
    CONSTRAINT voyage_definition_versions_publication_check
        CHECK ((status = 'DRAFT' AND published_at IS NULL)
            OR (status = 'PUBLISHED' AND published_at IS NOT NULL AND published_at >= created_at))
);

CREATE INDEX voyage_definition_versions_published_idx
    ON voyage_definition_versions (definition_key, version DESC)
    WHERE status = 'PUBLISHED';

CREATE TABLE voyage_definition_starter_keepers (
    voyage_definition_version_id uuid NOT NULL,
    keeper_definition_version_id uuid NOT NULL,
    position smallint NOT NULL,

    CONSTRAINT voyage_definition_starter_keepers_pkey
        PRIMARY KEY (voyage_definition_version_id, position),
    CONSTRAINT voyage_definition_starter_keepers_definition_fk
        FOREIGN KEY (voyage_definition_version_id) REFERENCES voyage_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_definition_starter_keepers_keeper_fk
        FOREIGN KEY (keeper_definition_version_id) REFERENCES keeper_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_definition_starter_keepers_position_check
        CHECK (position >= 0)
);

CREATE TABLE voyage_definition_initial_shop_offers (
    voyage_definition_version_id uuid NOT NULL,
    keeper_definition_version_id uuid NOT NULL,
    position smallint NOT NULL,
    cost integer NOT NULL,

    CONSTRAINT voyage_definition_initial_shop_offers_pkey
        PRIMARY KEY (voyage_definition_version_id, position),
    CONSTRAINT voyage_definition_initial_shop_offers_definition_fk
        FOREIGN KEY (voyage_definition_version_id) REFERENCES voyage_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_definition_initial_shop_offers_keeper_fk
        FOREIGN KEY (keeper_definition_version_id) REFERENCES keeper_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_definition_initial_shop_offers_position_check
        CHECK (position >= 0),
    CONSTRAINT voyage_definition_initial_shop_offers_cost_check
        CHECK (cost >= 0)
);

-- ── Voyage definition draft/published guards ───────────────────────────────

CREATE FUNCTION voyage_definition_require_draft_status(required_id uuid)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    def_status text;
BEGIN
    SELECT status INTO def_status
    FROM voyage_definition_versions
    WHERE id = required_id;

    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '23503',
            MESSAGE = format('voyage definition version %s does not exist', required_id);
    END IF;

    IF def_status <> 'DRAFT' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('voyage definition version %s is not mutable', required_id);
    END IF;
END;
$$;

CREATE FUNCTION voyage_definition_guard_child_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        PERFORM voyage_definition_require_draft_status(OLD.voyage_definition_version_id);
        PERFORM voyage_definition_require_draft_status(NEW.voyage_definition_version_id);
        RETURN NEW;
    END IF;
    PERFORM voyage_definition_require_draft_status(
        CASE WHEN TG_OP = 'DELETE' THEN OLD.voyage_definition_version_id
             ELSE NEW.voyage_definition_version_id
        END
    );
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER voyage_definition_starter_keepers_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON voyage_definition_starter_keepers
    FOR EACH ROW EXECUTE FUNCTION voyage_definition_guard_child_mutation();

CREATE TRIGGER voyage_definition_initial_shop_offers_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON voyage_definition_initial_shop_offers
    FOR EACH ROW EXECUTE FUNCTION voyage_definition_guard_child_mutation();

CREATE FUNCTION voyage_definition_guard_published_immutability()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.status = 'PUBLISHED' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('published voyage definition %s is immutable', OLD.id);
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'PUBLISHED' THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = format('published voyage definition %s is immutable', OLD.id);
        END IF;
        IF OLD.status = 'DRAFT' AND NEW.status = 'DRAFT' AND NEW.published_at IS NOT NULL THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = format('draft voyage definition %s cannot have a publication timestamp', OLD.id);
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER voyage_definition_versions_published_immutable_trigger
    BEFORE UPDATE OR DELETE ON voyage_definition_versions
    FOR EACH ROW EXECUTE FUNCTION voyage_definition_guard_published_immutability();

-- ── Standard MVP definition ────────────────────────────────────────────────
-- Published with starter Hull=100, Supplies=10, 3 fleet slots, no starter
-- Keepers (keeper definitions are not yet seeded). The initial shop is empty;
-- the shop is populated by content publication as Keeper definitions are added.
--

INSERT INTO voyage_definition_versions (
    id, definition_key, version, status, duration_days,
    starting_hull, starting_supplies, fleet_slot_count, initial_phase,
    launch_presentation, checksum, published_at
) VALUES (
    '00000000-0000-0000-0000-000000000002',
    'standard',
    1,
    'PUBLISHED',
    7,
    100,
    10,
    3,
    'PREPARATION',
    '{
        "modifier":{"id":"mod_default","name":"Standard Conditions","description":"Default voyage conditions."},
        "objective":{"id":"obj_default","name":"Navigate","description":"Complete the daily voyage.","progressLabel":null,"rewardLabel":null},
        "signals":[],
        "lineup":{
            "lockedAt":null,"maxSlots":3,
            "slots":[{"index":0,"keeper":null},{"index":1,"keeper":null},{"index":2,"keeper":null}],
            "synergies":[],"warnings":[{"code":"EMPTY_SLOT","severity":"WARNING","message":"All slots are empty."}]
        },
        "inventory":[],
        "shop":{"offers":[],"refreshAt":null,"rerollCost":2,"rerollIndex":0},
        "strategies":[],
        "selectedStrategyId":null,
        "pendingRewardCount":0
    }'::jsonb,
    E'\\x7374616e64617264' || repeat(E'\\x00', 23),
    transaction_timestamp()
);

-- ── Legacy definition for existing development voyages ─────────────────────

INSERT INTO voyage_definition_versions (
    id, definition_key, version, status, duration_days,
    starting_hull, starting_supplies, fleet_slot_count, initial_phase,
    launch_presentation, checksum, published_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',
    'legacy',
    1,
    'PUBLISHED',
    7,
    100,
    10,
    3,
    'PREPARATION',
    '{
        "modifier":{"id":"mod_default","name":"Standard Conditions","description":"Default voyage conditions."},
        "objective":{"id":"obj_default","name":"Navigate","description":"Complete the daily voyage.","progressLabel":null,"rewardLabel":null},
        "signals":[],
        "lineup":{
            "lockedAt":null,"maxSlots":3,
            "slots":[{"index":0,"keeper":null},{"index":1,"keeper":null},{"index":2,"keeper":null}],
            "synergies":[],"warnings":[{"code":"EMPTY_SLOT","severity":"WARNING","message":"All slots are empty."}]
        },
        "inventory":[],
        "shop":{"offers":[],"refreshAt":null,"rerollCost":2,"rerollIndex":0},
        "strategies":[],
        "selectedStrategyId":null,
        "pendingRewardCount":0
    }'::jsonb,
    E'\\x6c6567616379' || repeat(E'\\x00', 28),
    transaction_timestamp()
);

-- ── Add voyage_definition_version_id to voyages ────────────────────────────

ALTER TABLE voyages
    ADD COLUMN voyage_definition_version_id uuid;

UPDATE voyages AS v
SET voyage_definition_version_id = '00000000-0000-0000-0000-000000000001'
WHERE v.voyage_definition_version_id IS NULL;

ALTER TABLE voyages
    ALTER COLUMN voyage_definition_version_id SET NOT NULL,
    ADD CONSTRAINT voyages_definition_version_fk
        FOREIGN KEY (voyage_definition_version_id) REFERENCES voyage_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT;

-- ── Reusable idempotency keys ──────────────────────────────────────────────

CREATE TABLE idempotency_keys (
    player_id uuid NOT NULL,
    scope text NOT NULL,
    key text NOT NULL,
    state text NOT NULL,
    request_hash bytea NOT NULL,
    result_voyage_id uuid,
    response_status smallint,
    response_json jsonb,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    completed_at timestamptz,

    CONSTRAINT idempotency_keys_pkey
        PRIMARY KEY (player_id, scope, key),
    CONSTRAINT idempotency_keys_player_fk
        FOREIGN KEY (player_id) REFERENCES players (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT idempotency_keys_result_voyage_fk
        FOREIGN KEY (result_voyage_id) REFERENCES voyages (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT idempotency_keys_state_check
        CHECK (state IN ('IN_PROGRESS', 'COMPLETED')),
    CONSTRAINT idempotency_keys_scope_check
        CHECK (char_length(scope) BETWEEN 1 AND 64
            AND scope ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT idempotency_keys_key_check
        CHECK (char_length(key) BETWEEN 1 AND 255
            AND key ~ '^[A-Za-z0-9][A-Za-z0-9._:-]*$'),
    CONSTRAINT idempotency_keys_hash_length_check
        CHECK (octet_length(request_hash) = 32),
    CONSTRAINT idempotency_keys_state_transition_check
        CHECK ((state = 'IN_PROGRESS' AND completed_at IS NULL)
            OR (state = 'COMPLETED' AND completed_at IS NOT NULL)),
    CONSTRAINT idempotency_keys_result_shape_check
        CHECK ((state = 'COMPLETED' AND result_voyage_id IS NOT NULL AND response_status IS NOT NULL)
            OR (state = 'IN_PROGRESS' AND result_voyage_id IS NULL AND response_status IS NULL))
);

CREATE INDEX idempotency_keys_stale_idx
    ON idempotency_keys (created_at)
    WHERE state = 'IN_PROGRESS';

-- ── Voyage ledger entries (append-only) ────────────────────────────────────

CREATE TABLE voyage_ledger_entries (
    id uuid PRIMARY KEY,
    voyage_id uuid NOT NULL,
    entry_type text NOT NULL,
    amount integer NOT NULL,
    balance_after integer NOT NULL,
    source_type text NOT NULL,
    source_id text NOT NULL,
    reason_key text NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT voyage_ledger_entries_voyage_fk
        FOREIGN KEY (voyage_id) REFERENCES voyages (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_ledger_entries_entry_type_check
        CHECK (char_length(entry_type) BETWEEN 1 AND 64
            AND entry_type ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT voyage_ledger_entries_amount_check
        CHECK (amount > 0),
    CONSTRAINT voyage_ledger_entries_balance_after_check
        CHECK (balance_after >= 0),
    CONSTRAINT voyage_ledger_entries_source_type_check
        CHECK (char_length(source_type) BETWEEN 1 AND 64
            AND source_type ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT voyage_ledger_entries_source_id_check
        CHECK (char_length(source_id) BETWEEN 1 AND 128),
    CONSTRAINT voyage_ledger_entries_reason_key_check
        CHECK (char_length(reason_key) BETWEEN 1 AND 128
            AND reason_key ~ '^[a-z][a-z0-9_.]*$')
);

CREATE INDEX voyage_ledger_entries_voyage_idx
    ON voyage_ledger_entries (voyage_id, occurred_at, id);

-- ── Voyage lifecycle events (append-only) ──────────────────────────────────

CREATE TABLE voyage_lifecycle_events (
    id uuid PRIMARY KEY,
    public_id text NOT NULL,
    voyage_id uuid NOT NULL,
    event_type text NOT NULL,
    resulting_status text NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT transaction_timestamp(),
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT voyage_lifecycle_events_public_id_key UNIQUE (public_id),
    CONSTRAINT voyage_lifecycle_events_voyage_fk
        FOREIGN KEY (voyage_id) REFERENCES voyages (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT voyage_lifecycle_events_public_id_check
        CHECK (char_length(public_id) BETWEEN 5 AND 128
            AND public_id ~ '^evt_[A-Za-z0-9][A-Za-z0-9_-]*$'),
    CONSTRAINT voyage_lifecycle_events_event_type_check
        CHECK (char_length(event_type) BETWEEN 1 AND 64
            AND event_type ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT voyage_lifecycle_events_status_check
        CHECK (resulting_status IN ('ACTIVE', 'COMPLETED', 'FAILED', 'ABANDONED'))
);

CREATE INDEX voyage_lifecycle_events_voyage_idx
    ON voyage_lifecycle_events (voyage_id, occurred_at, id);

-- ── Terminal voyage immutability ──────────────────────────────────────────

CREATE FUNCTION voyage_guard_terminal_immutability()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        RETURN NEW;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status IN ('COMPLETED', 'FAILED', 'ABANDONED') THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'terminal voyage rows are immutable';
        END IF;
        RETURN NEW;
    END IF;

    IF TG_OP = 'DELETE' THEN
        IF OLD.status IN ('COMPLETED', 'FAILED', 'ABANDONED') THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = 'terminal voyage rows are immutable';
        END IF;
        RETURN OLD;
    END IF;

    RETURN NULL;
END;
$$;

CREATE TRIGGER voyages_terminal_immutability_trigger
    BEFORE UPDATE OR DELETE ON voyages
    FOR EACH ROW EXECUTE FUNCTION voyage_guard_terminal_immutability();

-- ── Remove deprecated text key ─────────────────────────────────────────────

ALTER TABLE voyages
    DROP CONSTRAINT IF EXISTS voyages_definition_version_key_check,
    DROP COLUMN IF EXISTS definition_version_key;
