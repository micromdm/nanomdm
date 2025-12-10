// Package storage defines interfaces, types, data, and helpers related
// to storage and retrieval for MDM enrollments and commands.
package storage

import (
	"errors"
)

// ErrDeviceChannelOnly is returned when storage operations are only possible on the device MDM channel.
var ErrDeviceChannelOnly = errors.New("operation supported on device channel only")

// AllStorage represents all required storage by NanoMDM.
type AllStorage interface {
	ServiceStore
	PushStore
	PushCertStore
	CommandEnqueuer
	CertAuthStore
	CertAuthRetriever
	StoreMigrator
	TokenUpdateTallyStore
	PushCertStorer
}

// ServiceStore stores & retrieves both command and check-in data.
type ServiceStore interface {
	CheckinStore
	CommandAndReportResultsStore
	BootstrapTokenStore
}
