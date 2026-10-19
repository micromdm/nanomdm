// Package block implements a middleware service that answers MDM
// requests from blocked enrollments with canned HTTP responses.
package block

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// ErrBlocked is wrapped in the HTTP status error returned for blocked enrollments.
var ErrBlocked = errors.New("enrollment blocked")

// normalize pulls out only the "device" ID (i.e. the "parent" of the)
// MDM relationship regardless of enrollment type. This means a block
// on a device channel also covers all of its user channels.
func normalize(e *mdm.Enrollment) *mdm.EnrollID {
	r := e.Resolved()
	if r == nil {
		return nil
	}
	return &mdm.EnrollID{
		ID:   r.DeviceChannelID,
		Type: r.Type,
	}
}

// Service is a middleware service that blocks MDM requests for blocked enrollments.
type Service struct {
	next       service.CheckinAndCommandService
	logger     log.Logger
	normalizer func(e *mdm.Enrollment) *mdm.EnrollID
	store      storage.BlockStore
}

type Option func(*Service)

// WithLogger sets the logger.
func WithLogger(logger log.Logger) Option {
	return func(s *Service) {
		s.logger = logger
	}
}

// New creates a new block middleware service. It will forward requests
// to next or return HTTP status errors for blocked enrollments.
func New(next service.CheckinAndCommandService, store storage.BlockStore, opts ...Option) *Service {
	if next == nil {
		panic("nil next service")
	}
	if store == nil {
		panic("nil store")
	}
	s := &Service{
		next:       next,
		logger:     log.NopLogger,
		normalizer: normalize,
		store:      store,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// normalizedRequest returns a copy of r with its enrollment ID normalized from e.
func (s *Service) normalizedRequest(r *mdm.Request, e *mdm.Enrollment) (*mdm.Request, error) {
	req := r.WithContext(r.Context())
	req.EnrollID = s.normalizer(e)
	if err := req.EnrollID.Validate(); err != nil {
		return nil, err
	}
	return req, nil
}

// retrieveBlock retrieves the block for the normalized enrollment of e.
// The returned request is the normalized request.
//
// It fails open: if the enrollment can't be normalized or the store
// lookup fails, the error is logged and a nil block is returned so the
// request is forwarded. A storage outage must not turn into rejected
// (or, for the unenroll style, un-enrolled) devices across the fleet.
func (s *Service) retrieveBlock(r *mdm.Request, e *mdm.Enrollment, kind string) (*mdm.Request, *storage.Block) {
	logger := ctxlog.Logger(r.Context(), s.logger)
	req, err := s.normalizedRequest(r, e)
	if err != nil {
		logger.Info("msg", "normalizing enrollment; forwarding", "request", kind, "err", err)
		return nil, nil
	}
	block, err := s.store.RetrieveBlock(req)
	if err != nil {
		logger.Info("msg", "retrieving block; forwarding", "id", req.ID, "request", kind, "err", err)
		return req, nil
	}
	return req, block
}

// blockStatus returns the HTTP status to answer a blocked request with.
// Zero is returned if the request should be forwarded instead.
// isCommand indicates the command report and dequeue request.
func blockStatus(block *storage.Block, isCommand bool) int {
	if block == nil {
		return 0
	}
	switch block.Style {
	case storage.StyleUnenroll:
		return http.StatusUnauthorized
	case storage.StyleServiceUnavailable:
		return http.StatusServiceUnavailable
	case storage.StyleCommandQueue:
		if isCommand {
			return http.StatusServiceUnavailable
		}
	}
	return 0
}

// check returns an HTTP status error if the enrollment of e is blocked.
// The kind is used for logging the type of request.
func (s *Service) check(r *mdm.Request, e *mdm.Enrollment, kind string, isCommand bool) error {
	req, block := s.retrieveBlock(r, e, kind)
	return s.blockError(r, req, block, kind, isCommand)
}

// blockError returns an HTTP status error if block applies to the request.
func (s *Service) blockError(r *mdm.Request, req *mdm.Request, block *storage.Block, kind string, isCommand bool) error {
	status := blockStatus(block, isCommand)
	if status == 0 {
		return nil
	}
	ctxlog.Logger(r.Context(), s.logger).Info(
		"msg", "blocked",
		"id", req.ID,
		"request", kind,
		"style", block.Style,
		"status", status,
	)
	err := fmt.Errorf("%w: style %s", ErrBlocked, block.Style)
	return service.NewHTTPStatusError(status, err)
}
