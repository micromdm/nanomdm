-- name: UpsertAuthenticate :exec
INSERT INTO
    devices (
        id,
        identity_cert,
        serial_number,
        authenticate,
        authenticate_at
    )
VALUES
    (?, ?, ?, ?, CURRENT_TIMESTAMP) AS new ON DUPLICATE KEY
UPDATE
    identity_cert = new.identity_cert,
    serial_number = new.serial_number,
    bootstrap_token_b64 = NULL,
    bootstrap_token_at = NULL,
    authenticate = new.authenticate,
    authenticate_at = CURRENT_TIMESTAMP;

-- name: UpdateDeviceTokenUpdate :exec
UPDATE
    devices
SET
    token_update = ?,
    token_update_at = CURRENT_TIMESTAMP
WHERE
    id = ?
LIMIT
    1;

-- name: UpdateDeviceTokenUpdateWithUnlock :exec
UPDATE
    devices
SET
    token_update = ?,
    token_update_at = CURRENT_TIMESTAMP,
    unlock_token = ?,
    unlock_token_at = CURRENT_TIMESTAMP
WHERE
    id = ?
LIMIT
    1;

-- name: UpsertUserTokenUpdate :exec
INSERT INTO
    users (
        id,
        device_id,
        user_short_name,
        user_long_name,
        token_update,
        token_update_at
    )
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) AS new ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_short_name = new.user_short_name,
    user_long_name = new.user_long_name,
    token_update = new.token_update,
    token_update_at = CURRENT_TIMESTAMP;

-- name: UpsertEnrollment :exec
INSERT INTO
    enrollments (
        id,
        device_id,
        user_id,
        `type`,
        topic,
        push_magic,
        token_hex,
        last_seen_at,
        token_update_tally
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, 1) AS new ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_id = new.user_id,
    `type` = new.`type`,
    topic = new.topic,
    push_magic = new.push_magic,
    token_hex = new.token_hex,
    enabled = 1,
    last_seen_at = CURRENT_TIMESTAMP,
    enrollments.token_update_tally = enrollments.token_update_tally + 1;

-- name: UpsertUserAuthenticate :exec
INSERT INTO
    users (
        id,
        device_id,
        user_short_name,
        user_long_name,
        user_authenticate,
        user_authenticate_at
    )
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) AS new ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_short_name = new.user_short_name,
    user_long_name = new.user_long_name,
    user_authenticate = new.user_authenticate,
    user_authenticate_at = new.user_authenticate_at;

-- name: UpsertUserAuthenticateDigest :exec
INSERT INTO
    users (
        id,
        device_id,
        user_short_name,
        user_long_name,
        user_authenticate_digest,
        user_authenticate_digest_at
    )
VALUES
    (?, ?, ?, ?, ?, CURRENT_TIMESTAMP) AS new ON DUPLICATE KEY
UPDATE
    device_id = new.device_id,
    user_short_name = new.user_short_name,
    user_long_name = new.user_long_name,
    user_authenticate_digest = new.user_authenticate_digest,
    user_authenticate_digest_at = new.user_authenticate_digest_at;

-- name: SelectTokenUpdateTally :one
SELECT
    token_update_tally
FROM
    enrollments
WHERE
    id = ?;

-- name: UpdateDisableEnrollment :exec
UPDATE
    enrollments
SET
    enabled = 0,
    token_update_tally = 0,
    last_seen_at = CURRENT_TIMESTAMP
WHERE
    device_id = ?
    AND enabled = 1;
