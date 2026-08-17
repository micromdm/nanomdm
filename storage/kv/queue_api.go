package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/storage"
)

const defaultQueueLimit = 100

// RetrieveQueuedCommands retrieves queued commands for the given enrollment ID.
func (s *KV) RetrieveQueuedCommands(ctx context.Context, req *storage.QueueQuery) (*storage.QueueQueryResult, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("enrollment ID is required")
	}

	var b kv.CRUDBucket = s.queue
	q := newQueue(b, req.ID, primaryQueue)

	// Handle pagination
	offset, limit := 0, defaultQueueLimit
	if req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultQueueLimit)
	}

	result := &storage.QueueQueryResult{
		Commands: make([]*storage.QueueCommand, 0),
	}

	count := 0
	for cmdUUID, err := q.getFirst(ctx); cmdUUID != ""; cmdUUID, err = q.getNext(ctx, cmdUUID) {
		if err != nil {
			return nil, fmt.Errorf("getting item from queue: %w", err)
		}

		// Check if this command has a non-NotNow status (i.e., already completed)
		status, err := b.Get(ctx, q.itemKeyName(cmdUUID, keyQueueStatus))
		if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
			return nil, fmt.Errorf("getting command status: %s: %w", cmdUUID, err)
		}
		if string(status) != "" && string(status) != "NotNow" {
			continue // skip completed commands
		}

		count++

		// Apply offset
		if count <= offset {
			continue
		}

		// Apply limit
		if len(result.Commands) >= limit {
			break
		}

		// Retrieve command data
		m, err := kv.GetMap(ctx, b, []string{
			join(cmdUUID, keyQueueRaw),
			join(cmdUUID, keyQueueRequestType),
		})
		if err != nil {
			return nil, fmt.Errorf("retrieving command: %s: %w", cmdUUID, err)
		}

		result.Commands = append(result.Commands, storage.NewQueueCommand(
			cmdUUID,
			string(m[join(cmdUUID, keyQueueRequestType)]),
			m[join(cmdUUID, keyQueueRaw)],
		))
	}

	return result, nil
}

// ClearQueueByID clears all queued commands for the given enrollment ID.
func (s *KV) ClearQueueByID(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("enrollment ID is required")
	}

	return kv.PerformCRUDBucketTxn(ctx, s.queue, func(ctx context.Context, b kv.CRUDBucket) error {
		q := newQueue(b, id, primaryQueue)
		return q.clear(ctx)
	})
}
