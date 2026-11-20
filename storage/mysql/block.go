package mysql

import (
	"database/sql"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// RetrieveBlock selects the block for r.ID, returning nil if r.ID is not blocked.
func (s *MySQLStorage) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	row, err := s.q.SelectBlock(r.Context(), r.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &storage.Block{
		Style:               storage.BlockStyle(row.Style),
		ClearOnAuthenticate: row.ClearOnAuthenticate,
	}, nil
}

// StoreBlock upserts block for r.ID.
func (s *MySQLStorage) StoreBlock(r *mdm.Request, block *storage.Block) error {
	if err := block.Validate(); err != nil {
		return err
	}
	return s.q.UpsertBlock(r.Context(), sqlc.UpsertBlockParams{
		ID:                  r.ID,
		Style:               string(block.Style),
		ClearOnAuthenticate: block.ClearOnAuthenticate,
	})
}

// RemoveBlock deletes any block for r.ID.
func (s *MySQLStorage) RemoveBlock(r *mdm.Request) error {
	return s.q.DeleteBlock(r.Context(), r.ID)
}
