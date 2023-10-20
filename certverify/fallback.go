package certverify

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
)

// CertVerifier is a simple interface for verifying a certificate.
type CertVerifier interface {
	Verify(context.Context, *x509.Certificate) error
}

// FallbackVerifier verfies certificate validity using multiple verifiers.
type FallbackVerifier struct {
	verifiers []CertVerifier
}

// NewFallbackVerifier creates a new verifier using other verifiers.
func NewFallbackVerifier(verifiers ...CertVerifier) *FallbackVerifier {
	return &FallbackVerifier{verifiers: verifiers}
}

// Verify performs certificate verification.
// Any verifier returning nil ("passes") will pass (return nil) and not
// check any other verifier.
// If all verifiers return non-nil ("fail") then an error for all
// verifiers will be returned.
func (v *FallbackVerifier) Verify(ctx context.Context, cert *x509.Certificate) error {
	errs := make([]string, len(v.verifiers))
	for i, verifier := range v.verifiers {
		err := verifier.Verify(ctx, cert)
		if err == nil {
			return nil
		}
		errs[i] = fmt.Sprintf("fallback error (%d): %v", i, err)
	}
	return errors.New(strings.Join(errs, "; "))
}
