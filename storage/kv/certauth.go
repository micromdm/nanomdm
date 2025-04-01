package kv

import (
	"context"
	"errors"

	"github.com/micromdm/nanomdm/mdm"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyCertHash = "cert_hash"
	keyHashCert = "hash_cert"
)

// HasCertHash checks if hash has ever been associated to any enrollment.
func (s *KV) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	return s.certAuth.Has(r.Context, join(hash, keyHashCert))
}

// EnrollmentHasCertHash checks that r.ID has any hash associated.
func (s *KV) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	return s.certAuth.Has(r.Context, join(r.ID, keyCertHash))
}

// IsCertHashAssociated checks that r.ID is associated with hash.
func (s *KV) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	kvHash, err := s.certAuth.Get(r.Context, join(r.ID, keyCertHash))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return false, nil
	}
	return hash == string(kvHash), err
}

// AssociateCertHash associates r.ID with hash.
// Here hash is a cryptographic hash of the request certificate.
func (s *KV) AssociateCertHash(r *mdm.Request, hash string) error {
	return kv.PerformCRUDBucketTxn(r.Context, s.certAuth, func(ctx context.Context, b kv.CRUDBucket) error {
		return kv.SetMap(ctx, b, map[string][]byte{
			join(r.ID, keyCertHash): []byte(hash),
			join(hash, keyHashCert): []byte(r.ID),
		})
	})
}

// EnrollmentFromHash retrieves an enrollment ID from a cert hash.
// An empty string is returned if no result is found.
func (s *KV) EnrollmentFromHash(ctx context.Context, hash string) (string, error) {
	r, err := s.certAuth.Get(ctx, join(hash, keyHashCert))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return "", nil
	}
	return string(r), err
}
