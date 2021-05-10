package mdm

import (
	"errors"
	"fmt"

	"github.com/groob/plist"
)

var ErrUnrecognizedMessageType = errors.New("unrecognized MessageType")

// MessageType represents the MessageType of a check-in message
type MessageType struct {
	MessageType string
}

// Authenticate is a representation of an "Authenticate" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/authenticaterequest
type Authenticate struct {
	Enrollment
	MessageType
	Topic string
	Raw   []byte // Original Authenticate XML plist

	// Fields that may be present but are not strictly required for the
	// operation of the MDM protocol. Nice-to-haves.
	SerialNumber string
}

// TokenUpdate is a representation of a "TokenUpdate" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/token_update
type TokenUpdate struct {
	Enrollment
	MessageType
	Push
	UnlockToken []byte `plist:",omitempty"`
	Raw         []byte // Original TokenUpdate XML plist
}

// CheckOut is a representation of a "CheckOut" check-in message type.
// See https://developer.apple.com/documentation/devicemanagement/checkoutrequest
type CheckOut struct {
	Enrollment
	MessageType
	Raw []byte // Original CheckOut XML plist
}

// newCheckinMessageForType returns a pointer to a check-in struct for MessageType t
func newCheckinMessageForType(t string, raw []byte) interface{} {
	switch t {
	case "Authenticate":
		return &Authenticate{Raw: raw}
	case "TokenUpdate":
		return &TokenUpdate{Raw: raw}
	case "CheckOut":
		return &CheckOut{Raw: raw}
	default:
		return nil
	}
}

// checkinUnmarshaller facilitates unmarshalling a plist check-in message.
type checkinUnmarshaller struct {
	message interface{}
	raw     []byte
}

// UnmarshalPlist populates the message field of w based on the contents of a plist.
func (w *checkinUnmarshaller) UnmarshalPlist(f func(interface{}) error) error {
	onlyType := new(MessageType)
	err := f(onlyType)
	if err != nil {
		return err
	}
	w.message = newCheckinMessageForType(onlyType.MessageType, w.raw)
	if w.message == nil {
		return fmt.Errorf("%w: %q", ErrUnrecognizedMessageType, onlyType.MessageType)
	}
	return f(w.message)
}

// DecodeCheckin unmarshals rawMessage into a specific check-in struct in message.
func DecodeCheckin(rawMessage []byte) (message interface{}, err error) {
	w := &checkinUnmarshaller{raw: rawMessage}
	err = plist.Unmarshal(rawMessage, w)
	message = w.message
	return
}
