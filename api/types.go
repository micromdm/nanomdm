package api

import (
	"fmt"
	"strings"
)

// EnrollmentResults are the per-enrollment ID results of push or enqueue APIs.
type EnrollmentResult struct {
	// PushError is present if there was an error sending the APNs push notification.
	PushError *Error `json:"push_error,omitempty"`

	// PushID is the "apns-id" of a successful APNs push notification.
	PushID string `json:"push_result,omitempty"`

	// EnqueueError is present if there was an error enqueuing the command.
	EnqueueError *Error `json:"command_error,omitempty"`
}

// APIResult is the result of push or enqueue APIs.
type APIResult struct {
	// Status is the per-enrollment ID results of push or enqueue APIs.
	// Map key is the enrollment ID.
	Status map[string]EnrollmentResult `json:"status,omitempty"`

	// NoPush signifies if APNs pushes were not enabled for this API call.
	NoPush bool `json:"no_push,omitempty"`

	// PushError is present if there was an error sending the APNs push notifications.
	PushError *Error `json:"push_error,omitempty"`

	// EnqueueError is present if there was an error enqueuing the command.
	EnqueueError *Error `json:"command_error,omitempty"`

	CommandUUID string `json:"command_uuid,omitempty"` // CommandUUID of the enqueued command.
	RequestType string `json:"request_type,omitempty"` // RequestType of the enqueued command.
}

// Error distills the APIResult errors to a simple error or returns nil.
// If there are more than one error for an enrollment ID the last error is returned.
// Error tries to preserve at least one "real" error by way of wrapping.
func (r *APIResult) Error() error {
	var errCt int
	var statusErr error
	var statusErrID string

	for id, result := range r.Status {
		if result.EnqueueError != nil {
			errCt++
			statusErr = fmt.Errorf("enqueue error: %w", result.EnqueueError)
			statusErrID = id
		} else if result.PushError != nil {
			errCt++
			statusErr = fmt.Errorf("push error: %w", result.PushError)
			statusErrID = id
		}
	}

	var errs []error

	if r.EnqueueError != nil {
		errs = append(errs, fmt.Errorf("enqueue error: %w", r.EnqueueError))
	}

	if r.PushError != nil {
		errs = append(errs, fmt.Errorf("push error: %w", r.PushError))
	}

	if errCt > 0 {
		errs = append(
			errs,
			fmt.Errorf("status errors (%d): last error for %s: %w", errCt, statusErrID, statusErr),
		)
	}

	if len(errs) == 1 {
		return errs[0]
	} else if len(errs) > 1 {
		var errStrs []string
		for _, err := range errs {
			errStrs = append(errStrs, err.Error())
		}
		return fmt.Errorf("%w; %s", errs[0], strings.Join(errStrs, "; "))
	}

	return nil
}

// QueueAPIResult is the result of queue APIs.
type QueueAPIResult struct {
	// Status is the per-enrollment ID results of queue APIs.
	// Map key is the enrollment ID.
	Status map[string]*Error `json:"status,omitempty"`
	// Error is present if there was a general error with the queue API call.
	Error *Error `json:"error,omitempty"`
}
