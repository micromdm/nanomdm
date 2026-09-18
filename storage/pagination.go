package storage

import "errors"

// Pagination errors.
var (
	// ErrBothCursorAndOffset is returned if both cursor and offset are set in a pagination request.
	ErrBothCursorAndOffset = errors.New("both cursor and offset set")

	// ErrOnlyCursor is returned if only cursor-based pagination is supported but offset was provided.
	ErrOnlyCursor = errors.New("only cursor-based pagination supported")

	// ErrOnlyOffset is returned if only offset-based pagination is supported but cursor was provided.
	ErrOnlyOffset = errors.New("only offset-based pagination supported")
)

// Pagination contains pagination parameters for queries.
type Pagination struct {
	Offset *int    `json:"offset,omitempty"`
	Limit  *int    `json:"limit,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

// PaginationNextCursor contains the cursor for the next page of results.
type PaginationNextCursor struct {
	NextCursor *string `json:"next_cursor,omitempty"`
}

// ValidErr returns an error if the pagination parameters are invalid.
// Specifically, it checks that both cursor and offset are not set simultaneously.
func (p *Pagination) ValidErr() error {
	if p == nil {
		return nil
	}
	if p.Cursor != nil && p.Offset != nil {
		return ErrBothCursorAndOffset
	}
	return nil
}

// Valid returns true if the pagination parameters are valid.
func (p *Pagination) Valid() bool {
	return p.ValidErr() == nil
}

// DefaultOffsetLimit returns the offset and limit values with defaults applied.
// If offset is not set, 0 is returned. If limit is not set, defaultLimit is returned.
func (p *Pagination) DefaultOffsetLimit(defaultLimit int) (offset, limit int) {
	if p == nil {
		return 0, defaultLimit
	}
	if p.Offset != nil {
		offset = *p.Offset
	}
	if p.Limit != nil {
		limit = *p.Limit
	} else {
		limit = defaultLimit
	}
	return
}

// ValidateDefaultOffsetLimit validates the pagination and returns offset and limit with defaults.
// Returns an error if both cursor and offset are set.
func (p *Pagination) ValidateDefaultOffsetLimit(defaultLimit int) (offset, limit int, err error) {
	if err = p.ValidErr(); err != nil {
		return
	}
	offset, limit = p.DefaultOffsetLimit(defaultLimit)
	return
}
