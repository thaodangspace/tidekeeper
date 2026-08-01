-- name: InsertStrategyDefinitionVersion :exec
INSERT INTO strategy_definition_versions (
    id, strategy_key, content_version, name, description, upside, downside, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(strategy_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.arg(upside), sqlc.arg(downside),
    sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: InsertRelicDefinitionVersion :exec
INSERT INTO relic_definition_versions (
    id, relic_key, content_version, name, description, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(relic_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: InsertSynergyDefinitionVersion :exec
INSERT INTO synergy_definition_versions (
    id, synergy_key, content_version, name, description, required_count, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(synergy_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.arg(required_count),
    sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: InsertDailyModifierDefinitionVersion :exec
INSERT INTO daily_modifier_definition_versions (
    id, modifier_key, content_version, name, description, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(modifier_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: InsertDailyObjectiveDefinitionVersion :exec
INSERT INTO daily_objective_definition_versions (
    id, objective_key, content_version, name, description, progress_label, reward_label, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(objective_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.narg(progress_label), sqlc.narg(reward_label),
    sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: InsertGameRuleSetVersion :exec
INSERT INTO game_rule_set_versions (
    id, rule_set_key, content_version, name, description, rule_key, rule_config
) VALUES (
    sqlc.arg(id), sqlc.arg(rule_set_key), sqlc.arg(content_version),
    sqlc.arg(name), sqlc.arg(description), sqlc.arg(rule_key), sqlc.arg(rule_config)
);

-- name: GetStrategyDefinitionVersion :one
SELECT id, strategy_key, content_version, name, description, upside, downside, rule_key, rule_config, created_at
FROM strategy_definition_versions
WHERE id = sqlc.arg(id);

-- name: GetRelicDefinitionVersion :one
SELECT id, relic_key, content_version, name, description, rule_key, rule_config, created_at
FROM relic_definition_versions
WHERE id = sqlc.arg(id);

-- name: GetSynergyDefinitionVersion :one
SELECT id, synergy_key, content_version, name, description, required_count, rule_key, rule_config, created_at
FROM synergy_definition_versions
WHERE id = sqlc.arg(id);

-- name: GetDailyModifierDefinitionVersion :one
SELECT id, modifier_key, content_version, name, description, rule_key, rule_config, created_at
FROM daily_modifier_definition_versions
WHERE id = sqlc.arg(id);

-- name: GetDailyObjectiveDefinitionVersion :one
SELECT id, objective_key, content_version, name, description, progress_label, reward_label, rule_key, rule_config, created_at
FROM daily_objective_definition_versions
WHERE id = sqlc.arg(id);

-- name: GetGameRuleSetVersion :one
SELECT id, rule_set_key, content_version, name, description, rule_key, rule_config, created_at
FROM game_rule_set_versions
WHERE id = sqlc.arg(id);

-- name: ListStrategiesForRelease :many
SELECT id, strategy_key, name, description, upside, downside, rule_key, rule_config
FROM strategy_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY strategy_key;

-- name: ListRelicsForRelease :many
SELECT id, relic_key, name, description, rule_key, rule_config
FROM relic_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY relic_key;

-- name: ListSynergiesForRelease :many
SELECT id, synergy_key, name, description, required_count, rule_key, rule_config
FROM synergy_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY synergy_key;

-- name: ListModifiersForRelease :many
SELECT id, modifier_key, name, description, rule_key, rule_config
FROM daily_modifier_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY modifier_key;

-- name: ListObjectivesForRelease :many
SELECT id, objective_key, name, description, progress_label, reward_label, rule_key, rule_config
FROM daily_objective_definition_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY objective_key;

-- name: ListGameRuleSetsForRelease :many
SELECT id, rule_set_key, name, description, rule_key, rule_config
FROM game_rule_set_versions
WHERE content_version = sqlc.arg(content_version)
ORDER BY rule_set_key;

-- name: ListLegacyCutoverTides :many
SELECT id, day_key, sequence_number, phase, lock_at, settle_after, content_version,
       modifier_definition_version_id, objective_definition_version_id, game_rule_set_version_id
FROM daily_tides
WHERE content_version = sqlc.arg(source_version)
   OR content_version = sqlc.arg(target_version)
   OR modifier_definition_version_id IS NOT NULL
   OR objective_definition_version_id IS NOT NULL
   OR game_rule_set_version_id IS NOT NULL
ORDER BY sequence_number;

-- name: ListCutoverProjectionsForTides :many
SELECT pds.daily_tide_id, pds.id AS state_id, v.schema_version, v.projection
FROM player_daily_states AS pds
JOIN player_daily_context_views AS v ON v.player_daily_state_id = pds.id
JOIN daily_tides AS t ON t.id = pds.daily_tide_id
WHERE t.content_version = sqlc.arg(source_version)
ORDER BY pds.daily_tide_id, pds.id;

-- name: UpdateDailyTideExactContent :exec
UPDATE daily_tides
SET content_version = sqlc.arg(content_version),
    modifier_definition_version_id = sqlc.arg(modifier_definition_version_id),
    objective_definition_version_id = sqlc.arg(objective_definition_version_id),
    game_rule_set_version_id = sqlc.arg(game_rule_set_version_id),
    updated_at = transaction_timestamp()
WHERE id = sqlc.arg(id);

-- name: UpsertDailyTideStrategyVersion :exec
INSERT INTO daily_tide_strategy_versions (
    daily_tide_id, strategy_definition_version_id, content_version
) VALUES (
    sqlc.arg(daily_tide_id), sqlc.arg(strategy_definition_version_id), sqlc.arg(content_version)
)
ON CONFLICT (daily_tide_id, strategy_definition_version_id) DO NOTHING;

-- name: UpdatePlayerDailyStateSelectedStrategy :exec
UPDATE player_daily_states
SET selected_strategy_definition_version_id = sqlc.arg(selected_strategy_definition_version_id),
    updated_at = transaction_timestamp()
WHERE id = sqlc.arg(id);

-- name: CountContentReleaseRows :one
SELECT
    (SELECT count(*) FROM basket_mapping_versions AS b WHERE b.content_version = sqlc.arg(release_version))::bigint AS basket_mapping_count,
    (SELECT count(*) FROM keeper_definition_versions AS k WHERE k.content_version = sqlc.arg(release_version))::bigint AS keeper_definition_count,
    (SELECT count(*)
     FROM basket_mapping_components AS c
     JOIN basket_mapping_versions AS b ON b.id = c.basket_mapping_version_id
     WHERE b.content_version = sqlc.arg(release_version))::bigint AS component_count,
    (SELECT count(*)
     FROM keeper_definition_upgrade_nodes AS n
     JOIN keeper_definition_versions AS k ON k.id = n.keeper_definition_version_id
     WHERE k.content_version = sqlc.arg(release_version))::bigint AS node_count,
    (SELECT count(*) FROM strategy_definition_versions AS s WHERE s.content_version = sqlc.arg(release_version))::bigint AS strategy_count,
    (SELECT count(*) FROM relic_definition_versions AS r WHERE r.content_version = sqlc.arg(release_version))::bigint AS relic_count,
    (SELECT count(*) FROM synergy_definition_versions AS s WHERE s.content_version = sqlc.arg(release_version))::bigint AS synergy_count,
    (SELECT count(*) FROM daily_modifier_definition_versions AS m WHERE m.content_version = sqlc.arg(release_version))::bigint AS modifier_count,
    (SELECT count(*) FROM daily_objective_definition_versions AS o WHERE o.content_version = sqlc.arg(release_version))::bigint AS objective_count,
    (SELECT count(*) FROM game_rule_set_versions AS g WHERE g.content_version = sqlc.arg(release_version))::bigint AS game_rule_set_count;
