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

-- name: StoreDeviceTokenUpdateWithoutUnlock :exec
UPDATE devices
SET
    token_update = ?,
    token_update_at = CURRENT_TIMESTAMP
WHERE id = ? LIMIT 1;

-- name: StoreDeviceTokenUpdateWithUnlock :exec
UPDATE devices
SET
    token_update = ?,
    token_update_at = CURRENT_TIMESTAMP,
    unlock_token = ?,
    unlock_token_at = CURRENT_TIMESTAMP
WHERE id = ? LIMIT 1;

-- name: RetrieveMigrationCheckinsDevices :many
SELECT authenticate, token_update FROM devices;

-- name: RetrieveMigrationCheckinsUsers :many
SELECT token_update FROM users;