package storage

import (
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
)

// BlockStyle is how a blocked enrollment is answered.
type BlockStyle string

const (
	// StyleUnenroll returns HTTP 401 to any request, which per the MDM
	// protocol causes the device to remove its MDM enrollment profile.
	StyleUnenroll BlockStyle = "unenroll"

	// StyleServiceUnavailable returns HTTP 503 to almost any request. This status
	// is not defined by the MDM protocol; it is our invented temporary
	// block that leaves the enrollment intact but halts progress.
	StyleServiceUnavailable BlockStyle = "service-unavailable"

	// StyleCommandQueue returns HTTP 503 just for the command report and
	// dequeue endpoints and allows all Check-In messages to proceed.
	StyleCommandQueue BlockStyle = "command-queue"

	// StyleUnrecognizedDevice returns HTTP 403 to any request with the
	// "com.apple.unrecognized.device" error body, which causes the device
	// to unenroll. Apple supports this from macOS 14 and iOS 17 and says
	// to use it instead of a 401; older OS versions ignore it.
	StyleUnrecognizedDevice BlockStyle = "unrecognized-device"
)

// ErrInvalidBlockStyle is returned when a block has an unknown style.
var ErrInvalidBlockStyle = errors.New("invalid block style")

// Valid returns an error if s is not a known block style.
func (s BlockStyle) Valid() error {
	switch s {
	case StyleUnenroll, StyleServiceUnavailable, StyleCommandQueue, StyleUnrecognizedDevice:
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidBlockStyle, string(s))
}

// Block is the block configuration of an enrollment.
type Block struct {
	// ClearOnAuthenticate, when true, clears the block when the device
	// sends an Authenticate check-in and allows that Authenticate to
	// proceed normally ("block until re-enrollment").
	ClearOnAuthenticate bool `json:"clear_on_authenticate"`

	Style BlockStyle `json:"style"`
}

// Validate returns an error if b is nil or has an invalid style.
func (b *Block) Validate() error {
	if b == nil {
		return errors.New("nil block")
	}
	return b.Style.Valid()
}

// BlockStore stores and retrieves enrollment blocks.
// The request enrollment ID should be normalized for just the device channel.
type BlockStore interface {
	// RetrieveBlock returns the block for the request's enrollment ID, or
	// nil if the enrollment is not blocked.
	RetrieveBlock(r *mdm.Request) (*Block, error)

	// StoreBlock sets (or replaces) the block for the enrollment ID.
	StoreBlock(r *mdm.Request, block *Block) error

	// RemoveBlock clears any block for the enrollment ID.
	// It is not an error if the enrollment is not blocked.
	RemoveBlock(r *mdm.Request) error
}
