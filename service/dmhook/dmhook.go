// Package dmhook provides a NanoMDM Declarative Management service
// that calls out to an HTTP endpoint for the DM protocol.
package dmhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/micromdm/nanomdm/http/hashbody"
	"github.com/micromdm/nanomdm/mdm"
)

type Doer interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

const (
	EnrollmentIDHeader       = "X-Enrollment-ID"
	EnrollmentTypeHeader     = "X-Enrollment-Type"
	EnrollmentParentIDHeader = "X-Enrollment-ParentID" // only if non-empty

	// HTTP header name used when including HMAC signatures.
	HMACHeader = "X-Hmac-Signature"
)

// DMHook is a a NanoMDM Declarative Management service
// that calls out to an HTTP endpoint for the DM protocol.
type DMHook struct {
	urlPrefix *url.URL
	doer      Doer
}

type Option func(*DMHook)

// WithClient configures an HTTP client to use when sending HTTP requests.
// This option should be specified before other options that modify the client.
func WithClient(client Doer) Option {
	return func(d *DMHook) {
		d.doer = client
	}
}

// WithSetHMACSecret will add a SHA-256 HMAC of the webhook DM request body using key.
// The HMAC is provided in the [HMACHeader] header and is Base-64 encoded.
// This option wraps the existing configured HTTP client. Beware client ordering.
func WithSetHMACSecret(key []byte) Option {
	return func(s *DMHook) {
		s.doer = hashbody.NewSetBodyHashClient(
			s.doer,
			HMACHeader,
			func() hash.Hash {
				return hmac.New(sha256.New, key)
			},
			base64.StdEncoding.EncodeToString,
		)
	}
}

// WithVerifyHMACSecret will verify a SHA-256 HMAC of the webhook DM response body using key.
// The HMAC is read from the [HMACHeader] header and is assumed to be Base-64 encoded.
// This option wraps the existing configured HTTP client. Beware client ordering.
func WithVerifyHMACSecret(key []byte) Option {
	return func(s *DMHook) {
		s.doer = hashbody.NewVerifyBodyHashClient(
			s.doer,
			HMACHeader,
			func() hash.Hash {
				return hmac.New(sha256.New, key)
			},
			base64.StdEncoding.DecodeString,
		)
	}
}

// New creates a new Declarative Management HTTP service.
func New(urlPrefix string, opts ...Option) (*DMHook, error) {
	pfx, err := url.Parse(urlPrefix)
	if err != nil {
		return nil, err
	}
	s := &DMHook{
		urlPrefix: pfx,
		doer:      http.DefaultClient,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, err
}

// endpointURL appends endpoint to the path of prefix up to and
// including its last slash. The endpoint is treated as an unescaped
// path: any "%", "?", or "#" is escaped rather than interpreted.
// Its path segments may not be empty, ".", or "..". The query and
// fragment of prefix are dropped.
func endpointURL(prefix *url.URL, endpoint string) (*url.URL, error) {
	for _, seg := range strings.Split(endpoint, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return nil, fmt.Errorf("invalid endpoint path segment: %q", seg)
		}
	}

	dir := prefix.Path[:strings.LastIndex(prefix.Path, "/")+1]
	if dir == "" {
		dir = "/"
	}

	u := *prefix
	u.Path, u.RawPath = dir+endpoint, "" // net/url escapes Path
	u.ForceQuery, u.RawQuery = false, ""
	u.Fragment, u.RawFragment = "", ""
	return &u, nil
}

// DeclarativeManagement calls out to an HTTP endpoint for the Declarative Management protocol.
func (c *DMHook) DeclarativeManagement(r *mdm.Request, message *mdm.DeclarativeManagement) ([]byte, error) {
	if c.urlPrefix == nil {
		return nil, errors.New("nil URL prefix")
	}
	if c.doer == nil {
		return nil, errors.New("nil HTTP client")
	}

	// assemble the prefix URL and the (device-supplied) DM Endpoint together
	targetURL, err := endpointURL(c.urlPrefix, message.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("endpoint URL: %w", err)
	}

	method := http.MethodGet
	if len(message.Data) > 0 {
		method = http.MethodPut
	}

	req, err := http.NewRequestWithContext(
		r.Context(),
		method,
		targetURL.String(),
		bytes.NewBuffer(message.Data),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set(EnrollmentIDHeader, r.ID)
	req.Header.Set(EnrollmentTypeHeader, r.Type.String())
	if r.ParentID != "" {
		req.Header.Set(EnrollmentParentIDHeader, r.ParentID)
	}

	// if we've been given Data (i.e. a DM status report)
	// then let the destination know what the incoming data is
	if len(message.Data) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do DM hook: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != 200 {
		return bodyBytes, fmt.Errorf("DM hook HTTP status: %s", resp.Status)
	}

	return bodyBytes, nil
}
