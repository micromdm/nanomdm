package hashbody

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"

	libhttp "github.com/micromdm/nanolib/http"
)

// SetBodyHashHeader reads the body of req and writes it to hasher to encode an HTTP header.
// If needed the body is replaced with a byte buffer for re-use.
// If hasher is nil a default SHA-256 hasher is used.
// If encoder is nil a default hex encoder is used.
// The final encoded string set in the HTTP header is returned.
func SetBodyHashHeader(req *http.Request, header string, hasher hash.Hash, encoder func([]byte) string) (string, error) {
	if req == nil {
		return "", errors.New("nil request")
	}
	if header == "" {
		return "", errors.New("empty header")
	}

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

// VerifyBodyHashHeader verifies the hasher of the resp HTTP body against the decoder header and optionally writes it back out to w.
// True is returned if the hashes match.
// The resp.Body is read and the caller is responsible for closing it.
// If hasher is nil a default SHA-256 hasher is used.
// If decoder is nil a default hex decoder is used.
func VerifyBodyHashHeader(resp *http.Response, header string, hasher hash.Hash, decoder func(string) ([]byte, error), w io.Writer) (bool, error) {
	if resp == nil {
		return false, errors.New("nil response")
	}
	if header == "" {
		return false, errors.New("empty header")
	}

	if decoder == nil {
		decoder = hex.DecodeString
	}

	decoded, err := decoder(resp.Header.Get(header))
	if err != nil {
		return false, fmt.Errorf("decoding %s header: %w", header, err)
	}

	if hasher == nil {
		hasher = sha256.New()
	} else {
		hasher.Reset()
	}

	if w == nil {
		w = hasher
	} else {
		w = io.MultiWriter(w, hasher)
	}

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return false, fmt.Errorf("copying body: %w", err)
	}

	bodyHash := hasher.Sum(nil)

	return subtle.ConstantTimeCompare(bodyHash, decoded) == 1, nil
}
