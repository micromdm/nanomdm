package hashbody

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"

	libhttp "github.com/micromdm/nanolib/http"
)

// SetBodyHashHeader reads the body of req and writes it to hasher to encode an HTTP header.
// If needed the body is replaced with a byte buffer for re-use.
// If hasher is nil a default SHA-256 hasher is used.
// If encoder is nil a default hex encoder is used.
// The final encoded string set in the HTTP header is returned.
func SetBodyHashHeader(req *http.Request, header string, hasher hash.Hash, encoder func([]byte) string) (string, error) {
	if hasher == nil {
		hasher = sha256.New()
	} else {
		hasher.Reset()
	}

	if encoder == nil {
		encoder = hex.EncodeToString
	}

	if err := libhttp.GetAndReplaceBody(req, hasher); err != nil {
		return "", fmt.Errorf("getting body: %w", err)
	}

	encoded := encoder(hasher.Sum(nil))

	req.Header.Set(header, encoded)

	return encoded, nil
}
