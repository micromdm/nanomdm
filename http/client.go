package http

import (
	"crypto/tls"
	"errors"
	"net/http"
)

var ErrNilCert = errors.New("nil cert")

// ClientWithCert injects cert for mTLS into a copy of client.
// Transports and TLS configs are created (if nil) or cloned as needed.
// If client is nil the default HTTP client will be used.
func ClientWithCert(client *http.Client, cert *tls.Certificate) (*http.Client, error) {
	if cert == nil {
		return client, ErrNilCert
	}

	if client == nil {
		client = http.DefaultClient
	}
	clientCopy := *client
	client = &clientCopy

	if client.Transport == nil {
		client.Transport = http.DefaultTransport
	}
	var transport *http.Transport
	if t, ok := client.Transport.(*http.Transport); !ok {
		return nil, errors.New("client transport is not an http.Transport")
	} else if t == nil {
		transport = &http.Transport{}
	} else {
		transport = t.Clone()
	}
	client.Transport = transport

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
	}
	transport.TLSClientConfig.Certificates = []tls.Certificate{*cert}

	return client, nil
}
