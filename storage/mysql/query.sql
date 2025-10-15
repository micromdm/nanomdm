-- name: DisableEnrollment :exec
UPDATE
  enrollments
SET
  enabled = 0,
  token_update_tally = 0,
  last_seen_at = CURRENT_TIMESTAMP
WHERE
  device_id = ? AND
  enabled = 1;

-- name: UpdateLastSeen :exec
UPDATE enrollments SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: RetrieveTokenUpdateTally :one
SELECT token_update_tally FROM enrollments WHERE id = ?;
