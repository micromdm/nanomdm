package kv

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/micromdm/nanolib/storage/kv"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

const (
	keyDeviceAuthenticate = "auth"
	keyDeviceCert         = "cert"
	keyDeviceSerial       = "serial"
	keyDeviceTokenUpdate  = "tok_upd"

	keyEnrollmentDisabled    = "disabled"
	keyEnrollmentUserChannel = "user_ch"
	keyEnrollmentUnlockToken = "unl_tok"
	keyEnrollmentTokenTally  = "tok_tal"
	keyEnrollmentType        = "type"
	keyEnrollmentEnrolledAt  = "enrolled_at"

	keyUserTokenUpdate   = keyDeviceTokenUpdate
	keyUserDeviceChannel = "device_ch"

	keyUserAuthenticate       = "user_auth"
	keyUserAuthenticateDigest = "user_auth_digest"

	valueDeviceDisabled = "1"

	valueEnrollmentUserChannelAssociated = "1"
)

// StoreAuthenticate stores the Authenticate check-in message from an enrollment.
func (s *KV) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	if r.ParentID != "" {
		return storage.ErrDeviceChannelOnly
	}

	// store device details
	err := kv.PerformCRUDBucketTxn(r.Context, s.devices, func(ctx context.Context, b kv.CRUDBucket) error {
		// write the raw authenticate message
		err := b.Set(ctx, join(r.ID, keyDeviceAuthenticate), msg.Raw)
		if err != nil {
			return err
		}

		// write our device identity certificate
		if r.Certificate != nil {
			err = b.Set(ctx, join(r.ID, keyDeviceCert), r.Certificate.Raw)
		} else {
			// clear any previous cert if it does not exist
			err = b.Delete(ctx, join(r.ID, keyDeviceCert))
		}
		if err != nil {
			return err
		}

		// write the serial number
		if msg.SerialNumber != "" {
			err = b.Set(ctx, join(r.ID, keyDeviceSerial), []byte(msg.SerialNumber))
		} else {
			// clear any previous serial if it is not set
			err = b.Delete(ctx, join(r.ID, keyDeviceSerial))
		}
		if err != nil {
			return err
		}

		// clear the device bootstrap token
		err = b.Delete(r.Context, join(r.ID, keyBootstrapToken))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// store enrollment details
	return kv.PerformBucketTxn(r.Context, s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		// delete per-enrollment one-time set keys
		err := kv.DeleteSlice(ctx, b, []string{
			join(r.ID, keyEnrollmentUnlockToken),
			join(r.ID, keyEnrollmentTokenTally),
			join(r.ID, keyEnrollmentEnrolledAt),
		})
		if err != nil {
			return err
		}

		// then loop through any user channels belonging to this enrollment
		// to delete the token tallys and disable enrollment
		for _, id := range userChannelEnrollments(ctx, r.ID, b) {
			err := b.Delete(ctx, join(id, keyEnrollmentTokenTally))
			if err != nil {
				return err
			}
		}

		return nil
	})
	// note that the NanoMDM service should be calling Disable after the Authenticate message.
}

// bumpTally increases the token tally by one and returns it.
// Note: this is a get-then-set operation: the device performing a token
// update syncronousely prevents race conditions, but that's all.
func bumpTally(ctx context.Context, id string, b kv.CRUDBucket) (int, error) {
	tallyBytes, err := b.Get(ctx, join(id, keyEnrollmentTokenTally))
	if err != nil && !errors.Is(err, kv.ErrKeyNotFound) {
		return 0, err
	}
	tally, _ := strconv.Atoi(string(tallyBytes))
	tally += 1
	return tally, b.Set(ctx, join(id, keyEnrollmentTokenTally), []byte(strconv.Itoa(tally)))
}

// StoreTokenUpdate stores the TokenUpdate check-in message.
// Storing this first TokenUpdate message represents a successful enrollment.
// Note both device and user channel enrollments receive TokenUpdate messages.
func (s *KV) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	// write the raw token update message
	// pick a bucket to write to depending on user- or device-channel enrollment
	var tokUpdBkt kv.CRUDBucketTxnBeginner = s.devices
	tokUpdKeyName := keyDeviceTokenUpdate
	if r.ParentID != "" {
		tokUpdBkt = s.users
		tokUpdKeyName = keyUserTokenUpdate
	}
	err := kv.PerformCRUDBucketTxn(r.Context, tokUpdBkt, func(ctx context.Context, b kv.CRUDBucket) error {
		if r.ParentID != "" {
			// associate our parent device-channel if this is a user-channel enrollment
			err := b.Set(ctx, join(r.ID, keyUserDeviceChannel), []byte(r.ParentID))
			if err != nil {
				return err
			}
		}
		return b.Set(ctx, join(r.ID, tokUpdKeyName), msg.Raw)
	})
	if err != nil {
		return err
	}

	return kv.PerformCRUDBucketTxn(r.Context, s.enrollments, func(ctx context.Context, b kv.CRUDBucket) error {
		if err := s.updateLastSeen(r, b); err != nil {
			return fmt.Errorf("updating last seen: %s", err)
		}

		// write the device type
		err := b.Set(ctx, join(r.ID, keyEnrollmentType), []byte(r.Type.String()))
		if err != nil {
			return err
		}

		// store the push info details
		err = storePushInfo(ctx, b, r.ID, msg)
		if err != nil {
			return err
		}

		// write the unlock token. future TokenUpdate messages may not include it.
		if len(msg.UnlockToken) > 0 {
			err = b.Set(ctx, join(r.ID, keyEnrollmentUnlockToken), msg.UnlockToken)
			if err != nil {
				return err
			}
		}

		// associate the user channel if we can
		if r.ParentID != "" {
			if err = assocUserChannel(ctx, b, r.ID, r.ParentID); err != nil {
				return err
			}
		}

		// bump our token update tally
		tally, err := bumpTally(ctx, r.ID, b)
		if err != nil {
			return err
		}

		if tally == 1 {
			// update our enrolled time
			err = b.Set(ctx, join(r.ID, keyEnrollmentEnrolledAt), timeFmt(time.Now()))
			if err != nil {
				return err
			}
		}

		// enable the enrollment
		return b.Delete(ctx, join(r.ID, keyEnrollmentDisabled))
	})
}

// assocUserChannel associates a user channel id with parent ID.
func assocUserChannel(ctx context.Context, b kv.RWBucket, id, parentID string) error {
	return b.Set(ctx, join(parentID, keyEnrollmentUserChannel, id), []byte(valueEnrollmentUserChannelAssociated))
}

// userChannelEnrollments retrieves the list of user channel enrollments belonging the parent id.
func userChannelEnrollments(ctx context.Context, id string, b kv.Bucket) []string {
	pfx := join(id, keyEnrollmentUserChannel) + keySep
	pfxLen := len(pfx)
	var ids []string
	for key := range b.KeysPrefix(ctx, pfx, nil) {
		ids = append(ids, key[0:pfxLen])
	}
	return ids
}

// Disable the MDM enrollment.
// If r is from a device channel then any user channels for this device
// also need to be disabled.
func (s *KV) Disable(r *mdm.Request) error {
	return kv.PerformBucketTxn(r.Context, s.enrollments, func(ctx context.Context, b kv.Bucket) error {
		// the Disable method is called from the NanoMDM service
		// for both Authenticate and CheckOut check-in messages.
		if err := s.updateLastSeen(r, b); err != nil {
			return fmt.Errorf("updating last seen: %s", err)
		}

		// first disable the enrollment we were asked to
		// by setting a disabled key for the id in the enrollments bucket
		err := b.Set(ctx, join(r.ID, keyEnrollmentDisabled), []byte(valueDeviceDisabled))
		if err != nil {
			return err
		}

		if r.ParentID != "" {
			// this is a user channel, our work is done
			return nil
		}

		// then loop through any user channels belonging to this enrollment
		for _, id := range userChannelEnrollments(ctx, r.ID, b) {
			err := b.Set(ctx, join(id, keyEnrollmentDisabled), []byte(valueDeviceDisabled))
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// getTally retrieves the token update tally.
func getTally(ctx context.Context, b kv.ROBucket, id string) (int, error) {
	tallyBytes, err := b.Get(ctx, join(id, keyEnrollmentTokenTally))
	if errors.Is(err, kv.ErrKeyNotFound) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	tally, _ := strconv.Atoi(string(tallyBytes))
	return tally, err
}

// StoreUserAuthenticate stores the UserAuthenticate check-in message from an enrollment.
func (s *KV) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	if err := s.updateLastSeen(r, nil); err != nil {
		return fmt.Errorf("updating last seen: %s", err)
	}

	// write the raw UserAuthenticate message to the users store
	key := keyUserAuthenticate
	if msg.DigestResponse != "" {
		key = keyUserAuthenticateDigest
	}
	err := s.users.Set(r.Context, join(r.ID, key), msg.Raw)
	if err != nil {
		return err
	}

	// disable the enrollment (not valid until after TokenUpdate)
	return s.enrollments.Set(r.Context, join(r.ID, keyEnrollmentDisabled), []byte(valueEnrollmentUserChannelAssociated))
}

// RetrieveTokenUpdateTally retrieves the TokenUpdate tally (count) for id.
// If no tally exists or is not yet set 0 with a nil error should be returned.
func (s *KV) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	return getTally(ctx, s.enrollments, id)
}
