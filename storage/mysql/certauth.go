package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// EnrollmentHasCertHash selects whether r.ID has any associated certificate hash.
func (s *MySQLStorage) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	ct, err := s.q.SelectCountCertAuthByID(r.Context(), r.ID)
	return ct > 0, err
}

// HasCertHash selects whether hash has ever been associated to any enrollment.
func (s *MySQLStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	ct, err := s.q.SelectCountCertAuthByHash(r.Context(), strings.ToLower(hash))
	return ct > 0, err
}

// IsCertHashAssociated selects whether r.ID is associated with hash.
func (s *MySQLStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	params := sqlc.SelectCountCertAuthByIDAndHashParams{
		ID:     r.ID,
		Sha256: strings.ToLower(hash),
	}
	ct, err := s.q.SelectCountCertAuthByIDAndHash(r.Context(), params)
	return ct > 0, err
}

// AssociateCertHash upserts hash to r.ID. The hash is stored lowercased;
// the "=" lookups lowercase their input to match.
func (s *MySQLStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	return s.q.UpsertCertHashAssociation(r.Context(), sqlc.UpsertCertHashAssociationParams{
		ID:     r.ID,
		Sha256: strings.ToLower(hash),
	})
}

// EnrollmentFromHash selects the enrollment ID corresponding to hash,
// returning an empty ID if no enrollment is associated with hash.
func (s *MySQLStorage) EnrollmentFromHash(ctx context.Context, hash string) (string, error) {
	id, err := s.q.SelectEnrollmentFromHash(ctx, strings.ToLower(hash))
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
