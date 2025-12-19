package mysql

import (
	"context"

	"github.com/micromdm/nanomdm/mdm"
)

func (s *MySQLStorage) RetrieveMigrationCheckins(ctx context.Context, c chan<- interface{}) error {
	// TODO: if a TokenUpdate does not include the latest UnlockToken
	// then we should synthesize a TokenUpdate to transfer it over.
	deviceRows, err := s.q.RetrieveMigrationCheckinsDevices(ctx)
	if err != nil {
		return err
	}
	for _, deviceRow := range deviceRows {
		var authBytes, tokenBytes []byte

		authBytes = []byte(deviceRow.Authenticate)
		if deviceRow.TokenUpdate.Valid {
			tokenBytes = []byte(deviceRow.TokenUpdate.String)
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
	userRows, err := s.q.RetrieveMigrationCheckinsUsers(ctx)
	if err != nil {
		return err
	}

	for _, token := range userRows {
		if !token.Valid {
			continue
		}

		tokenBytes := []byte(token.String)

		msg, err := mdm.DecodeCheckin(tokenBytes)
		if err != nil {
			c <- err
		} else {
			c <- msg
		}
	}
	return nil
}
