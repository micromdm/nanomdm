package kv

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/plist"
)

func getDecodeCheckIn(ctx context.Context, b kv.ROBucket, key string) (interface{}, error) {
	checkInBytes, err := b.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("getting check-in message: %w", err)
	}
	return mdm.DecodeCheckin(checkInBytes)
}

// synthesizeBSTokenMsg creates a SetBootstrapToken message from a Bootstrap token and UDID.
// TODO: to correctly synthesize this this may require knowing the device
// type so that we know which fields to populate in the Check-In message.
// for now just assume it's a normal Device type and use the UDID field.
func synthesizeBSTokenMsg(udid string, bsToken []byte) (*mdm.SetBootstrapToken, error) {
	bsTok := &mdm.SetBootstrapToken{
		MessageType:    mdm.MessageType{MessageType: "SetBootstrapToken"},
		Enrollment:     mdm.Enrollment{UDID: udid},
		BootstrapToken: mdm.BootstrapToken{BootstrapToken: bsToken},
	}
	var err error
	bsTok.Raw, err = plist.Marshal(bsTok)
	return bsTok, err
}

// RetrieveMigrationCheckins sends ordered enrollment-related MDM check-in messages to out.
func (s *KV) RetrieveMigrationCheckins(ctx context.Context, out chan<- interface{}) error {
	var ids []string
	for key := range s.devices.Keys(ctx, nil) {
		if strings.HasSuffix(key, keySep+keyDeviceAuthenticate) {
			id := key[0 : len(key)-(len(keySep)+len(keyDeviceAuthenticate))]
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		if disabled, err := s.enrollments.Has(ctx, join(id, keyEnrollmentDisabled)); err != nil {
			return fmt.Errorf("checking for disablement for %s: %w", id, err)
		} else if disabled {
			// this enrollment ID is disabled, skip it
			continue
		}

		auth, err := getDecodeCheckIn(ctx, s.devices, join(id, keyDeviceAuthenticate))
		if errors.Is(err, kv.ErrKeyNotFound) {
			// invalid enrollment
			continue
		} else if err != nil {
			return fmt.Errorf("getting authenticate check-in for %s: %w", id, err)
		}
		out <- auth

		// TODO: handle the situation where a TokenUpdate does not contain
		// the UnlockToken but we have it stored.
		tokUpd, err := getDecodeCheckIn(ctx, s.devices, join(id, keyDeviceTokenUpdate))
		if errors.Is(err, kv.ErrKeyNotFound) {
			// invalid enrollment
			continue
		} else if err != nil {
			return fmt.Errorf("getting token update check-in for %s: %w", id, err)
		}
		out <- tokUpd

		// try to handle the bootstrap token
		if bsToken, err := s.devices.Get(ctx, join(id, keyBootstrapToken)); err == nil {
			bsTok, err := synthesizeBSTokenMsg(id, bsToken)
			if err == nil {
				out <- bsTok
			}
		} else if !errors.Is(err, kv.ErrKeyNotFound) {
			return fmt.Errorf("getting bootstrap token for %s: %w", id, err)
		}

		// now loop through any user channel enrollments for this id
		var userIDs []string
		pfx := join(id, keyEnrollmentUserChannel) + keySep
		for key := range s.enrollments.KeysPrefix(ctx, pfx, nil) {
			userIDs = append(userIDs, key[len(pfx):])
		}

		for _, userID := range userIDs {
			if disabled, err := s.enrollments.Has(ctx, join(userID, keyEnrollmentDisabled)); err != nil {
				return fmt.Errorf("checking for disablement for %s: %w", id, err)
			} else if disabled {
				// this enrollment ID is disabled, skip it
				continue
			}

			// loop through userauthenticate and finally token update message(s)
			// for the users
			for _, keyPfx := range []string{keyUserAuthenticate, keyUserAuthenticateDigest, keyUserTokenUpdate} {
				msg, err := getDecodeCheckIn(ctx, s.users, join(userID, keyPfx))
				if errors.Is(err, kv.ErrKeyNotFound) {
					// message not found, skip to the next
					continue
				} else if err != nil {
					return fmt.Errorf("getting check-in message for %s: %w", userID, err)
				}
				out <- msg
			}
		}
	}

	return nil
}
