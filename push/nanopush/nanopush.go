// Pacakge nanopush implements an Apple APNs HTTP/2 service for MDM.
// It implements the PushProvider and PushProviderFactory interfaces.
package nanopush

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"golang.org/x/net/http2"
)

// NewClient describes a callback for setting up an HTTP client for Push notifications.
type NewClient func(*tls.Certificate) (*http.Client, error)

// ClientWithCert configures an mTLS client cert on the HTTP client.
func ClientWithCert(client *http.Client, cert *tls.Certificate) (*http.Client, error) {
	if cert == nil {
		return client, errors.New("no cert provided")
	}
	if client == nil {
		clone := *http.DefaultClient
		client = &clone
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{*cert},
	}
	config.BuildNameToCertificate()
	if client.Transport == nil {
		client.Transport = &http.Transport{}
	}
	transport := client.Transport.(*http.Transport)
	transport.TLSClientConfig = config
	// force HTTP/2
	err := http2.ConfigureTransport(transport)
	return client, err
}

func defaultNewClient(cert *tls.Certificate) (*http.Client, error) {
	return ClientWithCert(nil, cert)
}

// Factory instantiates new PushProviders.
type Factory struct {
	newClient  NewClient
	expiration time.Duration
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

// NewFactory creates a new Factory.
func NewFactory(opts ...Option) *Factory {
	f := &Factory{
		newClient: defaultNewClient,
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// NewPushProvider generates a new PushProvider given a tls keypair.
func (f *Factory) NewPushProvider(cert *tls.Certificate) (push.PushProvider, error) {
	p := &Provider{expiration: f.expiration}
	var err error
	p.client, err = f.newClient(cert)
	return p, err
}

type Provider struct {
	client     *http.Client
	expiration time.Duration
}

func (p *Provider) do1(ctx context.Context, pushInfo *mdm.Push) *push.Response {
	payload := []byte(`{"mdm":"` + pushInfo.PushMagic + `"}`)
	url := fmt.Sprintf("%s/3/device/%s", "https://api.push.apple.com", pushInfo.Token.String())
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return &push.Response{Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	if p.expiration > 0 {
		exp := time.Now().Add(p.expiration)
		req.Header.Set("apns-expiration", strconv.FormatInt(exp.Unix(), 10))
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return &push.Response{Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// TODO: better parsing!
		bodyR, _ := io.ReadAll(resp.Body)
		return &push.Response{Err: fmt.Errorf("invalid status code: %d: %s", resp.StatusCode, string(bodyR))}
	}
	return &push.Response{Id: resp.Header.Get("apns-id")}

}

func (p *Provider) Push(pushInfos []*mdm.Push) (map[string]*push.Response, error) {
	if len(pushInfos) < 1 {
		return nil, errors.New("no push data provided")
	}
	ret := make(map[string]*push.Response)
	for _, pushInfo := range pushInfos {
		if pushInfo == nil {
			continue
		}
		ret[pushInfo.Token.String()] = p.do1(context.TODO(), pushInfo)
	}
	return ret, nil
}
