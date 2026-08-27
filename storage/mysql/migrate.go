package mysql

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

// RetrieveMigrationCheckins streams device and user check-in messages into c.
func (s *MySQLStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {
	// this uses raw db.QueryContext rather than sqlc because it streams rows into
	// the channel; sqlc has no streaming mode (its :many buffers all rows).

	// TODO: if a TokenUpdate does not include the latest UnlockToken
	// then we should synthesize a TokenUpdate to transfer it over.
	deviceRows, err := s.db.QueryContext(
		ctx,
		`SELECT authenticate, token_update FROM devices;`,
	)
	if err != nil {
		return err
	}
	defer deviceRows.Close()
	for deviceRows.Next() {
		var authBytes, tokenBytes []byte
		if err := deviceRows.Scan(&authBytes, &tokenBytes); err != nil {
			return err
		}
		for _, msgBytes := range [][]byte{authBytes, tokenBytes} {
			msg, err := mdm.DecodeCheckin(msgBytes)
			if err != nil {
				c <- err
			} else {
				c <- msg
			}
		}
	}
	if err = deviceRows.Err(); err != nil {
		return err
	}
	userRows, err := s.db.QueryContext(
		ctx,
		`SELECT token_update FROM users;`,
	)
	if err != nil {
		return err
	}
	defer userRows.Close()
	for userRows.Next() {
		var msgBytes []byte
		if err := userRows.Scan(&msgBytes); err != nil {
			return err
		}
		msg, err := mdm.DecodeCheckin(msgBytes)
		if err != nil {
			c <- err
		} else {
			c <- msg
		}
	}
	if err = userRows.Err(); err != nil {
		return err
	}
	return nil
}
