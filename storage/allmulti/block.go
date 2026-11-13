package allmulti

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	val, err := ms.execStores(r.Context(), func(s storage.AllStorage) (interface{}, error) {
		return s.RetrieveBlock(r)
	})
	return val.(*storage.Block), err
}

func (ms *MultiAllStorage) StoreBlock(r *mdm.Request, block *storage.Block) error {
	_, err := ms.execStores(r.Context(), func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StoreBlock(r, block)
	})
	return err
}

func (ms *MultiAllStorage) RemoveBlock(r *mdm.Request) error {
	_, err := ms.execStores(r.Context(), func(s storage.AllStorage) (interface{}, error) {
		return nil, s.RemoveBlock(r)
	})
	return err
}
