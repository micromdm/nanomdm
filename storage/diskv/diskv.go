// Package diskv implements a NanoMDM storage backend using the diskv key-value store.
package diskv

import (
	"path/filepath"
	"strings"

	"github.com/micromdm/nanomdm/storage/kv"

	nlkv "github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanolib/storage/kv/kvdiskv"
	"github.com/micromdm/nanolib/storage/kv/kvtxn"
	"github.com/peterbourgon/diskv/v3"
)

// Diskv is a storage backend that uses diskv.
type Diskv struct {
	*kv.KV
}

// Split2X2Transform splits key into a path like /00/01 for a key of "0001".
// The key will be prefixed with zeros if its length is less than 4.
func Split2X2Transform(key string) []string {
	if len(key) < 4 {
		key = strings.Repeat("0", 4-len(key)) + key
	}
	return []string{key[0:2], key[2:4]}
}

// StripPrefixTransform wraps next in a function that trims prefix from the key.
func StripPrefixTransform(next diskv.TransformFunction, prefix string) diskv.TransformFunction {
	return func(key string) []string {
		return next(strings.TrimPrefix(key, prefix))
	}
}

func newBucket(path, name string) nlkv.TxnBucketWithCRUD {
	return newBucketWithTransform(path, name, Split2X2Transform)
}

func newBucketWithTransform(path, name string, transform diskv.TransformFunction) nlkv.TxnBucketWithCRUD {
	return kvtxn.New(kvdiskv.New(diskv.New(diskv.Options{
		BasePath:     filepath.Join(path, name),
		Transform:    transform,
		CacheSizeMax: 1024 * 1024,
	})))
}

// New creates a new storage backend that uses diskv.
func New(path string) *Diskv {
	return &Diskv{KV: kv.New(
		newBucket(path, "users"),
		newBucket(path, "cert_auth"),
		newBucket(path, "queue"),
		// try to store the push certs with transformed keys of the UUID within the Topic
		newBucketWithTransform(
			path,
			"push_cert",
			StripPrefixTransform(Split2X2Transform, "com.apple.mgmt.External."),
		),
		newBucket(path, "devices"),
		newBucket(path, "enrollments"),
	)}
}
