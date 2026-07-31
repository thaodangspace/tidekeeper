-- name: ListPlayerKeeperUnlocks :many
SELECT
    u.keeper_key,
    d.content_version AS definition_version,
    u.unlock_source,
    u.unlocked_at,
    d.name,
    d.current_name,
    d.sector_key,
    d.role_key,
    d.rarity_key
FROM player_keeper_unlocks AS u
JOIN keeper_definition_versions AS d
    ON d.id = u.unlocked_definition_version_id
WHERE u.player_id = sqlc.arg(player_id)
ORDER BY u.unlocked_at ASC, u.keeper_key ASC;
