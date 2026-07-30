-- name: CreateAccount :exec
INSERT INTO accounts (id, email, password_hash)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password_hash));

-- name: CreatePlayer :exec
INSERT INTO players (id, account_id, public_id)
VALUES (sqlc.arg(id), sqlc.arg(account_id), sqlc.arg(public_id));

-- name: CreateSession :exec
INSERT INTO sessions (id, account_id, player_id, token_digest, expires_at)
VALUES (
    sqlc.arg(id),
    sqlc.arg(account_id),
    sqlc.arg(player_id),
    sqlc.arg(token_digest),
    sqlc.arg(expires_at)
);

-- name: GetAccountForLogin :one
SELECT
    a.id AS account_id,
    p.id AS player_id,
    a.password_hash
FROM accounts AS a
JOIN players AS p
    ON p.account_id = a.id
WHERE a.email = sqlc.arg(email)
  AND a.status = 'ACTIVE';
