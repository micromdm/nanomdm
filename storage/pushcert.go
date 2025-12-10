package storage

import (
	"context"
	"crypto/tls"
)

// PushCertStore retrieves APNs push certificates.
type PushCertStore interface {
	// IsPushCertStale asks whether staleToken is stale or not.
	// The staleToken is returned from RetrievePushCert
	// and should turn stale (and return true) if the certificate has
	// changed—such as being renewed.
	IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error)
	RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error)
}

// PushCertStorer stores APNs push certificates.
type PushCertStorer interface {
	// StorePushCert stores the PEM certificate and private key.
	// The APNs topic (UserID OID), which implementations will likely
	// need to use as a key, is decoded from the from the PEM certificate.
	StorePushCert(ctx context.Context, pemCert, pemKey []byte) error
}
