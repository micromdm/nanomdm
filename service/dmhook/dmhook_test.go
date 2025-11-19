package dmhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
)

type mockDoer struct {
	lastRequest  *http.Request
	nextResponse *http.Response
	nextError    error
}

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	m.lastRequest = r
	return m.nextResponse, m.nextError
}

func makeResp(body, key []byte, header string, code int) *http.Response {
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewBuffer(body)),
		Header:     make(http.Header),
		StatusCode: code,
	}
	resp.Status = fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode))

	if len(key) > 0 {
		h := hmac.New(sha256.New, key)
		h.Write(body)
		resp.Header.Set(header, base64.StdEncoding.EncodeToString(h.Sum(nil)))
	}

	return resp
}

func verifyHTTPReq(req *http.Request, key []byte, header string) (bool, []byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(req.Header.Get(header))
	if err != nil {
		return false, nil, err
	}

	var buf bytes.Buffer

	h := hmac.New(sha256.New, key)
	m := io.MultiWriter(h, &buf)
	_, err = io.Copy(m, req.Body)
	if err != nil {
		return false, nil, err
	}

	return bytes.Equal(decoded, h.Sum(nil)), buf.Bytes(), nil
}

func TestDMHook(t *testing.T) {
	c := &mockDoer{}

	s, err := New("", WithClient(c), WithSetHMACSecret([]byte("pwOut")), WithVerifyHMACSecret([]byte("pwIn")))
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	r := mdm.NewRequestWithContext(ctx, nil)
	r.EnrollID = &mdm.EnrollID{
		ID:   "ID",
		Type: mdm.Device,
	}

	m := &mdm.DeclarativeManagement{
		Endpoint: "foo",
		Enrollment: mdm.Enrollment{
			UDID: "ID",
		},
		Data: []byte(`{"baz":"foo"}`),
	}

	respBytes := []byte(`{"foo":"bar"}`)

	// generate a response with an HMAC for validation against the WithVerifyHMACSecret() option
	// example of what might come come from a DM protocol server
	// our mockDoer will hand this back to the DMHook
	c.nextResponse = makeResp(respBytes, []byte("pwIn"), HMACHeader, 200)

	ret, err := s.DeclarativeManagement(r, m)
	if err != nil {
		t.Fatal(err)
	}

	// make sure the DDM path was appended
	if have, want := c.lastRequest.URL.Path, "/foo"; have != want {
		t.Errorf("URL path mismatch: have: %q, want: %q", have, want)
	}

	// compare our mock "server" response with the service response
	if have, want := ret, respBytes; !bytes.Equal(have, want) {
		t.Errorf("body mismatch: have: %s, want: %s", have, want)
	}

	// verify the generated HMAC from the DMHook against the WithSetHMACSecret() option
	valid, reqBytes, err := verifyHTTPReq(c.lastRequest, []byte("pwOut"), HMACHeader)
	if err != nil {
		t.Fatal(err)
	}

	// valid HMAC
	if have, want := valid, true; have != want {
		t.Errorf("hash invalid: have: %t, want: %t", have, want)
	}

	// compare the Data key which should be used as the body of the request against what it sould be
	if have, want := reqBytes, m.Data; !bytes.Equal(have, want) {
		t.Errorf("request body mismatch: have: %s, want: %s", have, want)
	}

	// generate a response with an invalid HMAC to make sure an error is generated
	c.nextResponse = makeResp(respBytes, []byte("pwInvalid"), HMACHeader, 200)

	_, err = s.DeclarativeManagement(r, m)
	if err == nil {
		t.Fatal("should have errored for invalid hash")
	}

}
