package hashbody

import (
	"bytes"
	"context"
	"hash"
	"hash/fnv"
	"io"
	"net/http"
	"testing"
)

func testHashAndVerify(t *testing.T, ctx context.Context, body []byte, header string, hasher hash.Hash) {
	req, err := http.NewRequestWithContext(ctx, "GET", "", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	_, err = SetBodyHashHeader(req, header, hasher, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp := &http.Response{Body: io.NopCloser(bytes.NewBuffer(body)), Header: make(http.Header)}
	resp.Header.Set(header, req.Header.Get(header))

	var buf2 bytes.Buffer

	valid, err := VerifyBodyHashHeader(resp, header, hasher, nil, &buf2)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := valid, true; have != want {
		t.Errorf("hash invalid: have: %t, want: %t", have, want)
	}

	if have, want := buf2.Bytes(), body; !bytes.Equal(have, want) {
		t.Errorf("body mismatch: have: %s, want: %s", have, want)
	}
}

func TestHashBody(t *testing.T) {
	ctx := context.Background()
	for _, test := range []struct {
		name   string
		body   []byte
		header string
		hasher hash.Hash
	}{
		{
			name:   "nil hasher",
			header: "X-Hash",
			body:   []byte("hello, world!"),
		},
		{
			name:   "nil body",
			header: "X-Hash",
			body:   nil,
		},
		{
			name:   "fnv",
			header: "X-Hash",
			body:   []byte("hello, world!"),
			hasher: fnv.New128(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			testHashAndVerify(t, ctx, test.body, test.header, test.hasher)
		})
	}
}
