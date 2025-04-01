package storage

import "context"

// StoreMigrator retrieves MDM check-ins
type StoreMigrator interface {
	// RetrieveMigrationCheckins sends the (decoded) forms of
	// Authenticate and TokenUpdate messages to the provided channel.
	// Note that order matters: device channel TokenUpdate messages must
	// follow Authenticate messages and user channel TokenUpdates must
	// follow the device channel TokenUpdate.
	RetrieveMigrationCheckins(context.Context, chan<- interface{}) error
}
