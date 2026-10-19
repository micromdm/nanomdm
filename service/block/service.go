package block

import (
	"github.com/micromdm/nanomdm/mdm"

	"github.com/micromdm/nanolib/log/ctxlog"
)

// Authenticate clears the block and forwards if the block has
// ClearOnAuthenticate set (i.e. "block until re-enrollment").
func (s *Service) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	req, block := s.retrieveBlock(r, &m.Enrollment, "Authenticate")
	if block != nil && block.ClearOnAuthenticate {
		logger := ctxlog.Logger(r.Context(), s.logger)
		// fail open: a block that can't be removed must not stop re-enrollment
		if err := s.store.RemoveBlock(req); err != nil {
			logger.Info("msg", "removing block; forwarding", "id", req.ID, "request", "Authenticate", "err", err)
		} else {
			logger.Info(
				"msg", "cleared block",
				"id", req.ID,
				"request", "Authenticate",
				"style", block.Style,
			)
		}
		return s.next.Authenticate(r, m)
	}
	if err := s.blockError(r, req, block, "Authenticate", false); err != nil {
		return err
	}
	return s.next.Authenticate(r, m)
}

// TokenUpdate is always forwarded, regardless of block style.
// TokenUpdate completes enrollment and refreshes the APNs push token;
// blocking it would degrade an otherwise-intact enrollment.
func (s *Service) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	return s.next.TokenUpdate(r, m)
}

func (s *Service) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	if err := s.check(r, &m.Enrollment, "CheckOut", false); err != nil {
		return err
	}
	return s.next.CheckOut(r, m)
}

func (s *Service) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	if err := s.check(r, &m.Enrollment, "UserAuthenticate", false); err != nil {
		return nil, err
	}
	return s.next.UserAuthenticate(r, m)
}

func (s *Service) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	if err := s.check(r, &m.Enrollment, "SetBootstrapToken", false); err != nil {
		return err
	}
	return s.next.SetBootstrapToken(r, m)
}

func (s *Service) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	if err := s.check(r, &m.Enrollment, "GetBootstrapToken", false); err != nil {
		return nil, err
	}
	return s.next.GetBootstrapToken(r, m)
}

func (s *Service) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	if err := s.check(r, &m.Enrollment, "DeclarativeManagement", false); err != nil {
		return nil, err
	}
	return s.next.DeclarativeManagement(r, m)
}

func (s *Service) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	if err := s.check(r, &m.Enrollment, "GetToken", false); err != nil {
		return nil, err
	}
	return s.next.GetToken(r, m)
}

func (s *Service) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	if err := s.check(r, &results.Enrollment, "CommandAndReportResults", true); err != nil {
		return nil, err
	}
	return s.next.CommandAndReportResults(r, results)
}
