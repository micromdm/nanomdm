package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage/mysql/sqlc"
)

// StoreAuthenticate upserts a device Authenticate message which kicks off an enrollment.
func (s *MySQLStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	// no enrollment record potentially yet exists, so no last seen update.

	var pemCert []byte
	if r.Certificate != nil {
		// a nil certificate is likely a migrated enrollment.
		pemCert = cryptoutil.PEMCertificate(r.Certificate.Raw)
	}

	return s.q.UpsertAuthenticate(r.Context(), sqlc.UpsertAuthenticateParams{
		ID:           r.ID,
		IdentityCert: pemCert,
		SerialNumber: nullEmptyString(msg.SerialNumber),
		Authenticate: msg.Raw,
	})
}

// StoreTokenUpdate updates the token update message and upserts the enrollment record.
func (s *MySQLStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	// last seen is upserted within the enrollment record update

	return s.txn.Exec(r.Context(), func(ctx context.Context, tx *sql.Tx, qtx *sqlc.Queries) error {
		var deviceID, userID string
		if r.ParentID == "" {
			deviceID = r.ID
			if len(msg.UnlockToken) < 1 {
				params := sqlc.UpdateDeviceTokenUpdateParams{
					TokenUpdate: msg.Raw,
					ID:          r.ID,
				}
				if err := qtx.UpdateDeviceTokenUpdate(ctx, params); err != nil {
					return fmt.Errorf("storing device token update: %w", err)
				}
			} else {
				params := sqlc.UpdateDeviceTokenUpdateWithUnlockParams{
					TokenUpdate: msg.Raw,
					UnlockToken: msg.UnlockToken,
					ID:          r.ID,
				}
				if err := qtx.UpdateDeviceTokenUpdateWithUnlock(ctx, params); err != nil {
					return fmt.Errorf("storing device token update with unlock: %w", err)
				}
			}
		} else {
			deviceID = r.ParentID
			userID = r.ID
			params := sqlc.UpsertUserTokenUpdateParams{
				ID:            r.ID,
				DeviceID:      r.ParentID,
				UserShortName: nullEmptyString(msg.UserShortName),
				UserLongName:  nullEmptyString(msg.UserLongName),
				TokenUpdate:   msg.Raw,
			}
			if err := qtx.UpsertUserTokenUpdate(ctx, params); err != nil {
				return fmt.Errorf("storing user token update: %w", err)
			}
		}
		params := sqlc.UpsertEnrollmentParams{
			ID:        r.ID,
			DeviceID:  deviceID,
			UserID:    nullEmptyString(userID),
			Type:      r.Type.String(),
			Topic:     msg.Topic,
			PushMagic: msg.PushMagic,
			TokenHex:  msg.Token.String(),
		}
		if err := qtx.UpsertEnrollment(ctx, params); err != nil {
			return fmt.Errorf("storing enrollment: %w", err)
		}
		return nil
	})
}

// StoreUserAuthenticate upserts the user authenticate message or digest.
func (s *MySQLStorage) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	defer s.updateLastSeen(r)

	// if the DigestResponse is empty then this is the first (of two)
	// UserAuthenticate messages depending on our response
	if msg.DigestResponse == "" {
		params := sqlc.UpsertUserAuthenticateParams{
			ID:               r.ID,
			DeviceID:         r.ParentID,
			UserShortName:    nullEmptyString(msg.UserShortName),
			UserLongName:     nullEmptyString(msg.UserLongName),
			UserAuthenticate: msg.Raw,
		}
		if err := s.q.UpsertUserAuthenticate(r.Context(), params); err != nil {
			return fmt.Errorf("storing user authenticate: %w", err)
		}
	} else {
		params := sqlc.UpsertUserAuthenticateDigestParams{
			ID:                     r.ID,
			DeviceID:               r.ParentID,
			UserShortName:          nullEmptyString(msg.UserShortName),
			UserLongName:           nullEmptyString(msg.UserLongName),
			UserAuthenticateDigest: msg.Raw,
		}
		if err := s.q.UpsertUserAuthenticateDigest(r.Context(), params); err != nil {
			return fmt.Errorf("storing user authenticate digest: %w", err)
		}
	}
	return nil
}

// Disable should only be called from an Authenticate or CheckOut message.
// It assumes it is via communication from the MDM device and updates the last seen timestamp.
func (s *MySQLStorage) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable the device channel")
	}
	return s.q.UpdateDisableEnrollment(r.Context(), r.ID)
}

// RetrieveTokenUpdateTally selects the token update tally for id.
func (s *MySQLStorage) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	dbTally, err := s.q.SelectTokenUpdateTally(ctx, id)
	return int(dbTally), err
}
