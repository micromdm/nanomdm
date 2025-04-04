// Package http includes handlers and utilties
package http

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

// ReadAllAndReplaceBody reads all of r.Body and replaces it with a new byte buffer.
func ReadAllAndReplaceBody(r *http.Request) ([]byte, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return b, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}
