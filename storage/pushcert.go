package storage

import (
	"context"
	"crypto/tls"
)

// PushCertStore stores and retrieves APNs push certificates.
type PushCertStore interface {
	// IsPushCertStale asks whether staleToken is stale or not.
	// The staleToken is returned from RetrievePushCert
	// and should turn stale (and return true) if the certificate has
	// changed—such as being renewed.
	IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error)
	RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error)
	StorePushCert(ctx context.Context, pemCert, pemKey []byte) error
}
