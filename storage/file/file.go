// Package file implements filesystem-based storage for MDM services
package file

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	"github.com/jessepeterson/nanomdm/cryptoutil"
	"github.com/jessepeterson/nanomdm/mdm"
)

const (
	AuthenticateFilename = "Authenticate.plist"
	TokenUpdateFilename  = "TokenUpdate.plist"
	UnlockTokenFilename  = "UnlockToken.dat"
	SerialNumberFilename = "SerialNumber.txt"
	IdentityCertFilename = "Identity.pem"

	CertAuthFilename             = "CertAuth.sha256.txt"
	CertAuthAssociationsFilename = "CertAuth.txt"
)

// FileStorage implements filesystem-based storage for MDM services
type FileStorage struct {
	path string
}

// New creates a new FileStorage backend
func New(path string) (*FileStorage, error) {
	err := os.Mkdir(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	return &FileStorage{path: path}, nil
}

type enrollment struct {
	id string
	fs *FileStorage
}

func (s *FileStorage) newEnrollment(id string) *enrollment {
	return &enrollment{fs: s, id: id}
}

func (e *enrollment) dir() string {
	return path.Join(e.fs.path, e.id)
}

func (e *enrollment) mkdir() error {
	return os.MkdirAll(e.dir(), 0755)
}

func (e *enrollment) dirPrefix(name string) string {
	return path.Join(e.dir(), name)
}

func (e *enrollment) writeFile(name string, bytes []byte) error {
	if name == "" {
		return errors.New("write: empty name")
	}
	if err := e.mkdir(); err != nil {
		return err
	}
	return ioutil.WriteFile(e.dirPrefix(name), bytes, 0755)
}

func (e *enrollment) readFile(name string) ([]byte, error) {
	if name == "" {
		return nil, errors.New("write: empty name")
	}
	return ioutil.ReadFile(e.dirPrefix(name))
}

// StoreAuthenticate stores the Authenticate message
func (s *FileStorage) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	e := s.newEnrollment(r.ID)
	if r.Certificate != nil {
		if err := e.writeFile(IdentityCertFilename, cryptoutil.PEMCertificate(r.Certificate.Raw)); err != nil {
			return err
		}
	}
	// A nice-to-have even though it's duplicated in msg
	if msg.SerialNumber != "" {
		err := e.writeFile(SerialNumberFilename, []byte(msg.SerialNumber))
		if err != nil {
			return err
		}
	}
	return e.writeFile(AuthenticateFilename, []byte(msg.Raw))
}

// StoreTokenUpdate stores the TokenUpdate message
func (s *FileStorage) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	e := s.newEnrollment(r.ID)
	// the UnlockToken should be saved separately in case future
	// TokenUpdates do not contain it and it gets overwritten
	if len(msg.UnlockToken) > 0 {
		if err := e.writeFile(UnlockTokenFilename, msg.UnlockToken); err != nil {
			return err
		}
	}
	return e.writeFile(TokenUpdateFilename, []byte(msg.Raw))
}
