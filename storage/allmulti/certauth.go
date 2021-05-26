package allmulti

import "github.com/jessepeterson/nanomdm/mdm"

func (ms *MultiAllStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	hasFinal, finalErr := ms.stores[0].HasCertHash(r, hash)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.HasCertHash(r, hash); err != nil {
			ms.logger.Info("method", "HasCertHash", "storage", n+1, "err", err)
			continue
		}
	}
	return hasFinal, finalErr
}

func (ms *MultiAllStorage) EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error) {
	hasFinal, finalErr := ms.stores[0].EnrollmentHasCertHash(r, hash)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.EnrollmentHasCertHash(r, hash); err != nil {
			ms.logger.Info("method", "EnrollmentHasCertHash", "storage", n+1, "err", err)
			continue
		}
	}
	return hasFinal, finalErr
}

func (ms *MultiAllStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	isAssocFinal, finalErr := ms.stores[0].IsCertHashAssociated(r, hash)
	for n, storage := range ms.stores[1:] {
		if _, err := storage.IsCertHashAssociated(r, hash); err != nil {
			ms.logger.Info("method", "IsCertHashAssociated", "storage", n+1, "err", err)
			continue
		}
	}
	return isAssocFinal, finalErr
}

func (ms *MultiAllStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	finalErr := ms.stores[0].AssociateCertHash(r, hash)
	for n, storage := range ms.stores[1:] {
		if err := storage.AssociateCertHash(r, hash); err != nil {
			ms.logger.Info("method", "AssociateCertHash", "storage", n+1, "err", err)
			continue
		}
	}
	return finalErr
}
