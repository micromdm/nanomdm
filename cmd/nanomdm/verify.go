package main

import (
	"crypto/x509"
	"errors"
)

// Verifier is a simple certificate verifier
type Verifier struct {
	verifyOpts x509.VerifyOptions
}

// NewVerifier creates a new Verifier
func NewVerifier(rootsPEM []byte, keyUsages ...x509.ExtKeyUsage) (*Verifier, error) {
	opts := x509.VerifyOptions{
		KeyUsages: keyUsages,
		Roots:     x509.NewCertPool(),
	}
	if len(rootsPEM) == 0 || !opts.Roots.AppendCertsFromPEM(rootsPEM) {
		return nil, errors.New("could not append root CA(s)")
	}
	return &Verifier{
		verifyOpts: opts,
	}, nil
}

// Verify performs certificate verification
func (v *Verifier) Verify(cert *x509.Certificate) error {
	if cert == nil {
		return errors.New("missing MDM certificate")
	}
	if _, err := cert.Verify(v.verifyOpts); err != nil {
		return err
	}
	return nil
}
