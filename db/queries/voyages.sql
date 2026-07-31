-- name: GetPublishedVoyageDefinition :one
SELECT id, definition_key, version, status, duration_days,
       starting_hull, starting_supplies, fleet_slot_count, initial_phase,
       launch_presentation, checksum, published_at, created_at
FROM voyage_definition_versions
WHERE definition_key = sqlc.arg(definition_key)
  AND status = 'PUBLISHED'
ORDER BY version DESC
LIMIT 1;

-- name: GetVoyageDefinitionStarterKeepers :many
SELECT vk.voyage_definition_version_id, vk.keeper_definition_version_id, vk.position,
       k.keeper_key, k.name, k.current_key, k.current_name,
       k.sector_key, k.role_key, k.rarity_key, k.passive_rule_key,
       k.passive_rule_config,
       n.node_key AS root_upgrade_node_key
FROM voyage_definition_starter_keepers AS vk
JOIN keeper_definition_versions AS k ON k.id = vk.keeper_definition_version_id
JOIN keeper_definition_upgrade_nodes AS n
    ON n.keeper_definition_version_id = k.id
   AND n.is_root
WHERE vk.voyage_definition_version_id = sqlc.arg(voyage_definition_version_id)
ORDER BY vk.position;

-- name: GetVoyageDefinitionInitialOffers :many
SELECT vo.voyage_definition_version_id, vo.keeper_definition_version_id, vo.position, vo.cost,
       k.keeper_key, k.name, k.current_key, k.current_name,
       k.sector_key, k.role_key, k.rarity_key, k.passive_rule_key,
       k.passive_rule_config
FROM voyage_definition_initial_shop_offers AS vo
JOIN keeper_definition_versions AS k ON k.id = vo.keeper_definition_version_id
WHERE vo.voyage_definition_version_id = sqlc.arg(voyage_definition_version_id)
ORDER BY vo.position;

-- name: GetOpenDailyTide :one
SELECT id, day_key, sequence_number, phase, lock_at, settle_after, content_version, created_at, updated_at
FROM daily_tides
WHERE day_key = sqlc.arg(day_key)
  AND lock_at > transaction_timestamp()
LIMIT 1;

-- name: InsertVoyage :one
INSERT INTO voyages (
    id, public_id, player_id, status, voyage_definition_version_id,
    current_day_number, fund_health, max_fund_health, capital, score,
    row_version, started_at
) VALUES (
    sqlc.arg(id), sqlc.arg(public_id), sqlc.arg(player_id), 'ACTIVE',
    sqlc.arg(voyage_definition_version_id),
    1, sqlc.arg(fund_health), sqlc.arg(max_fund_health), sqlc.arg(capital),
    '0.0000', 1, transaction_timestamp()
)
RETURNING id, public_id, player_id, status, voyage_definition_version_id,
          current_day_number, fund_health, max_fund_health, capital, score,
          row_version, started_at, completed_at, created_at, updated_at;

-- name: LockPlayerRow :one
SELECT id, account_id, public_id, onboarding_completed, locale, timezone,
       current_voyage_id, created_at, updated_at
FROM players
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: InsertKeeperInstance :exec
INSERT INTO keeper_instances (id, public_id, voyage_id, keeper_definition_version_id,
                              upgrade_node_key, acquired_day, acquired_source)
VALUES (sqlc.arg(id), sqlc.arg(public_id), sqlc.arg(voyage_id),
        sqlc.arg(keeper_definition_version_id), sqlc.arg(upgrade_node_key),
        sqlc.arg(acquired_day), sqlc.arg(acquired_source));

-- name: ListVoyageKeepers :many
SELECT
    v.id AS voyage_id,
    v.public_id AS voyage_public_id,
    i.public_id AS keeper_public_id,
    d.keeper_key AS definition_key,
    d.content_version AS definition_version,
    d.name,
    d.current_name,
    d.sector_key,
    d.role_key,
    d.rarity_key,
    n.depth AS node_depth,
    i.upgrade_node_key,
    i.acquired_day,
    i.acquired_source,
    i.acquired_at
FROM voyages AS v
JOIN keeper_instances AS i ON i.voyage_id = v.id
JOIN keeper_definition_versions AS d ON d.id = i.keeper_definition_version_id
JOIN keeper_definition_upgrade_nodes AS n
    ON n.keeper_definition_version_id = i.keeper_definition_version_id
   AND n.node_key = i.upgrade_node_key
WHERE v.id = sqlc.arg(voyage_id)
ORDER BY i.acquired_day ASC, i.acquired_at ASC, i.id ASC;

-- name: InsertPlayerDailyStateWithView :exec
WITH inserted_state AS (
    INSERT INTO player_daily_states (id, player_id, voyage_id, daily_tide_id, day_number, phase, version)
    VALUES (sqlc.arg(state_id), sqlc.arg(player_id), sqlc.arg(voyage_id),
            sqlc.arg(daily_tide_id), 1, 'PREPARATION', 1)
    RETURNING id
)
INSERT INTO player_daily_context_views (player_daily_state_id, schema_version, projection)
VALUES ((SELECT id FROM inserted_state), 1, sqlc.arg(projection));

-- name: InsertVoyageLedgerEntry :exec
INSERT INTO voyage_ledger_entries (id, voyage_id, entry_type, amount, balance_after,
                                    source_type, source_id, reason_key)
VALUES (sqlc.arg(id), sqlc.arg(voyage_id), sqlc.arg(entry_type), sqlc.arg(amount),
        sqlc.arg(balance_after), sqlc.arg(source_type), sqlc.arg(source_id),
        sqlc.arg(reason_key));

-- name: InsertVoyageLifecycleEvent :exec
INSERT INTO voyage_lifecycle_events (id, public_id, voyage_id, event_type, resulting_status)
VALUES (sqlc.arg(id), sqlc.arg(public_id), sqlc.arg(voyage_id),
        sqlc.arg(event_type), sqlc.arg(resulting_status));

-- name: UpdatePlayerCurrentVoyage :exec
UPDATE players
SET current_voyage_id = sqlc.arg(voyage_id),
    onboarding_completed = true,
    updated_at = transaction_timestamp()
WHERE id = sqlc.arg(player_id);

-- name: ClaimIdempotencyKey :one
INSERT INTO idempotency_keys (player_id, scope, key, state, request_hash)
VALUES (sqlc.arg(player_id), sqlc.arg(scope), sqlc.arg(key), 'IN_PROGRESS', sqlc.arg(request_hash))
ON CONFLICT (player_id, scope, key) DO NOTHING
RETURNING player_id, scope, key, state, request_hash, result_voyage_id,
          response_status, response_json, created_at, completed_at;

-- name: GetIdempotencyKeyForUpdate :one
SELECT player_id, scope, key, state, request_hash, result_voyage_id,
       response_status, response_json, created_at, completed_at
FROM idempotency_keys
WHERE player_id = sqlc.arg(player_id)
  AND scope = sqlc.arg(scope)
  AND key = sqlc.arg(key)
FOR UPDATE;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET state = 'COMPLETED',
    result_voyage_id = sqlc.arg(result_voyage_id),
    response_status = sqlc.arg(response_status),
    response_json = sqlc.arg(response_json),
    completed_at = transaction_timestamp()
WHERE player_id = sqlc.arg(player_id)
  AND scope = sqlc.arg(scope)
  AND key = sqlc.arg(key);

-- name: LockVoyageRowForUpdate :one
SELECT id, public_id, player_id, status, voyage_definition_version_id,
       current_day_number, fund_health, max_fund_health, capital, score,
       row_version, started_at, completed_at, created_at, updated_at
FROM voyages
WHERE id = sqlc.arg(id)
FOR UPDATE;

-- name: GetVoyageByPublicID :one
SELECT id, public_id, player_id, status, voyage_definition_version_id,
       current_day_number, fund_health, max_fund_health, capital, score,
       row_version, started_at, completed_at, created_at, updated_at
FROM voyages
WHERE public_id = sqlc.arg(public_id);

-- name: GetVoyageByPublicIDAndPlayerID :one
SELECT id, public_id, player_id, status, voyage_definition_version_id,
       current_day_number, fund_health, max_fund_health, capital, score,
       row_version, started_at, completed_at, created_at, updated_at
FROM voyages
WHERE public_id = sqlc.arg(public_id)
  AND player_id = sqlc.arg(player_id);

-- name: GetCurrentVoyageForPlayer :one
SELECT v.id, v.public_id, v.player_id, v.status, v.voyage_definition_version_id,
       v.current_day_number, v.fund_health, v.max_fund_health, v.capital, v.score,
       v.row_version, v.started_at, v.completed_at, v.created_at, v.updated_at
FROM voyages AS v
JOIN players AS p ON p.current_voyage_id = v.id AND p.id = v.player_id
WHERE p.id = sqlc.arg(player_id);

-- name: GetActiveVoyageForPlayer :one
SELECT v.id, v.public_id, v.player_id, v.status, v.voyage_definition_version_id,
       v.current_day_number, v.fund_health, v.max_fund_health, v.capital, v.score,
       v.row_version, v.started_at, v.completed_at, v.created_at, v.updated_at
FROM voyages AS v
WHERE v.player_id = sqlc.arg(player_id)
  AND v.status = 'ACTIVE';

-- name: UpdateVoyageStatus :one
UPDATE voyages
SET status = sqlc.arg(status),
    completed_at = CASE WHEN sqlc.arg(set_completed)::boolean
                        THEN transaction_timestamp() ELSE completed_at END,
    row_version = row_version + 1,
    updated_at = transaction_timestamp()
WHERE id = sqlc.arg(id)
  AND row_version = sqlc.arg(row_version)
RETURNING id, public_id, player_id, status, voyage_definition_version_id,
          current_day_number, fund_health, max_fund_health, capital, score,
          row_version, started_at, completed_at, created_at, updated_at;

-- name: ListLifecycleEventsForVoyage :many
SELECT id, public_id, voyage_id, event_type, resulting_status, occurred_at, created_at
FROM voyage_lifecycle_events
WHERE voyage_id = sqlc.arg(voyage_id)
ORDER BY occurred_at, id;

-- name: GetVoyageDefinitionByID :one
SELECT id, definition_key, version, status, duration_days,
       starting_hull, starting_supplies, fleet_slot_count, initial_phase,
       launch_presentation, checksum, published_at, created_at
FROM voyage_definition_versions
WHERE id = sqlc.arg(id);
