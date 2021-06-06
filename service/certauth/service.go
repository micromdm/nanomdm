package certauth

import (
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
)

func (s *CertAuth) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	req := r.Clone()
	req.EnrollID = s.normalizer(&m.Enrollment)
	if err := s.associateNewEnrollment(req); err != nil {
		return fmt.Errorf("cert auth: new enrollment: %w", err)
	}
	return s.next.Authenticate(r, m)
}

func (s *CertAuth) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	req := r.Clone()
	req.EnrollID = s.normalizer(&m.Enrollment)
	err := s.validateAssociateExistingEnrollment(req)
	if err != nil {
		return fmt.Errorf("cert auth: existing enrollment: %w", err)
	}
	return s.next.TokenUpdate(r, m)
}

func (s *CertAuth) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	req := r.Clone()
	req.EnrollID = s.normalizer(&m.Enrollment)
	err := s.validateAssociateExistingEnrollment(req)
	if err != nil {
		return fmt.Errorf("cert auth: existing enrollment: %w", err)
	}
	return s.next.CheckOut(r, m)
}

func (s *CertAuth) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	req := r.Clone()
	req.EnrollID = s.normalizer(&results.Enrollment)
	if err := s.validateAssociateExistingEnrollment(req); err != nil {
		return nil, fmt.Errorf("cert auth: existing enrollment: %w", err)
	}
	return s.next.CommandAndReportResults(r, results)
}
