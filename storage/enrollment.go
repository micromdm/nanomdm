package storage

import (
	"context"
	"time"

	"github.com/micromdm/nanomdm/mdm"
)

// EnrollmentsQueryFilter contains filter criteria for querying enrollments.
type EnrollmentsQueryFilter struct {
	// IDs filters by enrollment IDs.
	IDs []string `json:"ids,omitempty"`

	// Serials filters by device serial numbers.
	Serials []string `json:"serials,omitempty"`

	// UserShortNames filters by user short names (for user enrollments).
	UserShortNames []string `json:"user_short_names,omitempty"`

	// Types filters by enrollment type strings (e.g., "Device", "User").
	Types []string `json:"types,omitempty"`

	// Enabled filters by enrollment enabled status.
	// If nil, no filter is applied.
	Enabled *bool `json:"enabled,omitempty"`
}

// EnrollmentQueryOptions contains options for enrollment queries.
type EnrollmentQueryOptions struct {
	// IncludeDeviceCert includes the device identity certificate in the response.
	IncludeDeviceCert bool `json:"include_device_cert,omitempty"`

	// IncludeUnlockToken includes the unlock token in the response.
	IncludeUnlockToken bool `json:"include_unlock_token,omitempty"`
}

// EnrollmentsQuery contains the query parameters for listing enrollments.
type EnrollmentsQuery struct {
	// Filter contains the filter criteria for the query.
	Filter *EnrollmentsQueryFilter `json:"filter,omitempty"`

	// Pagination contains pagination parameters.
	Pagination *Pagination `json:"pagination,omitempty"`

	// Options contains query options.
	Options *EnrollmentQueryOptions `json:"options,omitempty"`
}

// DeviceEnrollment contains device-specific enrollment data.
type DeviceEnrollment struct {
	// SerialNumber is the device serial number.
	SerialNumber string `json:"serial_number,omitempty"`

	// DeviceCert is the PEM-encoded device identity certificate.
	// Only populated if IncludeDeviceCert option is set.
	DeviceCert string `json:"device_cert,omitempty"`

	// UnlockToken is the device unlock token (iOS/iPadOS).
	// Only populated if IncludeUnlockToken option is set.
	UnlockToken []byte `json:"unlock_token,omitempty"`
}

// UserEnrollment contains user-specific enrollment data.
type UserEnrollment struct {
	// UserShortName is the user's short name.
	UserShortName string `json:"user_short_name,omitempty"`

	// UserLongName is the user's long (display) name.
	UserLongName string `json:"user_long_name,omitempty"`
}

// Enrollment represents an MDM enrollment.
type Enrollment struct {
	// ID is the unique enrollment identifier.
	ID string `json:"id"`

	// Type is the enrollment type.
	Type mdm.EnrollType `json:"type"`

	// Device contains device-specific enrollment data.
	// Present for device enrollments or as the parent for user enrollments.
	Device *DeviceEnrollment `json:"device,omitempty"`

	// User contains user-specific enrollment data.
	// Only present for user-channel enrollments.
	User *UserEnrollment `json:"user,omitempty"`

	// Enabled indicates if the enrollment is currently enabled.
	Enabled bool `json:"enabled"`

	// TokenUpdateTally is the count of TokenUpdate messages received.
	TokenUpdateTally int `json:"token_update_tally"`

	// LastSeen is the timestamp of the last activity from this enrollment.
	LastSeen time.Time `json:"last_seen"`
}

// EnrollmentsQueryResult contains the result of an enrollments query.
type EnrollmentsQueryResult struct {
	// Enrollments is the list of enrollments matching the query.
	Enrollments []*Enrollment `json:"enrollments"`

	PaginationNextCursor
}

// EnrollmentsStore retrieves enrollment data.
type EnrollmentsStore interface {
	// QueryEnrollments queries enrollments based on the provided request.
	QueryEnrollments(ctx context.Context, req *EnrollmentsQuery) (*EnrollmentsQueryResult, error)
}
