package file

import (
	"context"
	"os"

	"github.com/micromdm/nanomdm/mdm"
)

func sendCheckinMessage(e *enrollment, filename string, c chan<- interface{}) {
	msgBytes, err := e.readFile(filename)
	if err != nil {
		c <- err
		return
	}
	msg, err := mdm.DecodeCheckin(msgBytes)
	if err != nil {
		c <- err
		return
	}
	c <- msg
}

func (s *FileStorage) RetrieveMigrationCheckins(_ context.Context, c chan<- interface{}) error {
	for _, userLoop := range []bool{false, true} {
		entries, err := os.ReadDir(s.path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			e := s.newEnrollment(entry.Name())
			authExists, err := e.fileExists(AuthenticateFilename)
			if err != nil {
				c <- err
			}
			// an Authenticate should not exist for a user-channel
			// enrollment. skip it if found.
			if userLoop && authExists {
				continue
			} else if !userLoop && authExists {
				sendCheckinMessage(e, AuthenticateFilename, c)
			}
			tokExists, err := e.fileExists(TokenUpdateFilename)
			if err != nil {
				c <- err
			}
			// if neither an authenticate nor tokenupdate exists then
			// this is an invalid enrollment and we should skip it
			if !tokExists && !authExists {
				continue
			}
			// TODO: if we have an UnlockToken for a device we
			// should synthesize it into a TokenUpdate message because
			// they are saved out-of-band.
			sendCheckinMessage(e, TokenUpdateFilename, c)
		}
	}
	return nil
}
