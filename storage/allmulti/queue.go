package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

func (ms *MultiAllStorage) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	finalErr := ms.stores[0].StoreCommandReport(r, report)
	for n, storage := range ms.stores[1:] {
		if err := storage.StoreCommandReport(r, report); err != nil {
			ms.logger.Info("method", "StoreCommandReport", "storage", n+1, "err", err)
			continue
		}
	}
	return finalErr
}

func (ms *MultiAllStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	skipFinal, finalErr := ms.stores[0].RetrieveNextCommand(r, skipNotNow)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.RetrieveNextCommand(r, skipNotNow); err != nil {
			ms.logger.Info("method", "RetrieveNextCommand", "storage", n+1, "err", err)
			continue
		}
	}
	return skipFinal, finalErr
}

func (ms *MultiAllStorage) ClearQueue(r *mdm.Request) error {
	finalErr := ms.stores[0].ClearQueue(r)
	for n, storage := range ms.stores[1:] {
		if err := storage.ClearQueue(r); err != nil {
			ms.logger.Info("method", "ClearQueue", "storage", n+1, "err", err)
			continue
		}
	}
	return finalErr
}

func (ms *MultiAllStorage) EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
	finalMap, finalErr := ms.stores[0].EnqueueCommand(ctx, id, cmd)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.EnqueueCommand(ctx, id, cmd); err != nil {
			ms.logger.Info("method", "EnqueueCommand", "storage", n+1, "err", err)
			continue
		}
	}
	return finalMap, finalErr
}
