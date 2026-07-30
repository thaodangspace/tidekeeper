-- Immutable, versioned Keeper catalog content. A content release is assembled while
-- DRAFT, then becomes append-only when PUBLISHED.

CREATE TABLE content_releases (
    version bigint PRIMARY KEY,
    status text NOT NULL,
    checksum bytea NOT NULL,
    published_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT content_releases_version_check
        CHECK (version > 0),
    CONSTRAINT content_releases_status_check
        CHECK (status IN ('DRAFT', 'PUBLISHED')),
    CONSTRAINT content_releases_checksum_length_check
        CHECK (octet_length(checksum) = 32),
    CONSTRAINT content_releases_publication_check
        CHECK ((status = 'DRAFT' AND published_at IS NULL)
            OR (status = 'PUBLISHED' AND published_at IS NOT NULL AND published_at >= created_at))
);

CREATE TABLE market_assets (
    id uuid PRIMARY KEY,
    asset_key text NOT NULL,
    symbol text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT market_assets_asset_key_key UNIQUE (asset_key),
    CONSTRAINT market_assets_symbol_key UNIQUE (symbol),
    CONSTRAINT market_assets_asset_key_check
        CHECK (asset_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT market_assets_symbol_check
        CHECK (symbol ~ '^[A-Za-z][A-Za-z0-9]*$')
);

CREATE TABLE basket_mapping_versions (
    id uuid PRIMARY KEY,
    mapping_key text NOT NULL,
    content_version bigint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT basket_mapping_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_mapping_versions_mapping_version_key
        UNIQUE (mapping_key, content_version),
    CONSTRAINT basket_mapping_versions_id_content_version_key
        UNIQUE (id, content_version),
    CONSTRAINT basket_mapping_versions_mapping_key_check
        CHECK (mapping_key ~ '^[a-z][a-z0-9_]*$')
);

CREATE INDEX basket_mapping_versions_content_version_idx
    ON basket_mapping_versions (content_version, mapping_key);

CREATE TABLE basket_mapping_components (
    basket_mapping_version_id uuid NOT NULL,
    market_asset_id uuid NOT NULL,
    weight numeric(9,8) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT basket_mapping_components_key
        PRIMARY KEY (basket_mapping_version_id, market_asset_id),
    CONSTRAINT basket_mapping_components_mapping_fk
        FOREIGN KEY (basket_mapping_version_id) REFERENCES basket_mapping_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_mapping_components_market_asset_fk
        FOREIGN KEY (market_asset_id) REFERENCES market_assets (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT basket_mapping_components_weight_check
        CHECK (weight > 0 AND weight <= 1.00000000)
);

CREATE INDEX basket_mapping_components_market_asset_idx
    ON basket_mapping_components (market_asset_id);

CREATE TABLE keeper_definition_versions (
    id uuid PRIMARY KEY,
    keeper_key text NOT NULL,
    content_version bigint NOT NULL,
    name text NOT NULL,
    current_key text NOT NULL,
    current_name text NOT NULL,
    sector_key text NOT NULL,
    role_key text NOT NULL,
    rarity_key text NOT NULL,
    base_risk_key text NOT NULL,
    expected_turbulence_bps integer,
    passive_rule_key text NOT NULL,
    passive_rule_config jsonb NOT NULL,
    upgrade_tree jsonb NOT NULL,
    basket_mapping_version_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT keeper_definition_versions_keeper_version_key
        UNIQUE (keeper_key, content_version),
    CONSTRAINT keeper_definition_versions_content_fk
        FOREIGN KEY (content_version) REFERENCES content_releases (version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_definition_versions_mapping_same_release_fk
        FOREIGN KEY (basket_mapping_version_id, content_version)
        REFERENCES basket_mapping_versions (id, content_version)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_definition_versions_keeper_key_check
        CHECK (keeper_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT keeper_definition_versions_name_check
        CHECK (char_length(name) BETWEEN 1 AND 128),
    CONSTRAINT keeper_definition_versions_current_key_check
        CHECK (current_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT keeper_definition_versions_current_name_check
        CHECK (char_length(current_name) BETWEEN 1 AND 128),
    CONSTRAINT keeper_definition_versions_sector_key_check
        CHECK (sector_key IN ('CREST', 'HARBOR', 'FORGE', 'CURRENT', 'ECHO', 'VEIL', 'BLOOM', 'EMBER')),
    CONSTRAINT keeper_definition_versions_role_key_check
        CHECK (role_key IN ('VANGUARD', 'WARDEN', 'COMPOUNDER', 'SUPPORT', 'ORACLE', 'CONTRARIAN', 'TRICKSTER', 'GROWTH')),
    CONSTRAINT keeper_definition_versions_rarity_key_check
        CHECK (rarity_key IN ('COMMON', 'RARE', 'EPIC')),
    CONSTRAINT keeper_definition_versions_base_risk_key_check
        CHECK (base_risk_key IN ('VERY_LOW', 'LOW_TO_MEDIUM', 'MEDIUM', 'MEDIUM_TO_HIGH', 'HIGH', 'VERY_HIGH')),
    CONSTRAINT keeper_definition_versions_expected_turbulence_check
        CHECK (expected_turbulence_bps IS NULL
            OR expected_turbulence_bps BETWEEN 1 AND 1000000),
    CONSTRAINT keeper_definition_versions_passive_rule_key_check
        CHECK (passive_rule_key ~ '^[A-Z][A-Z0-9_]*$'),
    CONSTRAINT keeper_definition_versions_passive_rule_config_check
        CHECK (jsonb_typeof(passive_rule_config) = 'object'),
    CONSTRAINT keeper_definition_versions_upgrade_tree_check
        CHECK (jsonb_typeof(upgrade_tree) = 'object')
);

CREATE INDEX keeper_definition_versions_content_version_idx
    ON keeper_definition_versions (content_version, keeper_key);
CREATE INDEX keeper_definition_versions_mapping_idx
    ON keeper_definition_versions (basket_mapping_version_id);

CREATE FUNCTION keeper_catalog_require_draft_content_release(required_version bigint)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    release_status text;
BEGIN
    SELECT status
    INTO release_status
    FROM content_releases
    WHERE version = required_version;

    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '23503',
            MESSAGE = format('content release %s does not exist', required_version);
    END IF;

    IF release_status <> 'DRAFT' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('content release %s is not mutable', required_version);
    END IF;
END;
$$;

CREATE FUNCTION keeper_catalog_guard_release_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        PERFORM keeper_catalog_require_draft_content_release(NEW.content_version);
        RETURN NEW;
    END IF;

    PERFORM keeper_catalog_require_draft_content_release(OLD.content_version);
    IF TG_OP = 'UPDATE' THEN
        PERFORM keeper_catalog_require_draft_content_release(NEW.content_version);
        RETURN NEW;
    END IF;
    RETURN OLD;
END;
$$;

CREATE TRIGGER basket_mapping_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON basket_mapping_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE TRIGGER keeper_definition_versions_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON keeper_definition_versions
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_release_mutation();

CREATE FUNCTION keeper_catalog_component_content_version(mapping_id uuid)
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_version bigint;
BEGIN
    SELECT content_version
    INTO resolved_version
    FROM basket_mapping_versions
    WHERE id = mapping_id;

    RETURN resolved_version;
END;
$$;

CREATE FUNCTION keeper_catalog_guard_component_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    existing_version bigint;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        existing_version := keeper_catalog_component_content_version(OLD.basket_mapping_version_id);
        IF existing_version IS NOT NULL THEN
            PERFORM keeper_catalog_require_draft_content_release(existing_version);
        END IF;
    END IF;

    IF TG_OP <> 'DELETE' THEN
        existing_version := keeper_catalog_component_content_version(NEW.basket_mapping_version_id);
        IF existing_version IS NOT NULL THEN
            PERFORM keeper_catalog_require_draft_content_release(existing_version);
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER basket_mapping_components_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON basket_mapping_components
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_component_mutation();

CREATE FUNCTION keeper_catalog_assert_basket_weight_total()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    affected_mapping_id uuid;
    component_weight_total numeric(9,8);
    affected_mapping_ids uuid[];
BEGIN
    affected_mapping_ids := CASE
        WHEN TG_OP = 'INSERT' THEN ARRAY[NEW.basket_mapping_version_id]
        WHEN TG_OP = 'DELETE' THEN ARRAY[OLD.basket_mapping_version_id]
        ELSE ARRAY[OLD.basket_mapping_version_id, NEW.basket_mapping_version_id]
    END;

    FOREACH affected_mapping_id IN ARRAY affected_mapping_ids LOOP
        IF EXISTS (
            SELECT 1
            FROM basket_mapping_versions
            WHERE id = affected_mapping_id
        ) THEN
            SELECT COALESCE(sum(weight), 0)::numeric(9,8)
            INTO component_weight_total
            FROM basket_mapping_components
            WHERE basket_mapping_version_id = affected_mapping_id;

            IF component_weight_total <> 1.00000000::numeric(9,8) THEN
                RAISE EXCEPTION USING
                    ERRCODE = '23514',
                    MESSAGE = format(
                        'basket mapping %s component weights must total 1.00000000; got %s',
                        affected_mapping_id,
                        component_weight_total
                    );
            END IF;
        END IF;
    END LOOP;

    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER basket_mapping_components_weight_total_trigger
    AFTER INSERT OR UPDATE OR DELETE ON basket_mapping_components
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_assert_basket_weight_total();

CREATE FUNCTION keeper_catalog_guard_asset_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION USING
        ERRCODE = '23514',
        MESSAGE = 'market assets are immutable; add a new canonical asset instead';
END;
$$;

CREATE TRIGGER market_assets_immutable_trigger
    BEFORE UPDATE OR DELETE ON market_assets
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_asset_mutation();

CREATE FUNCTION keeper_catalog_guard_published_release_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' AND OLD.status = 'PUBLISHED' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23514',
            MESSAGE = format('published content release %s is immutable', OLD.version);
    END IF;

    IF TG_OP = 'UPDATE' THEN
        IF OLD.status = 'PUBLISHED' THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = format('published content release %s is immutable', OLD.version);
        END IF;
        IF OLD.status = 'DRAFT' AND NEW.status = 'DRAFT' AND NEW.published_at IS NOT NULL THEN
            RAISE EXCEPTION USING
                ERRCODE = '23514',
                MESSAGE = format('draft content release %s cannot have a publication timestamp', OLD.version);
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER content_releases_published_immutable_trigger
    BEFORE UPDATE OR DELETE ON content_releases
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_published_release_mutation();
