package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/micromdm/nanomdm/api"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// writeJSON encodes v to JSON writing to w using the HTTP status of header.
// An error during encoding is logged to logger if it is not nil.
func writeJSON(w http.ResponseWriter, v interface{}, header int, logger log.Logger) {
	if header < 1 {
		header = http.StatusInternalServerError
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(header)

	if v == nil {
		return
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err := enc.Encode(v)
	if err != nil && logger != nil {
		logger.Info("msg", "encoding json", "err", err)
	}
}

// logAndWriteJSONError logs msg and err to logger as well as writes err to w as JSON.
func logAndWriteJSONError(logger log.Logger, w http.ResponseWriter, msg string, err error, header int) {
	if logger != nil {
		logger.Info("msg", msg, "err", err)
	}

	errStr := "<nil error>"
	if err != nil {
		errStr = err.Error()
	}

	out := &ErrorResponseJson{Error: errStr}

	writeJSON(w, out, header, logger)
}

// writeAPIResult encodes r to JSON writing to w using the HTTP status of header.
func writeAPIResult(r *api.APIResult, w http.ResponseWriter, header int, logger log.Logger) {
	if r == nil {
		nilErr := api.NewError(errors.New("nil API result"))
		r = &api.APIResult{
			EnqueueError: nilErr,
			PushError:    nilErr,
		}
		header = 0 // override http status if a nil API result happens
	}

	writeJSON(w, r, header, logger)
}

// amendAPIError amends or inserts err into e.
func amendAPIError(err error, e **api.Error) {
	if e == nil || err == nil {
		return
	}
	if *e == nil {
		// add new
		*e = api.NewError(err)
	} else {
		// amend any existing error
		*e = api.NewError(fmt.Errorf("result API error: %w; previous error: %v", err, (*e).Err))
	}
}

// PathIDGetter returns the list of comma-separated enrollment IDs from r.
func PathIDGetter(r *http.Request) ([]string, error) {
	if r.URL.Path == "" {
		return nil, errors.New("empty path")
	}
	return strings.Split(r.URL.Path, ","), nil
}

// PushHandler sends APNs push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to push to. This
// probably necessitates stripping the URL prefix before using. Also
// note we expose Go errors to the output as this is meant for "API"
// users.
//
// Deprecated: use [PushToIDsHandler] instead.
// Use [PathIDGetter] with it for the previous behavior.
func PushHandler(pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return PushToIDsHandler(pusher, logger, PathIDGetter)
}

// PushToIDsHandler sends APNs push notifications to MDM enrollments.
// Use idGetter to get the slice of enrollment IDs from the HTTP request.
func PushToIDsHandler(pusher push.Pusher, logger log.Logger, idGetter func(*http.Request) ([]string, error)) http.HandlerFunc {
	if pusher == nil {
		panic("nil pusher")
	}

	pe, peErr := api.NewPushEnqueuer(nil, pusher, api.WithLogger(logger))
	if peErr != nil {
		panic(peErr)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var pr *api.APIResult
		header := http.StatusInternalServerError
		logger := ctxlog.Logger(r.Context(), logger)

		defer func() {
			writeAPIResult(pr, w, header, logger)
		}()

		ids, err := idGetter(r)
		if err != nil {
			err = fmt.Errorf("getting enrollment ids: %w", err)
			logger.Info("err", err)
			// synthesize an API result error
			pr = new(api.APIResult)
			amendAPIError(err, &pr.PushError)
			return
		}

		pr, header, err = pe.Push(r.Context(), ids)
		if err != nil {
			if pr == nil {
				pr = new(api.APIResult)
			}
			// amend the result json with our error
			// so as to be visible to HTTP API callers
			amendAPIError(err, &pr.PushError)
			logs := []interface{}{
				"msg", "sending push",
				"id_count", len(ids),
				"err", err,
			}
			if len(ids) > 0 {
				logs = append(logs, "id_first", ids[0])
			}
			logger.Info(logs...)
		}
	}
}

// RawCommandEnqueueHandler enqueues a raw MDM command plist and sends
// push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to enqueue (and
// push to. This probably necessitates stripping the URL prefix before
// using. Also note we expose Go errors to the output as this is meant
// for "API" users.
//
// Deprecated: use [RawCommandEnqueueToIDsHandler] instead.
// Use [PathIDGetter] with it for the previous behavior.
func RawCommandEnqueueHandler(enqueuer storage.CommandEnqueuer, pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return RawCommandEnqueueToIDsHandler(enqueuer, pusher, logger, PathIDGetter)
}

// RawCommandEnqueueToIDsHandler enqueues a raw MDM command and sends
// push notifications to MDM enrollments.
// Use idGetter to get the slice of enrollment IDs from the HTTP request.
func RawCommandEnqueueToIDsHandler(enqueuer storage.CommandEnqueuer, pusher push.Pusher, logger log.Logger, idGetter func(*http.Request) ([]string, error)) http.HandlerFunc {
	if enqueuer == nil {
		panic("nil enqueuer")
	}

	pe, peErr := api.NewPushEnqueuer(enqueuer, pusher, api.WithLogger(logger))
	if peErr != nil {
		panic(peErr)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var er *api.APIResult
		header := http.StatusInternalServerError
		logger := ctxlog.Logger(r.Context(), logger)

		defer func() {
			writeAPIResult(er, w, header, logger)
		}()

		ids, err := idGetter(r)
		if err != nil {
			err = fmt.Errorf("getting enrollment ids: %w", err)
			logger.Info("err", err)
			// synthesize an API result error
			er = new(api.APIResult)
			amendAPIError(err, &er.EnqueueError)
			return
		}

		cmdBytes, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			// synthesize an API result error
			er := new(api.APIResult)
			amendAPIError(err, &er.EnqueueError)
			return
		}

		noPush := r.URL.Query().Get("nopush") != ""

		er, header, err = pe.RawCommandEnqueueWithPush(r.Context(), cmdBytes, ids, noPush)
		if err != nil {
			if er == nil {
				er = new(api.APIResult)
			}
			// amend the result json with our error
			// so as to be visible to HTTP API callers
			amendAPIError(err, &er.EnqueueError)
			logs := []interface{}{
				"msg", "enqueueing",
				"id_count", len(ids),
				"err", err,
			}
			if len(ids) > 0 {
				logs = append(logs, "id_first", ids[0])
			}
			logger.Info(logs...)
		}
	}
}
