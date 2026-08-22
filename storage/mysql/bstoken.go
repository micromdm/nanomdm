package mysql

import (
	"database/sql"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	params := sqlc.StoreBootstrapTokenParams{
		BootstrapTokenB64: nullEmptyString(msg.BootstrapToken.BootstrapToken.String()),
		ID:                r.ID,
	}
	err := s.q.StoreBootstrapToken(r.Context(), params)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenB64 sql.NullString
	tokenB64, err := s.q.RetrieveBootstrapToken(r.Context(), r.ID)
	if err != nil || !tokenB64.Valid {
		return nil, err
	}
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(tokenB64.String)
	if err == nil {
		err = s.updateLastSeen(r)
	}
	return bsToken, err
}
