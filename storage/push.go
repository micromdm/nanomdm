package storage

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

// PushStore retrieves APNs push-related data.
type PushStore interface {
	// RetrievePushInfo retrieves push data for the given ids.
	//
	// If an ID does not exist or is not enrolled properly then
	// implementations should silently skip returning any push data for
	// them. It is up to the caller to discern any missing IDs from the
	// returned map.
	RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error)
}
