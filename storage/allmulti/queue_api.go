package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/storage"
)

// RetrieveQueuedCommands retrieves queued commands from the first storage backend.
func (ms *MultiAllStorage) RetrieveQueuedCommands(ctx context.Context, req *storage.QueueQuery) (*storage.QueueQueryResult, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.RetrieveQueuedCommands(ctx, req)
	})
	if val == nil {
		return nil, err
	}
	return val.(*storage.QueueQueryResult), err
}

// ClearQueueByID clears all queued commands from the first storage backend.
func (ms *MultiAllStorage) ClearQueueByID(ctx context.Context, id string) error {
	_, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.ClearQueueByID(ctx, id)
	})
	return err
}
