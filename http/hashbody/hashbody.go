package hashbody

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
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

	err := GetAndReplaceBody(req, hasher)
	if err != nil {
		return "", fmt.Errorf("getting body: %w", err)
	}

	encoded := encoder(hasher.Sum(nil))

	req.Header.Set(header, encoded)

	return encoded, nil
}

// GetAndReplaceBody returns the body of req as a bytes slice.
// If needed the body is replaced with a byte buffer for re-use.
func GetAndReplaceBodyBytes(req *http.Request) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := GetAndReplaceBody(req, buf)
	return buf.Bytes(), err
}

// GetAndReplaceBody writes the body of req to w.
// If needed the body is replaced with a byte buffer for re-use.
func GetAndReplaceBody(req *http.Request, w io.Writer) error {
	if req == nil {
		return errors.New("nil request")
	}
	if req.Body == nil || w == nil {
		return nil
	}

	var err error

	body := req.Body
	var buf *bytes.Buffer

	if req.GetBody != nil {
		// GetBody returns a copy of a the body for reading
		// using it implies we don't need to replace the body
		body, err = req.GetBody()
		if err != nil {
			return fmt.Errorf("getting body: %w", err)
		} else if body == nil {
			return nil
		}
	} else {
		buf = new(bytes.Buffer)
		w = io.MultiWriter(buf, w)
	}

	_, err = io.Copy(w, body)
	if err != nil {
		return fmt.Errorf("copying body: %w", err)
	}
	defer body.Close()

	if buf != nil {
		req.Body = io.NopCloser(buf)
	}

	return nil
}
