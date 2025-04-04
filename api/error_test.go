package api

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestErrorJSON(t *testing.T) {
	inStr := "hello, world!"
	inErr := errors.New(inStr)

	inError := NewError(inErr)

	jsonBytes, err := json.Marshal(inError)
	if err != nil {
		t.Fatal(err)
	}

	outError := NewError(nil)

	err = json.Unmarshal(jsonBytes, outError)
	if err != nil {
		t.Fatal(err)
	}

	// compare the very initial error to the
	// marshalled-then-unmarshalled error
	if want, have := inStr, outError.Error(); want != have {
		t.Errorf("want: %v, have: %v", want, have)
	}
}
