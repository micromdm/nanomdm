package kv

import (
	"context"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const (
	keyEnrollmentBlockStyle       = "block_style"
	keyEnrollmentBlockClearOnAuth = "block_clr_auth"

	valueBlockClearOnAuth = "1"
)

// RetrieveBlock returns the block for r.ID, or nil if r.ID is not blocked.
func (s *KV) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	var block *storage.Block
	return block, kv.PerformBucketTxn(r.Context(), s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		style, err := b.Get(ctx, join(r.ID, keyEnrollmentBlockStyle))
		if errors.Is(err, kv.ErrKeyNotFound) {
			return nil
		} else if err != nil {
			return err
		}
		clearOnAuth, err := b.Has(ctx, join(r.ID, keyEnrollmentBlockClearOnAuth))
		if err != nil {
			return err
		}
		block = &storage.Block{
			Style:               storage.BlockStyle(style),
			ClearOnAuthenticate: clearOnAuth,
		}
		return nil
	})
}

// StoreBlock sets (or replaces) the block for r.ID.
func (s *KV) StoreBlock(r *mdm.Request, block *storage.Block) error {
	if err := block.Validate(); err != nil {
		return err
	}
	return kv.PerformBucketTxn(r.Context(), s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		err := b.Set(ctx, join(r.ID, keyEnrollmentBlockStyle), []byte(block.Style))
		if err != nil {
			return err
		}
		if block.ClearOnAuthenticate {
			return b.Set(ctx, join(r.ID, keyEnrollmentBlockClearOnAuth), []byte(valueBlockClearOnAuth))
		}
		return b.Delete(ctx, join(r.ID, keyEnrollmentBlockClearOnAuth))
	})
}

// RemoveBlock clears any block for r.ID.
func (s *KV) RemoveBlock(r *mdm.Request) error {
	return kv.PerformBucketTxn(r.Context(), s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		return kv.DeleteSlice(ctx, b, []string{
			join(r.ID, keyEnrollmentBlockStyle),
			join(r.ID, keyEnrollmentBlockClearOnAuth),
		})
	})
}
