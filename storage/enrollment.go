package storage

import "context"

// Enrollment is the server-side enrollment summary exposed to administrators.
// Sensitive push tokens and identity certificates are intentionally omitted.
type Enrollment struct {
	ID               string `json:"id"`
	DeviceID         string `json:"device_id"`
	UserID           string `json:"user_id,omitempty"`
	Type             string `json:"type"`
	SerialNumber     string `json:"serial_number,omitempty"`
	Enabled          bool   `json:"enabled"`
	TokenUpdateTally int    `json:"token_update_tally"`
	LastSeenAt       string `json:"last_seen_at"`
	CreatedAt        string `json:"created_at,omitempty"`
	UpdatedAt        string `json:"updated_at,omitempty"`
}

// EnrollmentReader lists persisted enrollment summaries.
type EnrollmentReader interface {
	ListEnrollments(ctx context.Context, query string, limit int) ([]Enrollment, error)
	GetEnrollment(ctx context.Context, id string) (*Enrollment, error)
}
