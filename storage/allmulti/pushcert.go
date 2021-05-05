package allmulti

import (
	"context"
	"crypto/tls"
)

func (ms *MultiAllStorage) IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error) {
	finalStale, finalErr := ms.stores[0].IsPushCertStale(ctx, topic, staleToken)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.IsPushCertStale(ctx, topic, staleToken); err != nil {
			ms.logger.Info("method", "IsPushCertStale", "service", n+1, "err", err)
			continue
		}
	}
	return finalStale, finalErr
}

func (ms *MultiAllStorage) RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error) {
	finalCert, finalToken, finalErr := ms.stores[0].RetrievePushCert(ctx, topic)
	for n, storage := range ms.stores[1:] {
		if _, _, err := storage.RetrievePushCert(ctx, topic); err != nil {
			ms.logger.Info("method", "RetrievePushCert", "service", n+1, "err", err)
			continue
		}
	}
	return finalCert, finalToken, finalErr
}

func (ms *MultiAllStorage) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	finalErr := ms.stores[0].StorePushCert(ctx, pemCert, pemKey)
	for n, storage := range ms.stores[1:] {
		if err := storage.StorePushCert(ctx, pemCert, pemKey); err != nil {
			ms.logger.Info("method", "StorePushCert", "service", n+1, "err", err)
			continue
		}
	}
	return finalErr
}
