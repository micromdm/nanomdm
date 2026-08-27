package mysql

import (
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// StoreBootstrapToken updates the enrollment last seen and updates the Bootstrap Token for r.ID.
func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	defer s.updateLastSeen(r)

	// store NULL rather than an empty string when the token is empty
	bootstrapTokenB64 := []byte(msg.BootstrapToken.BootstrapToken.String())
	if len(bootstrapTokenB64) < 1 {
		bootstrapTokenB64 = nil
	}
	return s.q.UpdateBootstrapToken(r.Context(), sqlc.UpdateBootstrapTokenParams{
		BootstrapTokenB64: bootstrapTokenB64,
		ID:                r.ID,
	})
}

// RetrieveBootstrapToken updates the enrollment last seen and selects the Bootstrap Token for r.ID.
func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	defer s.updateLastSeen(r)

	tokenB64, err := s.q.SelectBootstrapToken(r.Context(), r.ID)
	if err != nil || len(tokenB64) < 1 {
		// return an error or nothing if we have a NULL bootstrap token.
		// this follows Apple spec for returning success if no bootstrap token.
		return nil, err
	}

	bsToken := new(mdm.BootstrapToken)
	return bsToken, bsToken.SetTokenString(string(tokenB64))
}
