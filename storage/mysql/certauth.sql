-- name: SelectCountCertAuthByID :one
SELECT
    COUNT(*)
FROM
    cert_auth_associations
WHERE
    id = ?;

-- name: SelectCountCertAuthByHash :one
SELECT
    COUNT(*)
FROM
    cert_auth_associations
WHERE
    sha256 = ?;

-- name: SelectCountCertAuthByIDAndHash :one
SELECT
    COUNT(*)
FROM
    cert_auth_associations
WHERE
    id = ?
    AND sha256 = ?;

-- name: SelectEnrollmentFromHash :one
SELECT
    id
FROM
    cert_auth_associations
WHERE
    sha256 = ?
LIMIT
    1;

-- name: UpsertCertHashAssociation :exec
INSERT INTO
    cert_auth_associations (id, sha256)
VALUES
    (?, ?) AS new ON DUPLICATE KEY
UPDATE
    sha256 = new.sha256;
