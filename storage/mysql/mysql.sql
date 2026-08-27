-- name: UpdateEnrollmentLastSeen :exec
UPDATE
    enrollments
SET
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    id = ?;
