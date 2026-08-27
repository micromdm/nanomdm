package allmulti

import (
	"context"

	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return s.IsPushCertStale(ctx, topic, staleToken)
	})
	return val.(bool), err
}

type retrievePushCertReturns struct {
	pemCert, pemKey []byte
	staleToken      string
}

func (ms *MultiAllStorage) RetrievePushCert(ctx context.Context, topic string) (pemCert, pemKey []byte, staleToken string, err error) {
	val, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		rets := new(retrievePushCertReturns)
		var err error
		rets.pemCert, rets.pemKey, rets.staleToken, err = s.RetrievePushCert(ctx, topic)
		return rets, err
	})
	rets := val.(*retrievePushCertReturns)
	return rets.pemCert, rets.pemKey, rets.staleToken, err
}

func (ms *MultiAllStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	_, err := ms.execStores(ctx, func(s storage.AllStorage) (interface{}, error) {
		return nil, s.StorePushCert(ctx, pemCert, pemKey)
	})
	return err
}
