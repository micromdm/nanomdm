package file

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

// RetrieveBlock reads the block for r.ID, returning nil if r.ID is not blocked.
func (s *FileStorage) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	e := s.newEnrollment(r.ID)
	b, err := e.readFile(BlockFilename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	block := new(storage.Block)
	return block, json.Unmarshal(b, block)
}

// StoreBlock writes block for r.ID.
func (s *FileStorage) StoreBlock(r *mdm.Request, block *storage.Block) error {
	if err := block.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(block)
	if err != nil {
		return err
	}
	e := s.newEnrollment(r.ID)
	return e.writeFile(BlockFilename, b)
}

// RemoveBlock removes any block for r.ID.
func (s *FileStorage) RemoveBlock(r *mdm.Request) error {
	e := s.newEnrollment(r.ID)
	err := os.Remove(e.dirPrefix(BlockFilename))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
