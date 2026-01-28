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

// QueueQuery represents a query for queued commands.
type QueueQuery struct {
	// ID is the enrollment ID to retrieve queued commands for.
	ID string
	// Pagination supports cursor-based pagination.
	Pagination *Pagination
}

// QueueQueryResult contains the result of a queue query.
type QueueQueryResult struct {
	Commands []*mdm.Command `json:"commands"`

	PaginationNextCursor

	// Error contains an error message if there was an error processing the request.
	Error string `json:"error,omitempty"`
}

// CommandQueueAPIStore can retrieve and clear queued commands by enrollment ID.
type CommandQueueAPIStore interface {
	// RetrieveQueuedCommands retrieves queued commands for the given enrollment ID.
	// If no commands are queued, an empty QueueQueryResult is returned with no error.
	// Implementations should not set internal error fields; errors should be returned via the error return value.
	RetrieveQueuedCommands(ctx context.Context, req *QueueQuery) (*QueueQueryResult, error)

	// ClearQueueByID clears all queued commands for the given enrollment ID.
	ClearQueueByID(ctx context.Context, id string) error
}
