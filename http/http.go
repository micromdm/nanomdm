// Package http includes handlers and utilties
package http

import (
	"bytes"
	"crypto/subtle"
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

// BasicAuth is a simple HTTP plain authentication middleware.
func BasicAuth(next http.Handler, username, password, realm string) http.HandlerFunc {
	uBytes := []byte(username)
	pBytes := []byte(password)
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), uBytes) != 1 || subtle.ConstantTimeCompare([]byte(p), pBytes) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// VersionHandler returns a simple JSON response from a version string.
func VersionHandler(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"` + version + `"}`))
	}
}
