-- name: InsertCommands :exec
INSERT INTO
    commands (command_uuid, request_type, command)
VALUES
    (?, ?, ?);

-- name: DeleteEnrollmentQueueCommand :exec
DELETE FROM
    enrollment_queue
WHERE
    id = ?
    AND command_uuid = ?;

-- name: DeleteCommandResult :exec
DELETE FROM
    command_results
WHERE
    id = ?
    AND command_uuid = ?;

-- name: SelectCommandReferenced :one
-- Reports if an enrollment still has command_uuid queued or has a result stored for it.
SELECT
    CAST(
        EXISTS(
            SELECT
                1
            FROM
                enrollment_queue AS q
            WHERE
                q.command_uuid = ?
        )
        OR EXISTS(
            SELECT
                1
            FROM
                command_results AS r
            WHERE
                r.command_uuid = ?
        ) AS SIGNED
    ) AS referenced;

-- name: SelectCommandForDelete :one
-- Claims the command row for collection. Returns no rows when another
-- enrollment already holds it, in which case that enrollment does the delete.
SELECT
    command_uuid
FROM
    commands
WHERE
    command_uuid = ? FOR
UPDATE
    SKIP LOCKED;

-- name: DeleteCommand :exec
DELETE FROM
    commands
WHERE
    command_uuid = ?;

-- name: UpsertCommandReport :exec
INSERT INTO
    command_results (
        id,
        command_uuid,
        `status`,
        result
    )
VALUES
    (?, ?, ?, ?) AS new ON DUPLICATE KEY
UPDATE
    `status` = new.`status`,
    result = new.result;

-- name: UpsertCommandReportNotNow :exec
INSERT INTO
    command_results (
        id,
        command_uuid,
        `status`,
        result,
        not_now_at,
        not_now_tally
    )
VALUES
    (?, ?, ?, ?, CURRENT_TIMESTAMP, 1) AS new ON DUPLICATE KEY
UPDATE
    `status` = new.`status`,
    result = new.result,
    command_results.not_now_tally = command_results.not_now_tally + 1;

-- name: SelectNextCommand :one
SELECT
    c.command_uuid,
    c.request_type,
    c.command
FROM
    enrollment_queue AS q
    INNER JOIN commands AS c ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results r ON r.command_uuid = q.command_uuid
    AND r.id = q.id
WHERE
    q.id = ?
    AND q.active = 1
    AND (
        r.status IS NULL
        OR (
            r.status = 'NotNow'
            AND CAST(sqlc.arg("include_not_now") AS SIGNED)
        )
    )
ORDER BY
    q.priority DESC,
    q.created_at
LIMIT
    1;

-- name: UpdateClearQueue :exec
UPDATE
    enrollment_queue AS q
    INNER JOIN enrollments AS e ON q.id = e.id
    INNER JOIN commands AS c ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results r ON r.command_uuid = q.command_uuid
    AND r.id = q.id
SET
    q.active = 0
WHERE
    e.device_id = ?
    AND active = 1
    AND (
        r.status IS NULL
        OR r.status = 'NotNow'
    );
