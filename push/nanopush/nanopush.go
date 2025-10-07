// Pacakge nanopush implements an Apple APNs HTTP/2 service for MDM.
// It implements the PushProvider and PushProviderFactory interfaces.
package nanopush

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	nanohttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/push"
	"golang.org/x/net/http2"
)

// NewClient describes a callback for setting up an HTTP client for Push notifications.
type NewClient func(*tls.Certificate) (*http.Client, error)

// ForceHTTP2 configures HTTP/2 enabled on the transport within client.
// The transport will be cloned if it is the same as the default transport.
func ForceHTTP2(client *http.Client) error {
	if client.Transport == nil || client.Transport == http.DefaultTransport {
		client.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	return http2.ConfigureTransport(client.Transport.(*http.Transport))
}

func defaultNewClient(cert *tls.Certificate) (*http.Client, error) {
	client, err := nanohttp.ClientWithCert(nil, cert)
	if err != nil {
		return client, fmt.Errorf("creating mTLS client: %w", err)
	}
	return client, ForceHTTP2(client)
}

// Factory instantiates new PushProviders.
type Factory struct {
	newClient  NewClient
	expiration time.Duration
	workers    int
}

type Option func(*Factory)

// WithNewClient sets a callback to setup an HTTP client for each
// new Push provider.
func WithNewClient(newClient NewClient) Option {
	return func(f *Factory) {
		f.newClient = newClient
	}
}

// WithExpiration sets the APNs expiration time for the push notifications.
func WithExpiration(expiration time.Duration) Option {
	return func(f *Factory) {
		f.expiration = expiration
	}
}

// WithWorkers sets how many worker goroutines to use when sending pushes.
func WithWorkers(workers int) Option {
	return func(f *Factory) {
		f.workers = workers
	}
}

// NewFactory creates a new Factory.
func NewFactory(opts ...Option) *Factory {
	f := &Factory{
		newClient: defaultNewClient,
		workers:   5,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// NewPushProvider generates a new PushProvider given a tls keypair.
func (f *Factory) NewPushProvider(cert *tls.Certificate) (push.PushProvider, error) {
	p := &Provider{
		expiration: f.expiration,
		workers:    f.workers,
		baseURL:    Production,
	}
	var err error
	p.client, err = f.newClient(cert)
	return p, err
}
