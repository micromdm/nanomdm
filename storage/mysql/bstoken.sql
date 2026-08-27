-- name: UpdateBootstrapToken :exec
UPDATE
    devices
SET
    bootstrap_token_b64 = ?,
    bootstrap_token_at = CURRENT_TIMESTAMP
WHERE
    id = ?
LIMIT
    1;

-- name: SelectBootstrapToken :one
SELECT
    bootstrap_token_b64
FROM
    devices
WHERE
    id = ?;
