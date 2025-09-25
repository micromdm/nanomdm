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

	// first test an Authenticate check-in message

	msgBytes, err := os.ReadFile("../../mdm/testdata/Authenticate.2.plist")
	if err != nil {
		t.Fatal(err)
	}

	msg, err := mdm.DecodeCheckin(msgBytes)
	if err != nil {
		t.Fatal(err)
	}

	a := msg.(*mdm.Authenticate)

	ctx := context.Background()

	r := mdm.NewRequestWithContext(ctx, nil)
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

	event, err := os.ReadFile("testdata/Authenticate.2.json")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bytes.TrimSpace(event), bytes.TrimSpace(reqBody)) {
		t.Error("submitted event is not equal to testdata")

		// os.WriteFile("testdata/output.Authenticate.2.json", reqBody, 0644)
	}

	// now test the command response event

	rawBytes, err := os.ReadFile("../../mdm/testdata/DeviceInformation.1.plist")
	if err != nil {
		t.Fatal(err)
	}

	cr, err := mdm.DecodeCommandResults(rawBytes)
	if err != nil {
		t.Fatal(err)
	}

	r = mdm.NewRequestWithContext(ctx, nil)
	// normally "resolved" but we're hardcoding here
	r.EnrollID = &mdm.EnrollID{
		ID:   cr.UDID,
		Type: mdm.Device,
	}

	_, err = w.CommandAndReportResults(r, cr)
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

	reqBody, err = io.ReadAll(c.lastRequest.Body)
	if err != nil {
		t.Error(err)
	}

	event, err = os.ReadFile("testdata/DeviceInformation.1.json")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(bytes.TrimSpace(event), bytes.TrimSpace(reqBody)) {
		t.Error("submitted event is not equal to testdata")

		os.WriteFile("testdata/output.DeviceInformation.1.json", reqBody, 0644)
	}
}
