package file

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

const defaultQueueLimit = 100

// RetrieveQueuedCommands retrieves queued commands for the given enrollment ID.
func (s *FileStorage) RetrieveQueuedCommands(_ context.Context, req *storage.QueueQuery) (*storage.QueueQueryResult, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("enrollment ID is required")
	}

	// Handle pagination
	offset, limit := 0, defaultQueueLimit
	if req.Pagination != nil {
		offset, limit = req.Pagination.DefaultOffsetLimit(defaultQueueLimit)
	}

	result := &storage.QueueQueryResult{
		Commands: make([]*storage.QueueCommand, 0),
	}

	e := s.newEnrollment(req.ID)

	// Collect commands from both Queue and NotNow queues
	count := 0
	for _, sub := range []string{subNotNow, subQueue} {
		q := e.newQueue(sub)
		entries, err := os.ReadDir(q.dir())
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("reading queue directory: %w", err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".plist") || strings.HasSuffix(name, ".result.plist") {
				continue
			}

			count++

			// Apply offset
			if count <= offset {
				continue
			}

			// Apply limit
			if len(result.Commands) >= limit {
				return result, nil
			}

			raw, err := os.ReadFile(path.Join(q.dir(), name))
			if err != nil {
				return nil, fmt.Errorf("reading command file: %w", err)
			}

			cmd, err := mdm.DecodeCommand(raw)
			if err != nil {
				continue // skip malformed commands
			}

			result.Commands = append(result.Commands, storage.NewQueueCommand(
				cmd.CommandUUID,
				cmd.Command.RequestType,
				cmd.Raw,
			))
		}
	}

	return result, nil
}

// ClearQueueByID clears all queued commands for the given enrollment ID.
func (s *FileStorage) ClearQueueByID(_ context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("enrollment ID is required")
	}

	e := s.newEnrollment(id)
	dest := e.newQueue(subInactive)

	for _, q := range []*queue{e.newQueue(subQueue), e.newQueue(subNotNow)} {
		raw, err := q.getNext()
		for raw != nil && err == nil {
			err = q.move(raw.CommandUUID, dest)
			if err != nil {
				return fmt.Errorf("clearing queue item: %w", err)
			}
			raw, err = q.getNext()
		}
		if err != nil {
			return fmt.Errorf("clearing queue: %w", err)
		}
	}

	return nil
}
