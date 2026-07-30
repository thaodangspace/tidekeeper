-- name: GetAuthenticatedSession :one
SELECT
    s.id AS session_id,
    s.account_id,
    s.player_id,
    s.expires_at,
    transaction_timestamp()::timestamptz AS authenticated_at
FROM sessions AS s
JOIN accounts AS a
    ON a.id = s.account_id
   AND a.status = 'ACTIVE'
JOIN players AS p
    ON p.id = s.player_id
   AND p.account_id = s.account_id
WHERE s.token_digest = sqlc.arg(token_digest)
  AND s.expires_at > transaction_timestamp()
  AND s.revoked_at IS NULL
  AND s.rotated_at IS NULL
  AND s.replaced_by_session_id IS NULL;
