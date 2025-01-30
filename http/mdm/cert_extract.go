package mdm

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"

	"github.com/micromdm/nanomdm/cryptoutil"
)

// ExtractRFC9440 attempts to parse a certificate out of an RFC 9440-style header value.
// RFC 9440 is, basically, the base64-encoded DER certificate surrounded by colons.
func ExtractRFC9440(headerValue string) (*x509.Certificate, error) {
	if len(headerValue) < 3 {
		return nil, errors.New("header too short")
	}
	if headerValue[0] != ':' || headerValue[len(headerValue)-1] != ':' {
		return nil, errors.New("invalid prefix or suffix")
	}
	certBytes, err := base64.StdEncoding.DecodeString(headerValue[1 : len(headerValue)-1])
	if err != nil {
		return nil, fmt.Errorf("decoding base64: %w", err)
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}
	return cert, nil
}

// ExtractQueryEscapedPEM parses a PEM certificate from a URL query-escaped header value.
// This is ostensibly to support Nginx' $ssl_client_escaped_cert in a `proxy_set_header` directive.
func ExtractQueryEscapedPEM(headerValue string) (*x509.Certificate, error) {
	if len(headerValue) < 1 {
		return nil, errors.New("header too short")
	}
	certPEM, err := url.QueryUnescape(headerValue)
	if err != nil {
		return nil, fmt.Errorf("query unescape: %w", err)

	}
	cert, err := cryptoutil.DecodePEMCertificate([]byte(certPEM))
	if err != nil {
		return nil, fmt.Errorf("decode certificate: %w", err)
	}
	return cert, nil
}
