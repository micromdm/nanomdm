package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
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
	ct, err := s.q.HasCertHash(r.Context(), strings.ToLower(hash))
	return ct > 0, err
}

func (s *MySQLStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	params := sqlc.IsCertHashAssociatedParams{
		ID:     r.ID,
		Sha256: strings.ToLower(hash),
	}
	ct, err := s.q.IsCertHashAssociated(r.Context(), params)
	return ct > 0, err
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
	id, err := s.q.EnrollmentFromHash(ctx, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
