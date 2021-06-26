package allmulti

import (
	"context"
	"crypto/tls"

	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	finalStale, finalErr := ms.stores[0].IsPushCertStale(ctx, topic, staleToken)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.IsPushCertStale(ctx, topic, staleToken)
		return err
	})
	return finalStale, finalErr
}

func (ms *MultiAllStorage) RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error) {
	finalCert, finalToken, finalErr := ms.stores[0].RetrievePushCert(ctx, topic)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, _, err := s.RetrievePushCert(ctx, topic)
		return err
	})

	return finalCert, finalToken, finalErr
}

func (ms *MultiAllStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	err := ms.stores[0].StorePushCert(ctx, pemCert, pemKey)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.StorePushCert(ctx, pemCert, pemKey)
	})
	return err
}
