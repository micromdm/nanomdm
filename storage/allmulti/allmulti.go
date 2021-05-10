package allmulti

import (
	"github.com/jessepeterson/nanomdm/log"
	"github.com/jessepeterson/nanomdm/mdm"
	"github.com/jessepeterson/nanomdm/storage"
)

// MultiAllStorage dispatches to multiple AllStorage instances.
// It returns results and errors from the first store and simply
// logs errors, if any, for the remaining.
type MultiAllStorage struct {
	logger log.Logger
	stores []storage.AllStorage
}

// New creates a new MultiAllStorage dispatcher.
func New(logger log.Logger, stores ...storage.AllStorage) *MultiAllStorage {
	if len(stores) < 1 {
		panic("must supply at least one store")
	}
	return &MultiAllStorage{logger: logger, stores: stores}
}

func (ms *MultiAllStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	finalErr := ms.stores[0].StoreAuthenticate(r, msg)
	for n, storage := range ms.stores[1:] {
		if err := storage.StoreAuthenticate(r, msg); err != nil {
			ms.logger.Info("method", "StoreAuthenticate", "service", n+1, "err", err)
			continue
		}
	}
	return finalErr
}

func (ms *MultiAllStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	finalErr := ms.stores[0].StoreTokenUpdate(r, msg)
	for n, storage := range ms.stores[1:] {
		if err := storage.StoreTokenUpdate(r, msg); err != nil {
			ms.logger.Info("method", "StoreTokenUpdate", "service", n+1, "err", err)
			continue
		}
	}
	return finalErr
}

func (ms *MultiAllStorage) Disable(r *mdm.Request) error {
	finalErr := ms.stores[0].Disable(r)
	for n, storage := range ms.stores[1:] {
		if err := storage.Disable(r); err != nil {
			ms.logger.Info("method", "Disable", "service", n+1, "err", err)
			continue
		}
	}
	return finalErr
}
