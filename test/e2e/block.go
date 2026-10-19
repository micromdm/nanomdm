package e2e

import (
	"context"
	"errors"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

func blockStore(t *testing.T, ctx context.Context, store storage.BlockStore) {
	d, _, _, err := setupEnrollment()
	if err != nil {
		t.Fatal(err)
	}
	r := d.NewMDMRequest(ctx)

	// not blocked yet
	b, err := store.RetrieveBlock(r)
	if err != nil {
		t.Fatal(err)
	}
	if b != nil {
		t.Fatalf("expected no block, have: %v", b)
	}

	// removing a missing block is not an error
	if err = store.RemoveBlock(r); err != nil {
		t.Fatal(err)
	}

	// invalid styles are rejected
	if err = store.StoreBlock(r, &storage.Block{Style: "invalid"}); !errors.Is(err, storage.ErrInvalidBlockStyle) {
		t.Errorf("expected invalid block style error, have: %v", err)
	}

	want := &storage.Block{Style: storage.StyleServiceUnavailable, ClearOnAuthenticate: true}
	if err = store.StoreBlock(r, want); err != nil {
		t.Fatal(err)
	}
	expectBlock(t, store, r, want)

	// replace the block, including clearing the option
	want = &storage.Block{Style: storage.StyleUnenroll}
	if err = store.StoreBlock(r, want); err != nil {
		t.Fatal(err)
	}
	expectBlock(t, store, r, want)

	// remove the block
	if err = store.RemoveBlock(r); err != nil {
		t.Fatal(err)
	}
	expectBlock(t, store, r, nil)
}

func expectBlock(t *testing.T, store storage.BlockStore, r *mdm.Request, want *storage.Block) {
	t.Helper()
	have, err := store.RetrieveBlock(r)
	if err != nil {
		t.Fatal(err)
	}
	if (have == nil) != (want == nil) {
		t.Fatalf("block: have: %v, want: %v", have, want)
	}
	if want != nil && *have != *want {
		t.Errorf("block: have: %v, want: %v", *have, *want)
	}
}
