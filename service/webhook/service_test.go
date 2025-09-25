package webhook

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/micromdm/nanomdm/mdm"
)

func checkinFromTestData(name string) (interface{}, error) {
	msg, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}

	return mdm.DecodeCheckin(msg)
}

type mockDoer struct {
	lastRequest *http.Request
}

func (m *mockDoer) Do(r *http.Request) (*http.Response, error) {
	m.lastRequest = r
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBuffer([]byte{})),
	}
	return resp, nil
}

func TestWebhook(t *testing.T) {
	w := New("", nil)

	// override timestamp generation
	w.nowFn = func() time.Time { return time.Time{} }

	c := &mockDoer{}

	// override the internal client with our mocked edition
	w.doer = c

	msg, err := checkinFromTestData("../../mdm/testdata/Authenticate.2.plist")
	if err != nil {
		t.Fatal(err)
	}

	a := msg.(*mdm.Authenticate)

	r := mdm.NewRequestWithContext(context.Background(), nil)
	// normally "resolved" but we're hardcoding here
	r.EnrollID = &mdm.EnrollID{
		ID:   a.UDID,
		Type: mdm.Device,
	}

	err = w.Authenticate(r, a)
	if err != nil {
		t.Error(err)
	}

	if c.lastRequest == nil {
		t.Fatal("no HTTP request made")
	}

	if want, have := http.MethodPost, c.lastRequest.Method; want != have {
		t.Errorf("want: %v, have: %v", want, have)
	}

	if want, have := "application/json; charset=utf-8", c.lastRequest.Header.Get("Content-Type"); want != have {
		t.Errorf("want: %v, have: %v", want, have)
	}

	reqBody, err := io.ReadAll(c.lastRequest.Body)
	if err != nil {
		t.Error(err)
	}

	event, err := os.ReadFile("testdata/event1.json")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bytes.TrimSpace(event), bytes.TrimSpace(reqBody)) {
		t.Error("submitted event is not equal to testdata")

		// os.WriteFile("testdata/output.json", reqBody, 0644)
	}
}
