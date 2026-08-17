package storage

import (
	"context"
	"encoding/base64"

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

// QueueCommand represents a command in the queue for API responses.
type QueueCommand struct {
	CommandUUID string `json:"command_uuid"`
	RequestType string `json:"request_type"`
	// Command is the Base64-encoded raw command plist.
	Command string `json:"command,omitempty"`
}

// NewQueueCommand creates a QueueCommand from the raw command data.
func NewQueueCommand(uuid, requestType string, raw []byte) *QueueCommand {
	return &QueueCommand{
		CommandUUID: uuid,
		RequestType: requestType,
		Command:     base64.StdEncoding.EncodeToString(raw),
	}
}

// QueueQuery contains the query parameters for retrieving queued commands.
type QueueQuery struct {
	ID         string
	Pagination *Pagination
}

// QueueQueryResult is the result of a queue query.
type QueueQueryResult struct {
	Commands []*QueueCommand `json:"commands"`
	PaginationNextCursor
	Error string `json:"error,omitempty"`
}

// CommandQueueAPIStore retrieves and clears command queue data for the API.
type CommandQueueAPIStore interface {
	// RetrieveQueuedCommands returns queued commands for the given enrollment ID.
	RetrieveQueuedCommands(ctx context.Context, req *QueueQuery) (*QueueQueryResult, error)
	// ClearQueueByID clears all queued commands for the given enrollment ID.
	ClearQueueByID(ctx context.Context, id string) error
}
