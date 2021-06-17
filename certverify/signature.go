package certverify

import (
	"crypto/x509"
	"errors"
)

// SignatureVerifier is a simple certificate verifier
type SignatureVerifier struct {
	ca *x509.Certificate
}

// NewSignatureVerifier creates a new Verifier
func NewSignatureVerifier(rootPEM []byte) (*SignatureVerifier, error) {
	ca, err := x509.ParseCertificate(rootPEM)
	if err != nil {
		return nil, err
	}
	return &SignatureVerifier{ca: ca}, nil
}

// Verify checks only the signature of the certificate against the CA
func (v *SignatureVerifier) Verify(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("missing MDM certificate")
	}
	return cert.CheckSignatureFrom(v.ca)
}
