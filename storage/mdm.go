package storage

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

type UserAuthenticateStore interface {
	// StoreUserAuthenticate stores the UserAuthenticate check-in message from an enrollment.
	StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error
}

// CheckinStore stores MDM check-in data.
type CheckinStore interface {
	// StoreAuthenticate stores the Authenticate check-in message from an enrollment.
	StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error

	// StoreTokenUpdate stores the TokenUpdate check-in message.
	// Storing this first TokenUpdate message represents a successful enrollment.
	// Note both device and user channel enrollments receive TokenUpdate messages.
	StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error

	// Disable the MDM enrollment.
	// If r is from a device channel then any user channels for this device
	// also need to be disabled.
	Disable(r *mdm.Request) error

	UserAuthenticateStore
}

type TokenUpdateTallyStore interface {
	// RetrieveTokenUpdateTally retrieves the TokenUpdate tally (count) for id.
	// If no tally exists or is not yet set 0 with a nil error should be returned.
	RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error)
}
