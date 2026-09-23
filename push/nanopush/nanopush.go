// Pacakge nanopush implements an Apple APNs HTTP/2 service for MDM.
// It implements the PushProvider and PushProviderFactory interfaces.
package nanopush

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	t := getOrCreateHTTPTransport(client)
	if t == nil {
		return errors.New("nil transport")
	}
	return http2.ConfigureTransport(t)
}

// getOrCreateHTTPTransport tries to return an [http.Transport] from client.
// If client is the [http.DefaultClient] we return early.
// If client transport is nil we try to clone the default HTTP transport and assign it.
func getOrCreateHTTPTransport(client *http.Client) *http.Transport {
	if client == http.DefaultClient {
		// we don't want to modify the default client
		return nil
	}
	if client.Transport == nil {
		if transport, ok := http.DefaultTransport.(*http.Transport); ok && transport != nil {
			client.Transport = transport.Clone()
		} else if transport == nil {
			client.Transport = &http.Transport{}
		}
	}
	if client.Transport != nil {
		if transport, ok := client.Transport.(*http.Transport); ok {
			return transport
		}
	}
	return nil
}

// UseProxyFromEnvironment configures the HTTP transport of client to
// use the proxy from the environment. See [http.ProxyFromEnvironment].
func UseProxyFromEnvironment(client *http.Client) {
	t := getOrCreateHTTPTransport(client)
	if t == nil {
		return
	}
	t.Proxy = http.ProxyFromEnvironment
}

func defaultNewClient(cert *tls.Certificate) (*http.Client, error) {
	client, err := nanohttp.ClientWithCert(nil, cert)
	if err != nil {
		return client, fmt.Errorf("creating mTLS client: %w", err)
	}
	UseProxyFromEnvironment(client)
	return client, ForceHTTP2(client)
}

// Factory instantiates new PushProviders.
type Factory struct {
	newClient  NewClient
	expiration time.Duration
	workers    int
	url        *string
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
// The default is 5. If workers is less than 1, it will be set to 1.
func WithWorkers(workers int) Option {
	return func(f *Factory) {
		if workers < 1 {
			workers = 1
		}
		f.workers = workers
	}
}

// WithPushServerURL sets the APNs server URL for push notifications.
func WithPushServerURL(url string) Option {
	if err := validatePushServerURL(url); err != nil {
		panic("validating push server url: " + err.Error())
	}
	return func(f *Factory) {
		f.url = &url
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
	url := Production
	if f.url != nil {
		url = *f.url
	}

	p := &Provider{
		expiration: f.expiration,
		workers:    f.workers,
		baseURL:    url,
	}
	var err error
	p.client, err = f.newClient(cert)
	return p, err
}

func validatePushServerURL(pushServerURL string) error {
	parsedURL, err := url.Parse(pushServerURL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" || (parsedURL.Scheme != "" && parsedURL.Host == "") {
		return errors.New("push server URL must contain a scheme")
	}
	if parsedURL.Path != "" {
		return errors.New("push server URL must not contain a path")
	}

	return nil
}
