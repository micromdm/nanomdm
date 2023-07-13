package nanopush

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
)

func TestPush(t *testing.T) {
	deviceToken := "c2732227a1d8021cfaf781d71fb2f908c61f5861079a00954a5453f1d0281433"
	pushMagic := "47250C9C-1B37-4381-98A9-0B8315A441C7"
	topic := "com.example.apns-topic"
	payload := []byte(`{"mdm":"` + pushMagic + `"}`)
	apnsID := "922D9F1F-B82E-B337-EDC9-DB4FC8527676"

	handler := http.NewServeMux()
	server := httptest.NewServer(handler)

	handler.HandleFunc("/3/device/", func(w http.ResponseWriter, r *http.Request) {
		expectURL := fmt.Sprintf("/3/device/%s", deviceToken)
		if have, want := r.URL.String(), expectURL; have != want {
			t.Errorf("url: have %q, want %q", have, want)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := body, payload; !bytes.Equal(have, want) {
			t.Errorf("body: have %q, want %q", string(have), string(want))
		}

		w.Header().Set("apns-id", apnsID)
	})

	prov := &Provider{
		baseURL: server.URL,
		client:  http.DefaultClient,
	}

	pushInfo := &mdm.Push{
		PushMagic: pushMagic,
		Topic:     topic,
	}
	pushInfo.SetTokenString(deviceToken)

	resp, err := prov.Push([]*mdm.Push{pushInfo})
	if err != nil {
		t.Fatal(err)
	}

	result, ok := resp[deviceToken]
	if !ok || result == nil {
		t.Fatal("device token not found (or is nil) in response")
	}

	if have, want := result.Id, apnsID; have != want {
		t.Errorf("url: have %q, want %q", have, want)
	}
}
