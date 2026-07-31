-- Normalized relational projection of each Keeper definition's validated
-- upgrade tree. The checksummed `keeper_definition_versions.upgrade_tree` JSON
-- remains the canonical content source; these rows are an integrity/read
-- projection derived from it. Existing published definitions are backfilled
-- from their immutable JSON before mutation guards are enabled.

CREATE TABLE keeper_definition_upgrade_nodes (
    keeper_definition_version_id uuid NOT NULL,
    node_key text NOT NULL,
    depth smallint NOT NULL,
    is_root boolean NOT NULL,
    created_at timestamptz NOT NULL DEFAULT transaction_timestamp(),

    CONSTRAINT keeper_definition_upgrade_nodes_pkey
        PRIMARY KEY (keeper_definition_version_id, node_key),
    CONSTRAINT keeper_definition_upgrade_nodes_definition_fk
        FOREIGN KEY (keeper_definition_version_id) REFERENCES keeper_definition_versions (id)
        ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT keeper_definition_upgrade_nodes_node_key_check
        CHECK (node_key ~ '^[a-z][a-z0-9_]*$'),
    CONSTRAINT keeper_definition_upgrade_nodes_depth_check
        CHECK (depth >= 0),
    CONSTRAINT keeper_definition_upgrade_nodes_root_check
        CHECK (is_root = (depth = 0))
);

-- Exactly one root per definition.
CREATE UNIQUE INDEX keeper_definition_upgrade_nodes_one_root_idx
    ON keeper_definition_upgrade_nodes (keeper_definition_version_id)
    WHERE is_root;

-- ── Defensive validation of the canonical content before backfill ───────────
-- Every definition must expose a structurally complete upgrade tree; malformed
-- content aborts the migration rather than being silently skipped.

DO $$
DECLARE
    bad_count bigint;
BEGIN
    SELECT count(*) INTO bad_count
    FROM keeper_definition_versions k
    WHERE jsonb_typeof(k.upgrade_tree) IS DISTINCT FROM 'object'
       OR jsonb_typeof(k.upgrade_tree->'nodes') IS DISTINCT FROM 'array'
       OR jsonb_array_length(k.upgrade_tree->'nodes') < 1;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'upgrade node backfill aborted: % definitions have a missing or malformed upgrade tree', bad_count;
    END IF;

    SELECT count(*) INTO bad_count
    FROM keeper_definition_versions k
    CROSS JOIN LATERAL jsonb_array_elements(k.upgrade_tree->'nodes') AS json_node
    WHERE json_node->>'key' = k.upgrade_tree->>'rootNodeKey';

    IF bad_count <> (SELECT count(*) FROM keeper_definition_versions) THEN
        RAISE EXCEPTION 'upgrade node backfill aborted: % definitions do not declare an existing root node',
            (SELECT count(*) FROM keeper_definition_versions) - bad_count;
    END IF;
END;
$$;

-- ── Backfill from canonical JSON with deterministic depths ──────────────────
-- Each validated node has exactly one parent, so the recursive traversal
-- resolves exactly one depth per node and terminates.

WITH RECURSIVE traversed AS (
    SELECT
        k.id AS keeper_definition_version_id,
        k.upgrade_tree->>'rootNodeKey' AS node_key,
        0::integer AS depth
    FROM keeper_definition_versions k

    UNION

    SELECT
        e.keeper_definition_version_id,
        e.child_key,
        t.depth + 1
    FROM traversed t
    JOIN LATERAL (
        SELECT
            k2.id AS keeper_definition_version_id,
            child.value AS child_key
        FROM keeper_definition_versions k2
        CROSS JOIN LATERAL jsonb_array_elements(k2.upgrade_tree->'nodes') AS node
        CROSS JOIN LATERAL jsonb_array_elements_text(node->'next') AS child
        WHERE k2.id = t.keeper_definition_version_id
          AND node->>'key' = t.node_key
    ) e ON e.keeper_definition_version_id = t.keeper_definition_version_id
)
INSERT INTO keeper_definition_upgrade_nodes (keeper_definition_version_id, node_key, depth, is_root)
SELECT
    keeper_definition_version_id,
    node_key,
    depth::smallint,
    (depth = 0)
FROM traversed;

-- ── Assert backfill completeness before completing the migration ────────────

DO $$
DECLARE
    bad_count bigint;
BEGIN
    SELECT count(*) INTO bad_count
    FROM keeper_definition_versions k
    CROSS JOIN LATERAL jsonb_array_elements(k.upgrade_tree->'nodes') AS json_node
    LEFT JOIN keeper_definition_upgrade_nodes n
        ON n.keeper_definition_version_id = k.id
        AND n.node_key = json_node->>'key'
    WHERE n.keeper_definition_version_id IS NULL;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'upgrade node backfill aborted: % JSON nodes have no normalized row', bad_count;
    END IF;

    SELECT count(*) INTO bad_count
    FROM (
        SELECT keeper_definition_version_id
        FROM keeper_definition_upgrade_nodes
        WHERE is_root
        GROUP BY keeper_definition_version_id
        HAVING count(*) <> 1
    ) bad_root;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'upgrade node backfill aborted: % definitions do not have exactly one root', bad_count;
    END IF;

    SELECT count(*) INTO bad_count
    FROM (
        SELECT k.id, jsonb_array_length(k.upgrade_tree->'nodes') AS expected
        FROM keeper_definition_versions k
    ) e
    JOIN (
        SELECT keeper_definition_version_id, count(*) AS actual
        FROM keeper_definition_upgrade_nodes
        GROUP BY keeper_definition_version_id
    ) a ON a.keeper_definition_version_id = e.id
    WHERE a.actual <> e.expected;

    IF bad_count > 0 THEN
        RAISE EXCEPTION 'upgrade node backfill aborted: % definitions have mismatched node totals', bad_count;
    END IF;
END;
$$;

-- ── Draft/published mutation guards ─────────────────────────────────────────
-- Node rows belong to a definition's content release; only a DRAFT release may
-- be assembled, matching the existing catalog draft-only guard pattern.

CREATE FUNCTION keeper_catalog_upgrade_node_content_version(definition_id uuid)
RETURNS bigint
LANGUAGE plpgsql
AS $$
DECLARE
    resolved_version bigint;
BEGIN
    SELECT content_version
    INTO resolved_version
    FROM keeper_definition_versions
    WHERE id = definition_id;

    RETURN resolved_version;
END;
$$;

CREATE FUNCTION keeper_catalog_guard_upgrade_node_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    existing_version bigint;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        existing_version := keeper_catalog_upgrade_node_content_version(OLD.keeper_definition_version_id);
        IF existing_version IS NOT NULL THEN
            PERFORM keeper_catalog_require_draft_content_release(existing_version);
        END IF;
    END IF;

    IF TG_OP <> 'DELETE' THEN
        existing_version := keeper_catalog_upgrade_node_content_version(NEW.keeper_definition_version_id);
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

CREATE TRIGGER keeper_definition_upgrade_nodes_draft_only_trigger
    BEFORE INSERT OR UPDATE OR DELETE ON keeper_definition_upgrade_nodes
    FOR EACH ROW EXECUTE FUNCTION keeper_catalog_guard_upgrade_node_mutation();
