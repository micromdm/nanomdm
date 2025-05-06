// Package mdm contains structures and helpers related to the Apple MDM protocol.
package mdm

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
)

// Enrollment represents the various enrollment-related data sent with requests.
type Enrollment struct {
	UDID             string `plist:",omitempty"`
	UserID           string `plist:",omitempty"`
	UserShortName    string `plist:",omitempty"`
	UserLongName     string `plist:",omitempty"`
	EnrollmentID     string `plist:",omitempty"`
	EnrollmentUserID string `plist:",omitempty"`
}

// EnrollID contains the custom enrollment IDs derived from enrollment
// data. It's populated by services. Usually this is the main/core
// service so that middleware or storage layers that use the Request
// are able to use the custom IDs.
//
// Be aware that the identifiers here are what are used for MDM client
// identification all around: database primary keys, logging,
// certificate associations, etc. Their format can be changed but it
// must be consistent across the lifetime of any enrolled device.
type EnrollID struct {
	Type     EnrollType
	ID       string
	ParentID string
}

func (eid *EnrollID) Validate() error {
	if eid == nil {
		return errors.New("nil enrollment id")
	}
	if eid.ID == "" {
		return errors.New("empty enrollment id")
	}
	if !eid.Type.Valid() {
		return errors.New("invalid enrollment id type")
	}
	return nil
}

// Request represents an MDM client request.
type Request struct {
	*EnrollID
	Certificate *x509.Certificate
	Params      map[string]string
	ctx         context.Context
}

// NewRequestWithContext creates a new [Request] with ctx and cert.
func NewRequestWithContext(ctx context.Context, cert *x509.Certificate) *Request {
	return &Request{ctx: ctx, Certificate: cert}
}

// Context returns the request's context. To change the context use [Request.WithContext].
// The returned context is always non-nil; it defaults to the background context.
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

// WithContext returns a shallow copy of r with its context changed to ctx.
// The provided ctx must be non-nil.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	return r2
}

// ParseError represents a failure to parse an MDM structure (usually Apple Plist)
type ParseError struct {
	Err     error
	Content []byte
}

// Unwrap returns the underlying error of the ParseError
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Error formats the ParseError as a string
func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error: %v: raw content: %v", e.Err, string(e.Content))
}
