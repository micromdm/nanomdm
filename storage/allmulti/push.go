package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

func (ms *MultiAllStorage) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	finalMap, finalErr := ms.stores[0].RetrievePushInfo(ctx, ids)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.RetrievePushInfo(ctx, ids); err != nil {
			ms.logger.Info("method", "RetrievePushInfo", "storage", n+1, "err", err)
			continue
		}
	}
	return finalMap, finalErr
}
