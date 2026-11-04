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

func TestDMHookEndpoint(t *testing.T) {
	for _, tc := range []struct {
		prefix   string
		endpoint string
		want     string // empty if an error is expected
	}{
		{"http://dm.example.com/dm/", "tokens", "http://dm.example.com/dm/tokens"},
		{"http://dm.example.com/dm/", "declaration-items", "http://dm.example.com/dm/declaration-items"},
		{"http://dm.example.com/dm/", "status", "http://dm.example.com/dm/status"},
		{"http://dm.example.com/dm/", "declaration/configuration/com.example.test", "http://dm.example.com/dm/declaration/configuration/com.example.test"},
		{"http://dm.example.com/dm/", "declaration/asset/0A7E4B4C-5F1F-4A4E-9C1A-7F0F5D2B7B11", "http://dm.example.com/dm/declaration/asset/0A7E4B4C-5F1F-4A4E-9C1A-7F0F5D2B7B11"},
		// slashes are passed through as path separators
		{"http://dm.example.com/dm/", "declaration/foo/bar", "http://dm.example.com/dm/declaration/foo/bar"},
		{"http://dm.example.com/dm/", "declaration/configuration/a/b", "http://dm.example.com/dm/declaration/configuration/a/b"},
		// the endpoint is unescaped: other characters are escaped, not interpreted
		{"http://dm.example.com/dm/", "tokens?a=b", "http://dm.example.com/dm/tokens%3Fa=b"},
		{"http://dm.example.com/dm/", "declaration/configuration/a#b", "http://dm.example.com/dm/declaration/configuration/a%23b"},
		{"http://dm.example.com/dm/", "?", "http://dm.example.com/dm/%3F"},
		{"http://dm.example.com/dm/", "declaration/configuration/a b", "http://dm.example.com/dm/declaration/configuration/a%20b"},
		{"http://dm.example.com/dm/", "declaration/configuration/caf\u00e9", "http://dm.example.com/dm/declaration/configuration/caf%C3%A9"},
		{"http://dm.example.com/dm/", "declaration/configuration/50%off", "http://dm.example.com/dm/declaration/configuration/50%25off"},
		{"http://dm.example.com/dm/", "declaration/configuration/a%20b", "http://dm.example.com/dm/declaration/configuration/a%2520b"},
		{"http://dm.example.com/dm/", "declaration/configuration/a%2Fb", "http://dm.example.com/dm/declaration/configuration/a%252Fb"},
		{"http://dm.example.com/dm/", "..%2f..%2fx", "http://dm.example.com/dm/..%252f..%252fx"},
		{"http://dm.example.com/dm/", "declaration/%2e%2e/%2e%2e/x", "http://dm.example.com/dm/declaration/%252e%252e/%252e%252e/x"},
		{"http://dm.example.com/dm/", "..%2f..%2fadmin?", "http://dm.example.com/dm/..%252f..%252fadmin%3F"},
		// URL-looking endpoints are just path segments
		{"http://dm.example.com/dm/", "http:x", "http://dm.example.com/dm/http:x"},
		{"http://dm.example.com/dm/", "mailto:a@example.com", "http://dm.example.com/dm/mailto:a@example.com"},
		{"http://dm.example.com/dm/", "http://internal.example.com/x", ""},
		{"http://dm.example.com/dm/", "//internal.example.com/x", ""},
		{"http://dm.example.com/dm/", "//user@internal.example.com/x", ""},
		{"http://dm.example.com/dm/", "///x", ""},
		// empty, ".", and ".." segments are rejected
		{"http://dm.example.com/dm/", "/abs", ""},
		{"http://dm.example.com/dm/", "/dm/tokens", ""},
		{"http://dm.example.com/dm/", "", ""},
		{"http://dm.example.com/dm/", "tokens/", ""},
		{"http://dm.example.com/dm/", "declaration//x", ""},
		{"http://dm.example.com/dm/", "./tokens", ""},
		{"http://dm.example.com/dm/", "../outside-prefix", ""},
		{"http://dm.example.com/dm/", "declaration/../../x", ""},
		// the prefix query and fragment are dropped
		{"http://dm.example.com/dm/?k=v#f", "tokens", "http://dm.example.com/dm/tokens"},
		// the prefix userinfo is kept
		{"http://u:p@dm.example.com/d%20m/", "tokens", "http://u:p@dm.example.com/d%20m/tokens"},
		// without a trailing slash the last path element is replaced
		{"http://dm.example.com/dm", "tokens", "http://dm.example.com/tokens"},
		{"http://dm.example.com/a/dm", "tokens", "http://dm.example.com/a/tokens"},
		{"http://dm.example.com/a/dm", "../tokens", ""},
		{"http://dm.example.com", "tokens", "http://dm.example.com/tokens"},
		{"http://dm.example.com", "//internal.example.com/x", ""},
	} {
		t.Run(tc.prefix+" "+tc.endpoint, func(t *testing.T) {
			c := &mockDoer{nextResponse: makeResp(nil, nil, "", 200)}

			s, err := New(tc.prefix, WithClient(c))
			if err != nil {
				t.Fatal(err)
			}

			r := mdm.NewRequestWithContext(context.Background(), nil)
			r.EnrollID = &mdm.EnrollID{ID: "ID", Type: mdm.Device}

			_, err = s.DeclarativeManagement(r, &mdm.DeclarativeManagement{Endpoint: tc.endpoint})
			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected error, request sent to: %s", c.lastRequest.URL)
				}
				if c.lastRequest != nil {
					t.Errorf("request should not have been sent: %s", c.lastRequest.URL)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if have, want := c.lastRequest.URL.String(), tc.want; have != want {
				t.Errorf("URL mismatch: have: %q, want: %q", have, want)
			}
		})
	}
}
