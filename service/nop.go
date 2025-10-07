package service

import "github.com/micromdm/nanomdm/mdm"

// NopService is a NanoMDM service that does nothing and returns no errors.
type NopService struct{}

// Authenticate does nothing and returns nil.
func (s *NopService) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	return nil
}

// TokenUpdate does nothing and returns nil.
func (s *NopService) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	return nil
}

// CheckOut does nothing and returns nil.
func (s *NopService) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	return nil
}

// UserAuthenticate does nothing and returns nil, nil.
func (s *NopService) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	return nil, nil
}

// SetBootstrapToken does nothing and returns nil.
func (s *NopService) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	return nil
}

// GetBootstrapToken does nothing and returns nil, nil.
func (s *NopService) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	return nil, nil
}

// DeclarativeManagement does nothing and returns nil.
func (s *NopService) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	return nil, nil
}

// GetToken does nothing and returns nil, nil.
func (s *NopService) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return nil, nil
}

// CommandAndReportResults does nothing and returns nil, nil.
func (s *NopService) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	return nil, nil
}
