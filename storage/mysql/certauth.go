package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
)

// Executes SQL statements that return a single COUNT(*) of rows.
func (s *MySQLStorage) queryRowContextRowExists(ctx context.Context, query string, args ...interface{}) (bool, error) {
	var ct int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&ct)
	return ct > 0, err
}

func (s *MySQLStorage) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	ct, err := s.q.EnrollmentHasCertHash(r.Context(), r.ID)
	return ct > 0, err
}

func (s *MySQLStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context(),
		`SELECT COUNT(*) FROM cert_auth_associations WHERE sha256 = ?;`,
		strings.ToLower(hash),
	)
}

func (s *MySQLStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context(),
		`SELECT COUNT(*) FROM cert_auth_associations WHERE id = ? AND sha256 = ?;`,
		r.ID, strings.ToLower(hash),
	)
}

func (s *MySQLStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	_, err := s.db.ExecContext(
		r.Context(), `
INSERT INTO cert_auth_associations (id, sha256) VALUES (?, ?) AS new
ON DUPLICATE KEY
UPDATE sha256 = new.sha256;`,
		r.ID,
		strings.ToLower(hash),
	)
	return err
}

func (s *MySQLStorage) EnrollmentFromHash(ctx context.Context, hash string) (string, error) {
	var id string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id FROM cert_auth_associations WHERE sha256 = ? LIMIT 1;`,
		hash,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
