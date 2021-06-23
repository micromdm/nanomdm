package mysql

import (
	"github.com/micromdm/nanomdm/mdm"
)

func (s *MySQLStorage) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	_, err := s.db.ExecContext(
		r.Context,
		`UPDATE devices SET bootstrap_token_b64 = ?, bootstrap_token_at = CURRENT_TIMESTAMP WHERE id = ? LIMIT 1;`,
		nullEmptyString(msg.BootstrapToken.BootstrapToken.String()),
		r.ID,
	)
	return err
}

func (s *MySQLStorage) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	var tokenB64 string
	err := s.db.QueryRowContext(
		r.Context,
		`SELECT bootstrap_token_b64 FROM devices WHERE id = ?;`,
		r.ID,
	).Scan(&tokenB64)
	if err != nil {
		return nil, err
	}
	bsToken := new(mdm.BootstrapToken)
	err = bsToken.SetTokenString(tokenB64)
	return bsToken, err
}
