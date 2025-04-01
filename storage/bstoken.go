package storage

import "github.com/micromdm/nanomdm/mdm"

type BootstrapTokenStore interface {
	StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error

	// RetrieveBootstrapToken retrieves the previously-escrowed Bootstrap Token.
	// If a token has not yet been escrowed then a nil token and no error should be returned.
	RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error)
}
