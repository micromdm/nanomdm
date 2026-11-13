package pgsql

import (
	"database/sql"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

// RetrieveBlock selects the block for r.ID, returning nil if r.ID is not blocked.
func (s *PgSQLStorage) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	var style string
	var clearOnAuth bool
	err := s.db.QueryRowContext(
		r.Context(),
		`SELECT style, clear_on_authenticate FROM blocked_enrollments WHERE id = $1;`,
		r.ID,
	).Scan(&style, &clearOnAuth)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &storage.Block{
		Style:               storage.BlockStyle(style),
		ClearOnAuthenticate: clearOnAuth,
	}, nil
}

// StoreBlock upserts block for r.ID.
func (s *PgSQLStorage) StoreBlock(r *mdm.Request, block *storage.Block) error {
	if err := block.Validate(); err != nil {
		return err
	}
	_, err := s.db.ExecContext(
		r.Context(), `
INSERT INTO blocked_enrollments (id, style, clear_on_authenticate)
VALUES ($1, $2, $3)
ON CONFLICT ON CONSTRAINT blocked_enrollments_pkey DO UPDATE SET
    style = EXCLUDED.style,
    clear_on_authenticate = EXCLUDED.clear_on_authenticate;`,
		r.ID,
		string(block.Style),
		block.ClearOnAuthenticate,
	)
	return err
}

// RemoveBlock deletes any block for r.ID.
func (s *PgSQLStorage) RemoveBlock(r *mdm.Request) error {
	_, err := s.db.ExecContext(
		r.Context(),
		`DELETE FROM blocked_enrollments WHERE id = $1;`,
		r.ID,
	)
	return err
}
