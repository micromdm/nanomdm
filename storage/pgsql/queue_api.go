package pgsql

import (
	"context"
	"fmt"

	"github.com/micromdm/nanomdm/storage"
)

const defaultQueueLimit = 100

// RetrieveQueuedCommands retrieves queued commands for the given enrollment ID.
func (s *PgSQLStorage) RetrieveQueuedCommands(ctx context.Context, req *storage.QueueQuery) (*storage.QueueQueryResult, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("enrollment ID is required")
	}

	// Handle pagination
	offset, limit := 0, defaultQueueLimit
	if req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultQueueLimit)
	}

	rows, err := s.db.QueryContext(
		ctx, `
SELECT
    c.command_uuid,
    c.request_type,
    c.command
FROM
    enrollment_queue AS q
    INNER JOIN commands AS c
        ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results AS r
        ON r.command_uuid = q.command_uuid AND r.id = q.id
WHERE
    q.id = $1 AND
    q.active = TRUE AND
    (r.status IS NULL OR r.status = 'NotNow')
ORDER BY
    q.priority DESC,
    q.created_at
LIMIT $2 OFFSET $3;`,
		req.ID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("querying queue: %w", err)
	}
	defer rows.Close()

	result := &storage.QueueQueryResult{
		Commands: make([]*storage.QueueCommand, 0),
	}

	for rows.Next() {
		var uuid, requestType string
		var raw []byte
		if err := rows.Scan(&uuid, &requestType, &raw); err != nil {
			return nil, fmt.Errorf("scanning queue command: %w", err)
		}
		result.Commands = append(result.Commands, storage.NewQueueCommand(uuid, requestType, raw))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating queue commands: %w", err)
	}

	return result, nil
}

// ClearQueueByID clears all queued commands for the given enrollment ID.
func (s *PgSQLStorage) ClearQueueByID(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("enrollment ID is required")
	}

	_, err := s.db.ExecContext(
		ctx, `
UPDATE enrollment_queue
SET active = FALSE
FROM enrollment_queue AS q
    INNER JOIN commands AS c
        ON q.command_uuid = c.command_uuid
    LEFT JOIN command_results AS r
        ON r.command_uuid = q.command_uuid AND r.id = q.id
WHERE
    enrollment_queue.id = $1 AND
    enrollment_queue.active = TRUE AND
    (r.status IS NULL OR r.status = 'NotNow') AND
    enrollment_queue.id = q.id;`,
		id,
	)
	if err != nil {
		return fmt.Errorf("clearing queue: %w", err)
	}

	return nil
}
