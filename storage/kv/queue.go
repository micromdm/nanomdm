package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
)

const (
	keyQueueStatus = "status"
	keyQueueReport = "report"

	keyQueueRaw         = "raw"
	keyQueueRequestType = "req_type"

	primaryQueue = "queue"
)

// lockQueue locks the queue of enrollment id for writing and returns
// the unlock function.
//
// Every queue operation reads the list pointers (first, last, prev,
// next) and then writes some of them. The bucket transactions do not
// keep read locks, so two operations on the same queue could
// interleave: an enqueue that read "last" while the only queued
// command was being acknowledged re-set "last" after the unlink had
// deleted "first" and "last", which left a queue with "last" but no
// "first". RetrieveNextCommand walks from "first", so no command was
// ever delivered to that enrollment again. The lock is held until the
// transaction has committed.
func (s *KV) lockQueue(id string) func() {
	s.queueLocks.Lock(id)
	return func() { s.queueLocks.Unlock(id) }
}

func (s *KV) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	if err := s.updateLastSeen(r, nil); err != nil {
		return fmt.Errorf("updating last seen: %s", err)
	}

	if report.Status == "Idle" {
		return nil
	} else if report.CommandUUID == "" {
		return errors.New("empty command UUID")
	}

	defer s.lockQueue(r.ID)()
	return kv.PerformCRUDBucketTxn(r.Context(), s.queue, func(ctx context.Context, b kv.CRUDBucket) error {
		q := newQueue(b, r.ID, primaryQueue)

		// write the status and raw report
		err := kv.SetMap(ctx, b, map[string][]byte{
			q.itemKeyName(report.CommandUUID, keyQueueReport): report.Raw,
			q.itemKeyName(report.CommandUUID, keyQueueStatus): []byte(report.Status),
		})
		if err != nil {
			return fmt.Errorf("setting command %s: %w", report.CommandUUID, err)
		}

		if report.Status != "NotNow" {
			q := newQueue(b, r.ID, primaryQueue)
			if err = q.unlink(ctx, report.CommandUUID); err != nil {
				return fmt.Errorf("unlink %s: %w", report.CommandUUID, err)
			}
		}

		return nil
	})
}

// RetrieveNextCommand walks the queue linked list to find the next command in the queue.
// If skipNotNow is true then commands that were previously responded to with "NotNow"
func (s *KV) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	var b kv.CRUDBucket = s.queue

	s.queueLocks.RLock(r.ID)
	defer s.queueLocks.RUnlock(r.ID)

	q := newQueue(b, r.ID, primaryQueue)

	for cmdUUID, err := q.getFirst(r.Context()); cmdUUID != ""; cmdUUID, err = q.getNext(r.Context(), cmdUUID) {
		if err != nil {
			return nil, fmt.Errorf("getting item from queue: %w", err)
		} else if cmdUUID == "" {
			return nil, nil
		}

		// get the status of the found command
		status, err := b.Get(r.Context(), q.itemKeyName(cmdUUID, keyQueueStatus))
		if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
			return nil, fmt.Errorf("getting command status: %s: %w", cmdUUID, err)
		}

		if string(status) == "NotNow" && skipNotNow {
			continue
		}

		m, err := kv.GetMap(r.Context(), b, []string{
			join(cmdUUID, keyQueueRaw),
			join(cmdUUID, keyQueueRequestType),
		})
		if err != nil {
			return nil, fmt.Errorf("retrieving command: %s: %w", cmdUUID, err)
		}

		return &mdm.Command{
			CommandUUID: cmdUUID,
			Command: struct {
				RequestType string
			}{
				string(m[join(cmdUUID, keyQueueRequestType)]),
			},
			Raw: m[join(cmdUUID, keyQueueRaw)],
		}, nil
	}

	return nil, nil
}

// ClearQueue clears all queued commands for the enrollment ID in r.
func (s *KV) ClearQueue(r *mdm.Request) error {
	defer s.lockQueue(r.ID)()
	return kv.PerformCRUDBucketTxn(r.Context(), s.queue, func(ctx context.Context, b kv.CRUDBucket) error {
		q := newQueue(b, r.ID, primaryQueue)
		return q.clear(r.Context())
	})
}

func (s *KV) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	if has, err := s.queue.Has(ctx, join(cmd.CommandUUID, keyQueueRaw)); err != nil {
		return nil, err
	} else if has {
		return nil, fmt.Errorf("command already exists: %s", cmd.CommandUUID)
	}

	errs := make(map[string]error)
	err := kv.PerformCRUDBucketTxn(ctx, s.queue, func(ctx context.Context, b kv.CRUDBucket) error {
		err := kv.SetMap(ctx, b, map[string][]byte{
			join(cmd.CommandUUID, keyQueueRaw):         cmd.Raw,
			join(cmd.CommandUUID, keyQueueRequestType): []byte(cmd.Command.RequestType),
		})
		if err != nil {
			return fmt.Errorf("writing command %s: %w", cmd.CommandUUID, err)
		}

		// add to queue for each id
		for _, id := range ids {
			// enqueue writes through to s.queue, not the transaction,
			// so the lock is only needed around it.
			unlock := s.lockQueue(id)
			q := newQueue(s.queue, id, primaryQueue)
			err := q.enqueue(ctx, cmd.CommandUUID)
			unlock()
			if err != nil {
				errs[id] = fmt.Errorf("enqueue for %s: %w", cmd.CommandUUID, err)
			}
		}

		return nil
	})
	return errs, err
}
