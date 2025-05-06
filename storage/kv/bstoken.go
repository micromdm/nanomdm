package kv

import (
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/storage/kv"
)

const keyBootstrapToken = "bstoken"

// StoreBootstrapToken stores the Bootstrap Token into the device KV store.
func (s *KV) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	if err := s.updateLastSeen(r, nil); err != nil {
		return fmt.Errorf("updating last seen: %s", err)
	}
	if r.ParentID != "" {
		return storage.ErrDeviceChannelOnly
	}
	return s.devices.Set(
		r.Context(),
		join(r.ID, keyBootstrapToken),
		msg.BootstrapToken.BootstrapToken,
	)
}

// RetrieveBootstrapToken retrieves the Bootstrap Token from the device KV store.
func (s *KV) RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	if err := s.updateLastSeen(r, nil); err != nil {
		return nil, fmt.Errorf("updating last seen: %s", err)
	}
	if r.ParentID != "" {
		return nil, storage.ErrDeviceChannelOnly
	}
	v, err := s.devices.Get(
		r.Context(),
		join(r.ID, keyBootstrapToken),
	)
	if errors.Is(err, kv.ErrKeyNotFound) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("retrieving bootstrap token: %w", err)
	}
	return &mdm.BootstrapToken{BootstrapToken: v}, nil
}
