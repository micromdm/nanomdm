-- name: SelectBlock :one
SELECT
    style,
    clear_on_authenticate
FROM
    blocked_enrollments
WHERE
    id = ?;

-- name: UpsertBlock :exec
INSERT INTO
    blocked_enrollments (id, style, clear_on_authenticate)
VALUES
    (?, ?, ?) AS new ON DUPLICATE KEY
UPDATE
    style = new.style,
    clear_on_authenticate = new.clear_on_authenticate;

-- name: DeleteBlock :exec
DELETE FROM
    blocked_enrollments
WHERE
    id = ?;
