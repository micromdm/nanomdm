package hashbody

import (
	"bytes"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
)

// ErrHashBodyHeaderInvalid occurs when the hash of the body contained in the header is invalid.
var ErrHashBodyHeaderInvalid = errors.New("invalid body hash header")

type Doer interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

// SetBodyHashClient is an HTTP client wrapper that adds a body hash header to the request.
type SetBodyHashClient struct {
	doer    Doer
	header  string
	newHash func() hash.Hash
	encoder func([]byte) string
}

// NewSetBodyHashClient sets up a new body hash header client wrapper.
// The upstream client is provided in doer and will panic if nil.
// The name of the HTTP header is provided in header and will panic if nil.
// The encoder and newHash can be nil per [SetBodyHashHeader].
func NewSetBodyHashClient(doer Doer, header string, newHash func() hash.Hash, encoder func([]byte) string) *SetBodyHashClient {
	if doer == nil {
		panic("nil doer")
	}
	if header == "" {
		panic("empty header")
	}
	if newHash == nil {
		newHash = func() hash.Hash { return nil }
	}
	return &SetBodyHashClient{
		doer:    doer,
		header:  header,
		newHash: newHash,
		encoder: encoder,
	}
}

// Do sets the body hash header and dispatches to the upstream client.
func (c *SetBodyHashClient) Do(req *http.Request) (*http.Response, error) {
	if c.doer == nil {
		return nil, errors.New("nil upstream client")
	}
	if c.newHash == nil {
		return nil, errors.New("nil hasher")
	}

	_, err := SetBodyHashHeader(req, c.header, c.newHash(), c.encoder)
	if err != nil {
		return nil, fmt.Errorf("set body hash header: %w", err)
	}

	return c.doer.Do(req)
}

// VerifyBodyHashClient is an HTTP client wrapper that verifies the hash in a header of the response.
type VerifyBodyHashClient struct {
	doer    Doer
	header  string
	newHash func() hash.Hash
	decoder func(string) ([]byte, error)
}

// NewVerifyBodyHashClient sets up a new body hash header verifying client wrapper.
// The upstream client is provided in doer and will panic if nil.
// The name of the HTTP header is provided in header and will panic if nil.
// The decoder and newHash can be nil per [VerifyBodyHashHeader].
func NewVerifyBodyHashClient(doer Doer, header string, newHash func() hash.Hash, decoder func(string) ([]byte, error)) *VerifyBodyHashClient {
	if doer == nil {
		panic("nil doer")
	}
	if header == "" {
		panic("empty header")
	}
	if newHash == nil {
		newHash = func() hash.Hash { return nil }
	}
	return &VerifyBodyHashClient{
		doer:    doer,
		header:  header,
		newHash: newHash,
		decoder: decoder,
	}
}

// Do dispatches to the upstream client and verifies the body hash header.
func (c *VerifyBodyHashClient) Do(req *http.Request) (*http.Response, error) {
	if c.doer == nil {
		return nil, errors.New("nil upstream client")
	}
	if c.newHash == nil {
		return nil, errors.New("nil hasher")
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return resp, err
	}
	// close the body since we intend to read it and replace it
	defer resp.Body.Close()

	var buf bytes.Buffer

	valid, err := VerifyBodyHashHeader(resp, c.header, c.newHash(), c.decoder, &buf)
	if err != nil {
		return nil, fmt.Errorf("verify body hash header: %w", err)
	}
	if !valid {
		return nil, ErrHashBodyHeaderInvalid
	}

	// reset body since we've read it all with the verifier
	resp.Body = io.NopCloser(&buf)

	return resp, nil
}
