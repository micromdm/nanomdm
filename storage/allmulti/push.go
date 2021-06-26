package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	finalMap, finalErr := ms.stores[0].RetrievePushInfo(ctx, ids)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.RetrievePushInfo(ctx, ids)
		return err
	})
	return finalMap, finalErr
}
