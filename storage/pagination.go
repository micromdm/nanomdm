package storage

import "errors"

var (
	ErrBothCursorAndOffset = errors.New("both cursor and offset set")
	ErrOnlyCursor          = errors.New("only cursor-based pagination supported")
	ErrOnlyOffset          = errors.New("only offset-based pagination supported")
)

// Pagination supports either offset or cursor based pagination methods for results.
// Backend implementations may support one or both methods, and may use either by default.
// Offset and Cursor cannot both be set.
type Pagination struct {
	// For offset based pagination queries, Offset must be set.
	// Cannot be used with Cursor.
	Offset *int `json:"offset,omitempty"`

	// Backend implementations may have a default Limit, so omitting it may be possible.
	Limit *int `json:"limit,omitempty"`

	// For cursor based queries, Cursor must be set.
	// Cannot be used with Offset.
	// The initial cursor can be set to "" (empty string).
	// The next cursor should be returned in the result.
	Cursor *string `json:"cursor,omitempty"`
}

// PaginationNextCursor is for embedding in query result types.
type PaginationNextCursor struct {
	// When using cursor-based pagination (for backends that support it)
	// this should contain the next cursor to use in the pagination.
	NextCursor *string `json:"next_cursor,omitempty"`
}

// ValidErr returns validation errors for p.
func (p *Pagination) ValidErr() error {
	if p == nil {
		// it's technically valid to have no pagination.
		return nil
	}
	if p.Cursor != nil && p.Offset != nil {
		return ErrBothCursorAndOffset
	}
	return nil
}

// Valid returns true if p is valid.
func (p *Pagination) Valid() bool {
	return p == nil || p.ValidErr() == nil
}

// DefaultOffsetLimit pulls out default offset and limit parameters from p.
// If the limit in p is nil or less than 1, then default limit is returned,
// otherwise the limit in p is returned.
// Defaults of 0, 0 are otherwise returned.
func (p *Pagination) DefaultOffsetLimit(defaultLimit int) (offset int, limit int) {
	if p == nil {
		limit = defaultLimit
		return
	}
	if p.Offset != nil {
		offset = *p.Offset
	}
	if p.Limit == nil || *p.Limit < 1 {
		limit = defaultLimit
	} else {
		limit = *p.Limit
	}
	return
}

// ValidateDefaultOffsetLimit returns pagination details and errors, if present.
func (p *Pagination) ValidateDefaultOffsetLimit(defaultLimit int) (cursor string, offset, limit int, err error) {
	err = p.ValidErr()
	offset, limit = p.DefaultOffsetLimit(defaultLimit)
	if p != nil && p.Cursor != nil {
		cursor = *p.Cursor
	}
	return
}
