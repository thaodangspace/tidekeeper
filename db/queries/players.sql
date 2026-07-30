-- name: GetMeForPlayer :one
SELECT
    p.id AS player_id,
    p.public_id AS player_public_id,
    p.onboarding_completed,
    p.locale,
    p.timezone,
    v.public_id AS current_voyage_public_id
FROM players AS p
LEFT JOIN voyages AS v
    ON v.id = p.current_voyage_id
   AND v.player_id = p.id
WHERE p.id = sqlc.arg(player_id);
