package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error {
	err := ms.stores[0].StoreCommandReport(r, report)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.StoreCommandReport(r, report)
	})
	return err
}

func (ms *MultiAllStorage) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	skipFinal, finalErr := ms.stores[0].RetrieveNextCommand(r, skipNotNow)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.RetrieveNextCommand(r, skipNotNow)
		return err
	})
	return skipFinal, finalErr
}

func (ms *MultiAllStorage) ClearQueue(r *mdm.Request) error {
	err := ms.stores[0].ClearQueue(r)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.ClearQueue(r)
	})
	return err
}

func (ms *MultiAllStorage) EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error) {
	finalMap, finalErr := ms.stores[0].EnqueueCommand(ctx, id, cmd)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.EnqueueCommand(ctx, id, cmd)
		return err
	})
	return finalMap, finalErr
}
