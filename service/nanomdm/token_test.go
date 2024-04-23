package nanomdm

import (
	"bytes"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
)

func TestToken(t *testing.T) {
	m := NewTokenMux()
	inTok := []byte("hello")
	m.Handle("com.apple.maid", NewStaticToken(inTok))
	inMDMGetToken := &mdm.GetToken{TokenServiceType: "com.apple.maid"}
	outTok, err := m.GetToken(nil, inMDMGetToken)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(inTok, outTok.TokenData) {
		t.Error("input and output not equal")
	}
	// invalid type
	inMDMGetToken = &mdm.GetToken{TokenServiceType: "com.apple.does-not-exist"}
	_, err = m.GetToken(nil, inMDMGetToken)
	if err == nil {
		t.Fatal("should be an error")
	}
}
