package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/block"
	"github.com/micromdm/nanomdm/storage"
)

const blockURL = apiPrefix + "/block/"

func blockStore(t *testing.T, ctx context.Context, store storage.BlockStore) {
	d, auth, tok, err := setupEnrollment()
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

	// the block service answers with the stored style
	svc := block.New(&service.NopService{}, store)
	err = svc.TokenUpdate(d.NewMDMRequest(ctx), tok)
	expectErr(t, err, nil)
	err = svc.Authenticate(d.NewMDMRequest(ctx), auth)
	expectErr(t, err, block.ErrBlocked)
	var httpErr *service.HTTPStatusError
	if !errors.As(err, &httpErr) || httpErr.Status != http.StatusUnauthorized {
		t.Errorf("expected HTTP 401 error, have: %v", err)
	}

	// clear on authenticate removes the block from storage
	if err = store.StoreBlock(r, &storage.Block{Style: storage.StyleUnenroll, ClearOnAuthenticate: true}); err != nil {
		t.Fatal(err)
	}
	err = svc.Authenticate(d.NewMDMRequest(ctx), auth)
	expectErr(t, err, nil)
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

func blockAPI(t *testing.T, ctx context.Context, doer Doer) {
	const id = "block-api-test-id"

	do := func(method string, body []byte) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, blockURL+id, bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := doer.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}
	expectStatus := func(resp *http.Response, want int) {
		t.Helper()
		if resp.StatusCode != want {
			t.Errorf("status: have: %d, want: %d", resp.StatusCode, want)
		}
	}

	expectStatus(do(http.MethodGet, nil), http.StatusNotFound)
	expectStatus(do(http.MethodPut, []byte(`{"style":"invalid"}`)), http.StatusBadRequest)
	expectStatus(do(http.MethodPut, []byte(`not json`)), http.StatusBadRequest)

	expectStatus(do(http.MethodPut, []byte(`{"style":"command-queue","clear_on_authenticate":true}`)), http.StatusNoContent)
	resp := do(http.MethodGet, nil)
	expectStatus(resp, http.StatusOK)
	have := new(storage.Block)
	if err := json.NewDecoder(resp.Body).Decode(have); err != nil {
		t.Fatal(err)
	}
	if want := (storage.Block{Style: storage.StyleCommandQueue, ClearOnAuthenticate: true}); *have != want {
		t.Errorf("block: have: %v, want: %v", *have, want)
	}

	expectStatus(do(http.MethodDelete, nil), http.StatusNoContent)
	expectStatus(do(http.MethodGet, nil), http.StatusNotFound)
	expectStatus(do(http.MethodPost, nil), http.StatusMethodNotAllowed)
}
