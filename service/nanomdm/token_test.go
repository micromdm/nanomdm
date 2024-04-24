package nanomdm

import (
	"bytes"
	"context"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
)

func newTokenMDMReq() *mdm.Request {
	return &mdm.Request{Context: context.Background()}
}

func newGetToken(serviceType string, id string) *mdm.GetToken {
	return &mdm.GetToken{
		TokenServiceType: serviceType,
		Enrollment:       mdm.Enrollment{UDID: id},
	}
}

func TestToken(t *testing.T) {
	tokenTestData := []byte("hello")

	// create muxer
	m := NewTokenMux()

	// associate a new static token handler with a type
	m.Handle("com.apple.maid", NewStaticToken(tokenTestData))

	// create a new NanoMDM service
	s := New(nil, WithGetToken(m))

	// dispatch a GetToken check-in message
	resp, err := s.GetToken(newTokenMDMReq(), newGetToken("com.apple.maid", "AAAA-1111"))
	if err != nil {
		t.Fatal(err)
	}

	// check that our token data our matches (from the static handler)
	if !bytes.Equal(tokenTestData, resp.TokenData) {
		t.Error("input and output not equal")
	}

	// supply an invalid service type (not handled) and expect an error
	_, err = s.GetToken(newTokenMDMReq(), newGetToken("com.apple.does-not-exist", "AAAA-1111"))
	if err == nil {
		t.Fatal("should be an error")
	}
}
