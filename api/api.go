// Package api defines a consistent Go API for enqueueing or sending APNs pushes
// to multiple enrollment IDs.
package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"
)

// PushEnqueuer can enqueue commands and send APNs pushes.
type PushEnqueuer struct {
	logger log.Logger
	store  storage.CommandEnqueuer
	pusher push.Pusher
	noPush bool
}

// PushEnqueuerOptions configures the push enqueuer.
type PushEnqueuerOption func(*PushEnqueuer) error

// WithLogger configures a logger on a push enqueuer.
func WithLogger(logger log.Logger) PushEnqueuerOption {
	return func(pe *PushEnqueuer) error {
		pe.logger = logger
		return nil
	}
}

// WithNoPush disables push attempts.
func WithNoPush() PushEnqueuerOption {
	return func(pe *PushEnqueuer) error {
		pe.noPush = true
		return nil
	}
}

// NewPushEnqueuer creates a new push enqueuer.
func NewPushEnqueuer(store storage.CommandEnqueuer, pusher push.Pusher, opts ...PushEnqueuerOption) (*PushEnqueuer, error) {
	if store == nil && pusher == nil {
		return nil, errors.New("store and pusher both nil")
	}
	pe := &PushEnqueuer{
		logger: nil,
		store:  store,
		pusher: pusher,
	}
	for _, opt := range opts {
		if err := opt(pe); err != nil {
			return nil, err
		}
	}
	return pe, nil
}

// Push sends APNs notifications to ids.
func (pe *PushEnqueuer) Push(ctx context.Context, ids []string) (*APIResult, int, error) {
	return pe.EnqueueWithPush(ctx, nil, ids, false)
}

// EnqueueWithPush enqueues command and can send APNs pushes to ids.
// A command cannot be nil while noPush is true.
// The return integer is an indicator of errors with the actual errors
// contained within the API result.
// A 500 value indicates only errors (with no successes).
// A 207 value indicates some sucesses and some failures.
// A 200 value indicates no errors (with only accesses).
// Any other value is undefined.
func (pe *PushEnqueuer) EnqueueWithPush(ctx context.Context, command *mdm.Command, ids []string, noPush bool) (*APIResult, int, error) {
	// setup our result accumulator
	r := &APIResult{
		NoPush: noPush || pe.noPush,
	}

	if command == nil && noPush {
		return r, 500, errors.New("must enqueue or push")
	}

	if command != nil {
		doEnqueue(ctx, r, pe.logger, pe.store, command, ids)
	}

	if !noPush && !pe.noPush && r.EnqueueError == nil {
		// TODO: only push to non-erroring enrollment IDs
		doPush(ctx, r, pe.logger, pe.pusher, ids)
	}

	return r, code(r, len(ids)), nil
}

// code translates an [APIResult] to an interger code.
// See [EnqueueWithPush] for specific code meanings.
func code(r *APIResult, idCount int) int {
	if r == nil {
		return 500
	}

	if r.PushError != nil || r.EnqueueError != nil {
		// if there was any high-level error
		// we consider that a complete failure.
		return 500
	}

	var errCt int
	for _, er := range r.Status {
		if er.PushError != nil || er.EnqueueError != nil {
			errCt++
		}
	}

	if errCt < 1 {
		// if no high-level errors and no individual erros then all good.
		return 200
	} else if errCt == idCount {
		// same amount of errors as we attempted to send things
		return 500
	} else if errCt < idCount {
		return 207
	}

	// more errCt than idCount? assume that's bad.
	return 500
}

// RawCommandEnqueueWithPush enqueues rawCommand and can send APNs pushes to ids.
// See [EnqueueWithPush] for calling semantics.
func (pe *PushEnqueuer) RawCommandEnqueueWithPush(ctx context.Context, rawCommand []byte, ids []string, noPush bool) (*APIResult, int, error) {
	command, err := mdm.DecodeCommand(rawCommand)
	if err != nil {
		return nil, 500, fmt.Errorf("decoding command: %w", err)
	}
	return pe.EnqueueWithPush(ctx, command, ids, noPush)
}
