// Package storage defines interfaces, types, data, and helpers related
// to storage and retrieval for MDM enrollments and commands.
package storage

import (
	"context"
	"crypto/tls"

	"github.com/jessepeterson/nanomdm/mdm"
)

// CheckinStore stores MDM check-in data.
type CheckinStore interface {
	StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error
	StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error
}

// CommandAndReportResultsStore stores and retrieves MDM command queue data.
type CommandAndReportResultsStore interface {
	StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error
	RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error)
	ClearQueue(r *mdm.Request) error
}

// ServiceStore stores & retrieves both command and check-in data.
type ServiceStore interface {
	CheckinStore
	CommandAndReportResultsStore
}

// PushStore stores and retrieves APNs push-related data.
type PushStore interface {
	RetrievePushInfo(context.Context, []string) (map[string]*mdm.Push, error)
}

// PushCertStore stores and retrieves APNs push certificates.
type PushCertStore interface {
	// IsPushCertStale asks a PushStore if the staleToken it has
	// is stale or not. The staleToken is returned from RetrievePushCert
	// and should turn stale (and return true) if the certificate has
	// changed—such as being renewed.
	IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error)
	RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error)
	StorePushCert(ctx context.Context, pemCert, pemKey []byte) error
}

// CommandEnqueuer is able to enqueue MDM commands.
type CommandEnqueuer interface {
	EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error)
}

// CertAuthStore stores and retrieves cert-to-enrollment associations.
type CertAuthStore interface {
	HasCertHash(r *mdm.Request, hash string) (bool, error)
	EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error)
	IsCertHashAssociated(r *mdm.Request, hash string) (bool, error)
	AssociateCertHash(r *mdm.Request, hash string) error
}
