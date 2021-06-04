package allmulti

import "context"

func (ms *MultiAllStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {
	ms.logger.Info("msg", "only using first store for migration")
	return ms.stores[0].RetrieveMigrationCheckins(ctx, c)
}
