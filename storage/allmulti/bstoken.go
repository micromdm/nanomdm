package allmulti

import (
	"github.com/micromdm/nanomdm/mdm"
)

func (ms *MultiAllStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	finalErr := ms.stores[0].StoreBootstrapToken(r, msg)
	for n, storage := range ms.stores[1:] {
		if err := storage.StoreBootstrapToken(r, msg); err != nil {
			ms.logger.Info("method", "StoreBootstrapToken", "storage", n+1, "err", err)
			continue
		}
	}
	return finalErr
}

func (ms *MultiAllStorage) RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	finalToken, finalErr := ms.stores[0].RetrieveBootstrapToken(r, msg)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.RetrieveBootstrapToken(r, msg); err != nil {
			ms.logger.Info("method", "RetrieveBootstrapToken", "storage", n+1, "err", err)
			continue
		}
	}
	return finalToken, finalErr
}
