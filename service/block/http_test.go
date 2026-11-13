package block

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
)

const idleReport = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Status</key>
	<string>Idle</string>
	<key>UDID</key>
	<string>` + testUDID + `</string>
</dict>
</plist>`

// serveIdle sends an Idle command report for a device blocked with style
// through the MDM HTTP handler and returns the response.
func serveIdle(t *testing.T, style storage.BlockStyle) *http.Response {
	t.Helper()
	store := newMemStore()
	store.blocks[testUDID] = &storage.Block{Style: style}
	h := httpmdm.CheckinAndCommandHandler(New(&nextService{}, store), log.NopLogger)

	req := httptest.NewRequest(http.MethodPut, "/mdm", strings.NewReader(idleReport))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func TestHTTPUnrecognizedDevice(t *testing.T) {
	resp := serveIdle(t, storage.StyleUnrecognizedDevice)
	if have, want := resp.StatusCode, http.StatusForbidden; have != want {
		t.Errorf("status: have: %d, want: %d", have, want)
	}
	if have, want := resp.Header.Get("Content-Type"), "application/json"; have != want {
		t.Errorf("content type: have: %q, want: %q", have, want)
	}
	body, _ := io.ReadAll(resp.Body)
	if have, want := string(body), `{"code":"com.apple.unrecognized.device"}`; have != want {
		t.Errorf("body: have: %q, want: %q", have, want)
	}
}

func TestHTTPUnenroll(t *testing.T) {
	resp := serveIdle(t, storage.StyleUnenroll)
	if have, want := resp.StatusCode, http.StatusUnauthorized; have != want {
		t.Errorf("status: have: %d, want: %d", have, want)
	}
	// errors without a body are still answered with the status text
	if have := resp.Header.Get("Content-Type"); !strings.HasPrefix(have, "text/plain") {
		t.Errorf("content type: have: %q, want text/plain", have)
	}
}
