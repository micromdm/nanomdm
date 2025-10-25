package storage

import (
	"context"
	"time"

	"github.com/micromdm/nanomdm/mdm"
)

// EnrollmentAPIRequest represents a query for MDM enrollments. All query parameters are optional.
// If multiple parameters are provided, they are combined with logical AND.
// e.g. if both EnrollmentTypes and Serials are provided, only enrollments matching
// both criteria are returned.
type EnrollmentQueryFilter struct {
	IDs            []string
	Serials        []string
	UserShortNames []string
	Types          []string
	Enabled        *bool
}

// Support both limit/offset, as well as cursor
type Pagination struct {
	Limit  int
	Offset int
	Cursor string
}

type EnrollmentQueryOptions struct {
	// By default we do not include the Device Identity certificate in the response.
	IncludeDeviceCert bool
}

type EnrollmentsQuery struct {
	Filter     EnrollmentQueryFilter
	Pagination Pagination
	Options    EnrollmentQueryOptions
}

type DeviceEnrollment struct {
	SerialNumber string `json:"serial_number"`
}

type UserEnrollment struct {
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

type EnrollmentAPIResult struct {
	Enrollments []*Enrollment `json:"enrollments"`
	// NextToken is a cursor for pagination. If non-empty, more results may be fetched by
	// setting this value in the NextToken field of a subsequent request.
	NextToken string `json:"next_token,omitempty"`
	// Error is present if there was an error processing the request.
	Error string `json:"error,omitempty"`
}

type EnrollmentStore interface {
	// QueryEnrollments retrieves MDM enrollments matching the given request.
	// If no enrollments match the request, an empty EnrollmentAPIResult is returned with no error.
	// Implementations should not set the Error field of EnrollmentAPIResult; errors should be returned via the error return value.
	QueryEnrollments(ctx context.Context, req *EnrollmentAPIRequest) (*EnrollmentAPIResult, error)
}
