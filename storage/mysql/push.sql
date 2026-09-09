-- name: SelectPushInfo :many
SELECT
    id,
    topic,
    push_magic,
    token_hex
FROM
    enrollments
WHERE
    id IN (sqlc.slice('ids'));
