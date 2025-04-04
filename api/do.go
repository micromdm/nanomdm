package api

import (
	"context"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// doPush sends MDM APNs push notifications to ids using pusher.
// Results and/or errors are accumulated in r and logged to logger.
func doPush(ctx context.Context, r *APIResult, logger log.Logger, pusher push.Pusher, ids []string) {
	var errCt int
	var err error
	logs := []interface{}{
		"msg", "push",
		"id_count", len(ids),
	}
	if logger != nil {
		// setup our deferred logger
		defer func() {
			if err != nil || errCt > 0 {
				if errCt > 0 {
					logs = append(logs, "errs", errCt)
				}
				if err != nil {
					logs = append(logs, "err", err)
				}
				ctxlog.Logger(ctx, logger).Info(logs...)
			} else {
				ctxlog.Logger(ctx, logger).Debug(logs...)
			}
		}()
	}

	if r == nil {
		err = errors.New("nil accumulator")
		return
	}

	if len(ids) > 0 {
		logs = append(logs, "id_first", ids[0])
	}

	// even though command UUID and RequestType aren't really
	// applicable to pushing, include the logs here to try and
	// connect any dogs in the logging.
	if r.CommandUUID != "" {
		logs = append(logs, "command_uuid", r.CommandUUID)
	}
	if r.RequestType != "" {
		logs = append(logs, "request_type", r.RequestType)
	}

	if pusher == nil {
		err = errors.New("nil pusher")
		r.PushError = NewError(err)
		return
	}

	// send APNs push notification(s)
	pr, err := pusher.Push(ctx, ids)
	if err != nil {
		r.PushError = NewError(err)
	}

	if len(pr) > 0 && r.Status == nil {
		// init the results if there are any
		r.Status = make(map[string]EnrollmentResult)
	}

	// loop through any push responses and populate results
	var pushCt int
	for id, pushResponse := range pr {
		er := r.Status[id]
		er.PushID = pushResponse.Id
		if pushResponse.Err != nil {
			errCt++
			er.PushError = NewError(pushResponse.Err)
		} else {
			// we assume a lack of error means a "success"
			// however PushID could conceivably be empty still,
			// suggesting something else went wrong.
			pushCt++
		}
		r.Status[id] = er
	}

	logs = append(logs, "count", pushCt)
}

// doEnqueue enqueues the MDM command to ids using store.
// Results and/or errors are accumulated in r and logged to logger.
func doEnqueue(ctx context.Context, r *APIResult, logger log.Logger, store storage.CommandEnqueuer, cmd *mdm.Command, ids []string) {
	var idErrs map[string]error
	var err error
	logs := []interface{}{
		"msg", "enqueue",
		"id_count", len(ids),
	}
	if logger != nil {
		// setup our deferred logger
		defer func() {
			if err != nil || len(idErrs) > 0 {
				if len(idErrs) > 0 {
					logs = append(logs, "errs", len(idErrs))
				}
				if err != nil {
					logs = append(logs, "err", err)
				}
				ctxlog.Logger(ctx, logger).Info(logs...)
			} else {
				ctxlog.Logger(ctx, logger).Debug(logs...)
			}
		}()
	}

	if len(ids) > 0 {
		logs = append(logs, "id_first", ids[0])
	}

	if r == nil {
		err = errors.New("nil accumulator")
		return
	}

	if cmd != nil {
		r.CommandUUID = cmd.CommandUUID
		r.RequestType = cmd.Command.RequestType
		logs = append(logs,
			"command_uuid", r.CommandUUID,
			"request_type", r.RequestType,
		)
	}

	if store == nil {
		err = errors.New("nil store")
		r.EnqueueError = NewError(err)
		return
	}

	// enqueue command
	idErrs, err = store.EnqueueCommand(ctx, ids, cmd)
	if err != nil {
		r.EnqueueError = NewError(err)
	}

	if len(idErrs) > 0 && r.Status == nil {
		// init the results if there are any
		r.Status = make(map[string]EnrollmentResult)
	}

	// loop through any id errors and populate results
	for id, err := range idErrs {
		er := r.Status[id]
		if err == nil {
			err = errors.New("unknown enqueue error")
		}
		er.EnqueueError = NewError(err)
		r.Status[id] = er
	}

	logs = append(logs, "count", len(ids)-len(idErrs))
}
