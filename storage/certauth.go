package storage

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

// CertAuthStore stores and retrieves cert-to-enrollment associations.
// The request enrollment ID should be normalized for just the device channel.
// The hash parameter, when present, is likely (but not required) to be
// a 64-charachter hex string representation of a SHA-256 digest.
type CertAuthStore interface {
	// HasCertHash checks if hash has ever been associated to any enrollment.
	HasCertHash(r *mdm.Request, hash string) (has bool, err error)

	// EnrollmentHasCertHash checks that r.ID has any hash associated.
	// The hash parameter can usually be ignored.
	EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error)

	// IsCertHashAssociated checks that r.ID is associated to hash.
	IsCertHashAssociated(r *mdm.Request, hash string) (bool, error)

	// AssociateCertHash associates r.ID with hash.
	// Here hash is a cryptographic hash of the request certificate.
	AssociateCertHash(r *mdm.Request, hash string) error
}

type CertAuthRetriever interface {
	// EnrollmentFromHash retrieves a normalized enrollment ID from a cert hash.
	// The hash parameter, when present, is likely (but not required) to be
	// a 64-charachter hex string representation of a SHA-256 digest.
	// Implementations should return an empty string if no result is found.
	EnrollmentFromHash(ctx context.Context, hash string) (string, error)
}
