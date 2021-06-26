package allmulti

import (
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
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

type storageErrorer func(storage.AllStorage) error

func (ms *MultiAllStorage) runAndLogOthers(storageCallback storageErrorer) {
	for n, storage := range ms.stores[1:] {
		if err := storageCallback(storage); err != nil {
			ms.logger.Info("msg", n+1, "err", err)
		}
	}
}

func (ms *MultiAllStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	err := ms.stores[0].StoreAuthenticate(r, msg)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.StoreAuthenticate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	err := ms.stores[0].StoreTokenUpdate(r, msg)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.StoreTokenUpdate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) Disable(r *mdm.Request) error {
	err := ms.stores[0].Disable(r)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.Disable(r)
	})
	return err
}
