-- name: SelectPushCert :one
SELECT
    cert_pem,
    key_pem,
    stale_token
FROM
    push_certs
WHERE
    topic = ?;

-- name: SelectPushCertStaleToken :one
SELECT
    stale_token
FROM
    push_certs
WHERE
    topic = ?;

-- name: UpsertPushCert :exec
INSERT INTO
    push_certs (topic, cert_pem, key_pem, stale_token)
VALUES
    (?, ?, ?, 0) AS new ON DUPLICATE KEY
UPDATE
    cert_pem = new.cert_pem,
    key_pem = new.key_pem,
    push_certs.stale_token = push_certs.stale_token + 1;
