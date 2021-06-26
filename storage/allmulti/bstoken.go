package allmulti

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	err := ms.stores[0].StoreBootstrapToken(r, msg)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.StoreBootstrapToken(r, msg)
	})
	return err
}

func (ms *MultiAllStorage) RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	finalToken, finalErr := ms.stores[0].RetrieveBootstrapToken(r, msg)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.RetrieveBootstrapToken(r, msg)
		return err
	})
	return finalToken, finalErr
}
