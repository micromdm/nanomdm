package storage

import (
	"context"
	"time"

	"github.com/micromdm/nanomdm/mdm"
)

// EnrollmentsQueryFilter is a filter for enrollments. All query parameters are optional.
// If multiple parameters are provided, backend implementations should combine them with logical AND.
// e.g. if both EnrollmentTypes and Serials are provided, only enrollments matching
// both criteria are returned.
type EnrollmentsQueryFilter struct {
	IDs            []string `json:"ids,omitempty"`
	Serials        []string `json:"serials,omitempty"`
	UserShortNames []string `json:"user_short_names,omitempty"`
	Types          []string `json:"types,omitempty"`
	Enabled        *bool    `json:"enabled,omitempty"`
}

// Pagination supports either offset or cursor based pagination methods for results.
// Backend implementations may support one or both methods, and may use either by default.
// Offset and Cursor cannot both be set.
// Backend implementations may have a default Limit, so omitting it may be possible.
// For offset based queries, Offset must be set.
// For cursor based queries, Cursor must be set.
type Pagination struct {
	Limit *int `json:"limit,omitempty"`
	Offset *int  `json:"offset,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

// ValidErr returns any validation errors with p.
func (p Pagination) ValidError() error {
  if p.Cursor != nil && p.Offset != nil {
  	return errors.New("cursor and offset both exist")
  }
  return nil
}

// Valid returns true if p is valid.
func (p Pagination) Valid() bool {
	return p.ValidErr() == nil
}

type EnrollmentQueryOptions struct {
	// By default we do not include the Device Identity certificate in the response.
	IncludeDeviceCert bool `json:"include_device_cert,omitempty"`
	
	// Include the device UnlockToken in the response. By default not included.
	IncludeUnlockToken bool `json:"include_unlock_token,omitempty"`
}

// EnrollmentsQuery represents a query for MDM enrollments.
type EnrollmentsQuery struct {
	Filter    *EnrollmentsQueryFilter `json:"filter,omitempty"`
	Pagination *Pagination             `json:"pagination,omitempty"`
	Options    *EnrollmentQueryOptions `json:"options,omitempty"`
}

type EnrollmentDevice struct {
	SerialNumber string `json:"serial_number"`
	
	// Device Identity certificate in DER encoded form
	DeviceCertPEM *string `json:"device_cert,omitempty"`

	// Unlock Token of device, escrowed during initial and some subsequent Token Update check-in messages.
	UnlockToken *string `json:"unlock_token,omitempty"`
}

type EnrollmentUser struct {
	UserShortName string `json:"user_short_name,omitempty"`
	UserLongName  string `json:"user_long_name,omitempty"`
}

type Enrollment struct {
	// ID is the NanoMDM "enrollment ID":
	// https://github.com/micromdm/nanomdm/blob/main/docs/operations-guide.md#enrollment-ids
	ID string `json:"id"`

	// Type is enrollment type, e.g. Device, User, etc.
	Type mdm.EnrollType `json:"type,omitempty"`

	// Device will be non-nil for device channel enrollments.
	Device *DeviceEnrollment `json:"device,omitempty"`

	// User will be non-nil for user channel enrollments.
	User *UserEnrollment `json:"user,omitempty"`

	// Enabled indicates if the enrollment is active.
	Enabled bool `json:"enabled"`

	// TokenUpdateTally is the number of TokenUpdate messages received for this enrollment.
	TokenUpdateTally int `json:"token_update_tally"`

	// LastSeen is the time of the last request from this enrollment.
	LastSeen time.Time `json:"last_seen"`
}

type EnrollmentsQueryResult struct {
	Enrollments []*Enrollment `json:"enrollments"`

	// NextCursor is the enxt cursor for pagination. 
	// If present and using cursor based pagination more results may be fetched by
	// setting this value in the Cursor field of a subsequent request in the Pagination.
	NextCursor *string `json:"next_cursor,omitempty"`

	// Error is present if there was an error processing the request.
}

type EnrollmentsStore interface {
	// QueryEnrollments retrieves MDM enrollments matching the given request.
	// If no enrollments match the request, an empty EnrollmentAPIResult is returned with no error.
	// Implementations should not set the Error field of EnrollmentAPIResult; errors should be returned via the error return value.
	QueryEnrollments(ctx context.Context, req *EnrollmentsQuery) (*EnrollmentsQueryResult, error)
}
