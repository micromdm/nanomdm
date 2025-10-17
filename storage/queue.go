package storage

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

// CommandAndReportResultsStore stores and retrieves MDM command queue data.
type CommandAndReportResultsStore interface {
	StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error
	RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error)
	ClearQueue(r *mdm.Request) error
}

// CommandEnqueuer is able to enqueue MDM commands.
type CommandEnqueuer interface {
	EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error)
}

// CommandQueueAPIStore can retrieve and clear queued commands by enrollment ID.
type CommandQueueAPIStore interface {
	// RetrieveQueuedCommands retrieves queued commands for the given enrollment ID.
	// The cursor is used for pagination; an empty cursor retrieves from the start.
	// Limit specifies the maximum number of commands to retrieve. If limit is zero or negative, all commands are retrieved.
	// The retrieved commands and the next cursor (if more commands are available) are returned, or an error if any.
	RetrieveQueuedCommands(ctx context.Context, id, cursor string, limit int) (commands []*mdm.Command, nextCursor string, err error)
	// ClearQueueByID clears all queued commands for the given enrollment ID.
	ClearQueueByID(ctx context.Context, id string) error
}
