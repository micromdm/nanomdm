package mysql

import (
	"strings"

	"github.com/jessepeterson/nanomdm/mdm"
)

func (s *MySQLStorage) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE id = ?`,
		r.ID,
	)
}

func (s *MySQLStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE sha256 = ?`,
		strings.ToLower(hash),
	)
}

func (s *MySQLStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	return s.queryRowContextRowExists(
		r.Context,
		`SELECT COUNT(*) FROM cert_auth_associations WHERE id = ? AND sha256 = ?`,
		r.ID, strings.ToLower(hash),
	)
}

func (s *MySQLStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	exists, err := s.EnrollmentHasCertHash(r, hash)
	if err != nil {
		return err
	}
	if exists {
		_, err = s.db.ExecContext(
			r.Context,
			`UPDATE cert_auth_associations SET sha256 = ? WHERE id = ?`,
			hash, r.ID,
		)

	} else {
		_, err = s.db.ExecContext(
			r.Context,
			`INSERT INTO cert_auth_associations (id, sha256) VALUES (?, ?)`,
			r.ID, strings.ToLower(hash),
		)
	}
	return err
}
