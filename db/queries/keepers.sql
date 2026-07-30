-- name: ListKeeperInstancesForPlayer :many
SELECT
    i.public_id AS keeper_public_id,
    d.keeper_key AS definition_key,
    d.name,
    d.current_name,
    d.sector_key,
    d.role_key,
    d.rarity_key,
    i.level
FROM keeper_instances AS i
JOIN keeper_definition_versions AS d
    ON d.id = i.keeper_definition_version_id
WHERE i.player_id = sqlc.arg(player_id)
ORDER BY i.acquired_at ASC, i.id ASC;
