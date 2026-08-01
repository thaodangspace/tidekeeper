-- name: GetContentRelease :one
SELECT version, status, checksum, checksum_schema_version, published_at, created_at
FROM content_releases
WHERE version = sqlc.arg(version);

-- name: CreateDraftContentRelease :exec
INSERT INTO content_releases (version, status, checksum, checksum_schema_version)
VALUES (sqlc.arg(version), 'DRAFT', sqlc.arg(checksum), sqlc.arg(checksum_schema_version));

-- name: PublishContentRelease :exec
UPDATE content_releases
SET status = 'PUBLISHED', published_at = transaction_timestamp()
WHERE version = sqlc.arg(version)
  AND status = 'DRAFT';

-- name: GetMarketAssetByKey :one
SELECT id, asset_key, symbol, created_at
FROM market_assets
WHERE asset_key = sqlc.arg(asset_key);

-- name: InsertMarketAsset :exec
INSERT INTO market_assets (id, asset_key, symbol)
VALUES (sqlc.arg(id), sqlc.arg(asset_key), sqlc.arg(symbol));

-- name: InsertSectorDefinitionVersion :exec
INSERT INTO sector_definition_versions (
    id, sector_key, content_version, benchmark_method, minimum_eligible_baskets,
    relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units
) VALUES (
    sqlc.arg(id), sqlc.arg(sector_key), sqlc.arg(content_version), sqlc.arg(benchmark_method), sqlc.arg(minimum_eligible_baskets),
    sqlc.arg(relative_scale_units), sqlc.arg(relative_blend_weight_units), sqlc.arg(rank_blend_weight_units), sqlc.arg(score_cap_units)
);

-- name: InsertExpectedTurbulencePolicy :exec
INSERT INTO expected_turbulence_policies (
    id, policy_key, content_version, policy_type, static_value_units, floor_units, rounding_mode
) VALUES (
    sqlc.arg(id), sqlc.arg(policy_key), sqlc.arg(content_version), sqlc.arg(policy_type), sqlc.arg(static_value_units),
    sqlc.arg(floor_units), sqlc.arg(rounding_mode)
);

-- name: InsertBasketMappingVersion :exec
INSERT INTO basket_mapping_versions (
    id, mapping_key, content_version, sector_definition_version_id,
    minimum_covered_weight_units, expected_turbulence_policy_id,
    normalization_cap_units, benchmark_eligible
) VALUES (
    sqlc.arg(id), sqlc.arg(mapping_key), sqlc.arg(content_version), sqlc.narg(sector_definition_version_id),
    sqlc.narg(minimum_covered_weight_units), sqlc.narg(expected_turbulence_policy_id),
    sqlc.narg(normalization_cap_units), sqlc.narg(benchmark_eligible)
);

-- name: InsertBasketMappingComponent :exec
INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight)
VALUES (sqlc.arg(basket_mapping_version_id), sqlc.arg(market_asset_id), sqlc.arg(weight));

-- name: InsertKeeperDefinitionVersion :exec
INSERT INTO keeper_definition_versions (
    id,
    keeper_key,
    content_version,
    name,
    current_key,
    current_name,
    sector_key,
    role_key,
    rarity_key,
    base_risk_key,
    expected_turbulence_bps,
    passive_rule_key,
    passive_rule_config,
    upgrade_tree,
    basket_mapping_version_id
) VALUES (
    sqlc.arg(id),
    sqlc.arg(keeper_key),
    sqlc.arg(content_version),
    sqlc.arg(name),
    sqlc.arg(current_key),
    sqlc.arg(current_name),
    sqlc.arg(sector_key),
    sqlc.arg(role_key),
    sqlc.arg(rarity_key),
    sqlc.arg(base_risk_key),
    sqlc.narg(expected_turbulence_bps),
    sqlc.arg(passive_rule_key),
    sqlc.arg(passive_rule_config),
    sqlc.arg(upgrade_tree),
    sqlc.arg(basket_mapping_version_id)
);

-- name: InsertKeeperDefinitionUpgradeNode :exec
INSERT INTO keeper_definition_upgrade_nodes (keeper_definition_version_id, node_key, depth, is_root)
VALUES (sqlc.arg(keeper_definition_version_id), sqlc.arg(node_key), sqlc.arg(depth), sqlc.arg(is_root));

-- name: CountKeeperUpgradeNodesForRelease :one
SELECT count(*)::bigint AS node_count
FROM keeper_definition_upgrade_nodes AS n
JOIN keeper_definition_versions AS k ON k.id = n.keeper_definition_version_id
WHERE k.content_version = sqlc.arg(release_version);

-- name: CountCatalogReleaseRows :one
SELECT
    (SELECT count(*) FROM keeper_definition_versions AS k WHERE k.content_version = sqlc.arg(release_version))::bigint AS definition_count,
    (SELECT count(*) FROM basket_mapping_versions AS b WHERE b.content_version = sqlc.arg(release_version))::bigint AS mapping_count,
    (SELECT count(*)
     FROM basket_mapping_components AS c
     JOIN basket_mapping_versions AS m ON m.id = c.basket_mapping_version_id
     WHERE m.content_version = sqlc.arg(release_version))::bigint AS component_count,
    (SELECT count(*)
     FROM keeper_definition_upgrade_nodes AS n
     JOIN keeper_definition_versions AS k ON k.id = n.keeper_definition_version_id
     WHERE k.content_version = sqlc.arg(release_version))::bigint AS node_count;

-- name: ListMarketAssets :many
SELECT id, asset_key, symbol
FROM market_assets
ORDER BY asset_key;

-- name: ListSectorDefinitionsForRelease :many
SELECT id, sector_key, benchmark_method, minimum_eligible_baskets,
       relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units
FROM sector_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY sector_key;

-- name: ListTurbulencePoliciesForRelease :many
SELECT id, policy_key, policy_type, static_value_units, floor_units, rounding_mode
FROM expected_turbulence_policies
WHERE content_version = sqlc.arg(content_version)
ORDER BY policy_key;

-- name: ListBasketMappingsForRelease :many
SELECT m.id, m.mapping_key,
       COALESCE(s.sector_key, '') AS sector_key,
       m.minimum_covered_weight_units,
       COALESCE(t.policy_key, '') AS turbulence_policy_key,
       m.normalization_cap_units,
       m.benchmark_eligible
FROM basket_mapping_versions AS m
LEFT JOIN sector_definition_versions AS s ON s.id = m.sector_definition_version_id
LEFT JOIN expected_turbulence_policies AS t ON t.id = m.expected_turbulence_policy_id
WHERE m.content_version = sqlc.arg(content_version)
ORDER BY m.mapping_key;

-- name: ListBasketComponentsForRelease :many
SELECT c.basket_mapping_version_id, m.mapping_key, a.asset_key, c.weight
FROM basket_mapping_components AS c
JOIN basket_mapping_versions AS m ON m.id = c.basket_mapping_version_id
JOIN market_assets AS a ON a.id = c.market_asset_id
WHERE m.content_version = sqlc.arg(content_version)
ORDER BY m.mapping_key, a.asset_key;

-- name: ListKeeperDefinitionsForRelease :many
SELECT k.id, k.keeper_key, k.name, k.current_key, k.current_name,
       k.sector_key, k.role_key, k.rarity_key, k.base_risk_key,
       k.expected_turbulence_bps, k.passive_rule_key, k.passive_rule_config, k.upgrade_tree,
       m.mapping_key AS basket_mapping_key
FROM keeper_definition_versions AS k
JOIN basket_mapping_versions AS m ON m.id = k.basket_mapping_version_id
WHERE k.content_version = sqlc.arg(content_version)
ORDER BY k.keeper_key;
