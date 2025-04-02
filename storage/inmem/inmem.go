// Package inmem implements an in-memory NanoMDM storage backend.
package inmem

import (
	"github.com/micromdm/nanomdm/storage/kv"

	"github.com/micromdm/nanolib/storage/kv/kvmap"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
)

// InMem is an in-memory storage backend.
type InMem struct {
	*kv.KV
}

// New creates a new in-memory storage backend.
func New() *InMem {
	return &InMem{KV: kv.New(
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
		kvtxn.New(kvmap.New()),
	)}
}
