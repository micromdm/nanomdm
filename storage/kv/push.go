package kv

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
)

const (
	keyEnrollmentTopic     = "topic"
	keyEnrollmentPushMagic = "push_magic"
	keyEnrollmentToken     = "token"
)

// storePushInfo stores token update push metadata from msg into b for id.
func storePushInfo(ctx context.Context, b kv.CRUDBucket, id string, msg *mdm.TokenUpdate) error {
	return kv.SetMap(ctx, b, map[string][]byte{
		join(id, keyEnrollmentTopic):     []byte(msg.Topic),
		join(id, keyEnrollmentPushMagic): []byte(msg.PushMagic),
		join(id, keyEnrollmentToken):     msg.Token,
	})
}

// RetrievePushInfo retrieves push data for the given ids from the KV store.
func (s *KV) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	r := make(map[string]*mdm.Push)
	for _, id := range ids {
		m, err := kv.GetMap(ctx, s.enrollments, []string{
			join(id, keyEnrollmentTopic),
			join(id, keyEnrollmentPushMagic),
			join(id, keyEnrollmentToken),
		})
		if errors.Is(err, kv.ErrKeyNotFound) {
			// per the API contract drop any errors for this id
			continue
		} else if err != nil {
			return r, fmt.Errorf("retrieving push info for %s: %w", id, err)
		}
		r[id] = &mdm.Push{
			Topic:     string(m[join(id, keyEnrollmentTopic)]),
			PushMagic: string(m[join(id, keyEnrollmentPushMagic)]),
			Token:     m[join(id, keyEnrollmentToken)],
		}
	}
	return r, nil
}
