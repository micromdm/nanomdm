package allmulti

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func (ms *MultiAllStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	hasFinal, finalErr := ms.stores[0].HasCertHash(r, hash)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.HasCertHash(r, hash)
		return err
	})
	return hasFinal, finalErr
}

func (ms *MultiAllStorage) EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error) {
	hasFinal, finalErr := ms.stores[0].EnrollmentHasCertHash(r, hash)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.EnrollmentHasCertHash(r, hash)
		return err
	})
	return hasFinal, finalErr
}

func (ms *MultiAllStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	isAssocFinal, finalErr := ms.stores[0].IsCertHashAssociated(r, hash)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		_, err := s.IsCertHashAssociated(r, hash)
		return err
	})
	return isAssocFinal, finalErr
}

func (ms *MultiAllStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	err := ms.stores[0].AssociateCertHash(r, hash)
	ms.runAndLogOthers(func(s storage.AllStorage) error {
		return s.AssociateCertHash(r, hash)
	})
	return err
}
