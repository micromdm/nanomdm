package api

import (
	"encoding/json"
	"errors"
)

// Error wraps errors for marshalling and unmarshalling.
type Error struct {
	Err error
}

// NewError contains err in a marshalling and unmarhalling wrapper.
func NewError(err error) *Error {
	return &Error{err}
}

// Valid returns true if e and our error are not nil.
func (e *Error) Valid() bool {
	if e == nil || e.Err == nil {
		return false
	}
	return true
}

// Error returns the error string.
func (e *Error) Error() string {
	if !e.Valid() {
		return ""
	}
	return e.Err.Error()
}

// Unwrap returns the contained error.
func (e *Error) Unwrap() error {
	return e.Err
}

// MarshalJSON renders the contained error as a JSON string.
func (e *Error) MarshalJSON() ([]byte, error) {
	if !e.Valid() {
		return []byte(`"nil error"`), nil
	}
	return json.Marshal(e.Err.Error())
}

// UnmarshalJSON overwrites the contained error with a new plain string error.
func (e *Error) UnmarshalJSON(b []byte) error {
	if e == nil {
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	e.Err = errors.New(s)
	return nil
}
