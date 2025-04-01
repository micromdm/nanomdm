// Package kv implements a NanoMDM storage backend that uses key-value stores.
package kv

import (
	"strconv"
	"strings"
	"time"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
)

const (
	keySep = "."

	keyLastSeenAt = "last_seen_at"
)

// join concatenates s together by placing [keySep] in-between.
func join(s ...string) string {
	return strings.Join(s, keySep)
}

// KV is a NanoMDM storage backend that uses key-value stores.
type KV struct {
	certAuth, queue, pushCert, users kv.TxnCRUDBucket
	devices, enrollments             kv.TxnBucketWithCRUD
}

// New creates a new NanoMDM storage backend that uses key-value stores.
func New(users, certAuth, queue, pushCert kv.TxnCRUDBucket, devices, enrollments kv.TxnBucketWithCRUD) *KV {
	if devices == nil || users == nil || certAuth == nil || queue == nil || pushCert == nil || enrollments == nil {
		panic("nil bucket")
	}
	return &KV{
		devices:     devices,
		users:       users,
		enrollments: enrollments,
		certAuth:    certAuth,
		queue:       queue,
		pushCert:    pushCert,
	}
}

// timeFmt returns a string representation of microseconds since Unix epoch.
func timeFmt(t time.Time) []byte {
	return []byte(strconv.FormatInt(t.UnixMicro(), 10))
}

// updateLastSeen stores the the current time for the enrollment in r into b.
// The b parameter should only ever be the enrollments bucket or a transaction therein.
// If b is nil then the enrollments bucket of s is used.
func (s *KV) updateLastSeen(r *mdm.Request, b kv.RWBucket) error {
	if b == nil {
		b = s.enrollments
	}
	return b.Set(r.Context, join(r.ID, keyLastSeenAt), timeFmt(time.Now()))
}
