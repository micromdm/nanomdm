package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyQueueFirst = "first"
	keyQueueLast  = "last"

	keyQueueNext = "next"
	keyQueuePrev = "prev"
)

// queue maintains a linked list in a KV store.
type queue struct {
	b        kv.CRUDBucket
	queueKey string
	name     string
}

func newQueue(b kv.CRUDBucket, queueKey string, name string) *queue {
	return &queue{queueKey: queueKey, b: b, name: name}
}

func (q *queue) queueKeyName(name string) string {
	return join(q.queueKey, "queuekey", q.name, name)
}

func (q *queue) itemKeyName(id string, name string) string {
	return join(q.queueKey, id, "queueitem", name)
}

// getFirst returns the first (start) of the queue linked list for the queue ID.
func (q *queue) getFirst(ctx context.Context) (string, error) {
	r, err := q.b.Get(ctx, q.queueKeyName(keyQueueFirst))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("queue get first: %w", err)
	}
	return string(r), nil
}

// getLast returns the last (end) of the queue linked list for the queue ID.
func (q *queue) getLast(ctx context.Context) (string, error) {
	r, err := q.b.Get(ctx, q.queueKeyName(keyQueueLast))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("queue get last: %w", err)
	}
	return string(r), nil
}

// setFirst sets the first (start) of the queue linked list to the queue ID.
func (q *queue) setFirst(ctx context.Context, id string) error {
	err := q.b.Set(ctx, q.queueKeyName(keyQueueFirst), []byte(id))
	if err != nil {
		return fmt.Errorf("queue set first: %w", err)
	}
	return nil
}

// setLast sets the last (end) of the queue linked list to the queue ID.
func (q *queue) setLast(ctx context.Context, id string) error {
	err := q.b.Set(ctx, q.queueKeyName(keyQueueLast), []byte(id))
	if err != nil {
		return fmt.Errorf("queue set last: %w", err)
	}
	return nil
}

// getNext returns the next linked item for id.
func (q *queue) getNext(ctx context.Context, id string) (string, error) {
	r, err := q.b.Get(ctx, q.itemKeyName(id, keyQueueNext))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("queueitem get next of %s: %w", id, err)
	}
	return string(r), nil
}

// getPrev returns the previous linked list item for id.
func (q *queue) getPrev(ctx context.Context, id string) (string, error) {
	r, err := q.b.Get(ctx, q.itemKeyName(id, keyQueuePrev))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("queueitem get prev of %s: %w", id, err)
	}
	return string(r), nil
}

// setNext sets the next linked item for id.
func (q *queue) setNext(ctx context.Context, id string, next string) error {
	err := q.b.Set(ctx, q.itemKeyName(id, keyQueueNext), []byte(next))
	if err != nil {
		return fmt.Errorf("queueitem set next of %s to %s: %w", id, next, err)
	}
	return nil
}

// setPrev sets the previous linked list item for id.
func (q *queue) setPrev(ctx context.Context, id string, prev string) error {
	err := q.b.Set(ctx, q.itemKeyName(id, keyQueuePrev), []byte(prev))
	if err != nil {
		return fmt.Errorf("queueitem set prev of %s to %s: %w", id, prev, err)
	}
	return nil
}

// unlink removes id from the linked list.
func (q *queue) unlink(ctx context.Context, id string) error {
	prev, err := q.getPrev(ctx, id)
	if err != nil {
		return err
	}
	next, err := q.getNext(ctx, id)
	if err != nil {
		return err
	}
	if prev == "" {
		// a previous item is not recorded.
		// presumed to be the first in the queue.
		if next == "" {
			// prev and next are empty.
			// presumed to be the only item in the queue
			// so delete the first and last pointers
			if err = kv.DeleteSlice(ctx, q.b, []string{
				q.queueKeyName(keyQueueLast),
				q.queueKeyName(keyQueueFirst),
			}); err != nil {
				return fmt.Errorf("first first and last: %w", err)
			}
		} else {
			// a next item exists, but not a prev
			// presumed that we're the first in the list
			// set the first to the next.
			if err = q.setFirst(ctx, next); err != nil {
				return err
			}
			// remove the link to the prev item on the next item (as it is now the first)
			if err = q.b.Delete(ctx, q.itemKeyName(next, keyQueuePrev)); err != nil {
				return err
			}
		}
	} else {
		// a previous item is found
		if next == "" {
			// next is empty
			// presumed to be the last in the queue
			// this means we set the last to be our previous
			if err = q.setLast(ctx, prev); err != nil {
				return err
			}
		} else {
			// both a next and prev pointer exist
			// this means we're in the middle somewhere
			// stitch the next and prev together

			// set the next _of_ the prev to this next
			if err = q.setNext(ctx, prev, next); err != nil {
				return err
			}

			// set the prev _of_ the next to this prev
			if err = q.setPrev(ctx, next, prev); err != nil {
				return err
			}
		}
	}

	// remove the prev and next pointers of this item
	if err = kv.DeleteSlice(ctx, q.b, []string{
		q.itemKeyName(id, keyQueuePrev),
		q.itemKeyName(id, keyQueueNext),
	}); err != nil {
		return fmt.Errorf("delete prev and next: %w", err)
	}

	return nil
}

func (q *queue) enqueue(ctx context.Context, id string) error {
	last, err := q.getLast(ctx)
	if err != nil {
		return err
	}
	if last == "" {
		// last is empty
		// presume we're the first (and last command) and make it so.
		if err = q.setFirst(ctx, id); err != nil {
			return err
		}
		return q.setLast(ctx, id)
	}

	// we're not not the first

	// set us as the next command after last
	if err = q.setNext(ctx, last, id); err != nil {
		return err
	}

	// set our prev as the last command after last
	if err = q.setPrev(ctx, id, last); err != nil {
		return err
	}

	// and set a new queue last
	return q.setLast(ctx, id)
}

// clear removes all items in this linked queue.
func (q *queue) clear(ctx context.Context) error {
	var lastID string
	for id, err := q.getFirst(ctx); id != ""; id, err = q.getNext(ctx, id) {
		if err != nil {
			return fmt.Errorf("getting item from queue: %w", err)
		}
		if lastID != "" {
			err = kv.DeleteSlice(ctx, q.b, []string{
				q.itemKeyName(lastID, keyQueueNext),
				q.itemKeyName(lastID, keyQueuePrev),
			})
			if err != nil {
				return err
			}
		}
		if id == "" {
			return nil
		}
		lastID = id
	}
	return kv.DeleteSlice(ctx, q.b, []string{
		q.queueKeyName(keyQueueFirst),
		q.queueKeyName(keyQueueLast),
	})
}
