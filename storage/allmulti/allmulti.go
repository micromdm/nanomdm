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

type returnCollector struct {
	storeNumber int
	returnValue interface{}
	err         error
}

type errRunner func(storage.AllStorage) (interface{}, error)

func (ms *MultiAllStorage) execStores(r errRunner) (interface{}, error) {
	retChan := make(chan *returnCollector)
	for i, store := range ms.stores {
		go func(n int, s storage.AllStorage) {
			val, err := r(s)
			retChan <- &returnCollector{
				storeNumber: n,
				returnValue: val,
				err:         err,
			}
		}(i, store)
	}
	var finalErr error
	var finalValue interface{}
	for range ms.stores {
		sErr := <-retChan
		if sErr.storeNumber == 0 {
			finalErr = sErr.err
			finalValue = sErr.returnValue
		} else if sErr.err != nil {
			ms.logger.Info("n", sErr.storeNumber, "err", sErr.err)
		}
	}
	return finalValue, finalErr
}

func (ms *MultiAllStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	_, err := ms.execStores(func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreAuthenticate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	_, err := ms.execStores(func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreTokenUpdate(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) Disable(r *mdm.Request) error {
	_, err := ms.execStores(func(s storage.AllStorage) (interface{}, error) {
		return nil, s.Disable(r)
	})
	return err
}
