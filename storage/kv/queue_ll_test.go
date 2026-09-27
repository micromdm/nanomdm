package kv

import (
	"context"
	"reflect"
	"testing"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
)

// walk returns the queue's items in order, first to last.
func walk(t *testing.T, ctx context.Context, q *queue) []string {
	t.Helper()
	var ids []string
	id, err := q.getFirst(ctx)
	for ; id != "" && err == nil; id, err = q.getNext(ctx, id) {
		ids = append(ids, id)
	}
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestUnlink(t *testing.T) {
	for _, tc := range []struct {
		name   string
		queued []string
		unlink string
		want   []string
	}{
		// An id that was never enqueued (or was already unlinked) has no prev
		// and no next pointer, exactly like the only item in a queue. It must
		// not be mistaken for it: unlinking it must leave the queue alone.
		{"absent id, one queued", []string{"a"}, "x", []string{"a"}},
		{"absent id, two queued", []string{"a", "b"}, "x", []string{"a", "b"}},
		{"absent id, empty queue", nil, "x", nil},
		{"only item", []string{"a"}, "a", nil},
		{"first of two", []string{"a", "b"}, "a", []string{"b"}},
		{"middle of three", []string{"a", "b", "c"}, "b", []string{"a", "c"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			q := newQueue(kvmap.New(), "enrollment-1", primaryQueue)
			for _, id := range tc.queued {
				if err := q.enqueue(ctx, id); err != nil {
					t.Fatal(err)
				}
			}
			if err := q.unlink(ctx, tc.unlink); err != nil {
				t.Fatal(err)
			}
			if got := walk(t, ctx, q); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("after unlinking %q from %v the queue is %v, want %v", tc.unlink, tc.queued, got, tc.want)
			}
		})
	}
}
