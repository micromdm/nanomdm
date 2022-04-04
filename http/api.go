package http

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/log/ctxlog"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"
)

// enrolledAPIResult is a per-enrollment API result.
type enrolledAPIResult struct {
	PushError    string `json:"push_error,omitempty"`
	PushResult   string `json:"push_result,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

// enrolledAPIResults is a map of enrollments to a per-enrollment API result.
type enrolledAPIResults map[string]*enrolledAPIResult

// apiResult is the JSON reply returned from either pushing or queuing commands.
type apiResult struct {
	Status       enrolledAPIResults `json:"status,omitempty"`
	NoPush       bool               `json:"no_push,omitempty"`
	PushError    string             `json:"push_error,omitempty"`
	CommandError string             `json:"command_error,omitempty"`
	CommandUUID  string             `json:"command_uuid,omitempty"`
	RequestType  string             `json:"request_type,omitempty"`
}

type (
	ctxKeyIDFirst struct{}
	ctxKeyIDCount struct{}
)

func setAPIIDs(ctx context.Context, idFirst string, idCount int) context.Context {
	ctx = context.WithValue(ctx, ctxKeyIDFirst{}, idFirst)
	return context.WithValue(ctx, ctxKeyIDCount{}, idCount)
}

func ctxKVs(ctx context.Context) (out []interface{}) {
	id, ok := ctx.Value(ctxKeyIDFirst{}).(string)
	if ok {
		out = append(out, "id_first", id)
	}
	eType, ok := ctx.Value(ctxKeyIDCount{}).(int)
	if ok {
		out = append(out, "id_count", eType)
	}
	return
}

func setupCtxLog(ctx context.Context, ids []string, logger log.Logger) (context.Context, log.Logger) {
	if len(ids) > 0 {
		ctx = setAPIIDs(ctx, ids[0], len(ids))
		ctx = ctxlog.AddFunc(ctx, ctxKVs)
	}
	return ctx, ctxlog.Logger(ctx, logger)
}

// PushHandler sends APNs push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to push to. This
// probably necessitates stripping the URL prefix before using. Also
// note we expose Go errors to the output as this is meant for "API"
// users.
func PushHandler(pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path, ",")
		ctx, logger := setupCtxLog(r.Context(), ids, logger)
		output := apiResult{
			Status: make(enrolledAPIResults),
		}
		logs := []interface{}{"msg", "push"}
		pushResp, err := pusher.Push(ctx, ids)
		if err != nil {
			logs = append(logs, "err", err)
			output.PushError = err.Error()
		}
		var ct, errCt int
		for id, resp := range pushResp {
			output.Status[id] = &enrolledAPIResult{
				PushResult: resp.Id,
			}
			if resp.Err != nil {
				output.Status[id].PushError = resp.Err.Error()
				errCt += 1
			} else {
				ct += 1
			}
		}
		logs = append(logs, "count", ct)
		if errCt > 0 {
			logs = append(logs, "errs", errCt)
		}
		if err != nil || errCt > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
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
func RawCommandEnqueueHandler(enqueuer storage.CommandEnqueuer, pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := strings.Split(r.URL.Path, ",")
		ctx, logger := setupCtxLog(r.Context(), ids, logger)
		b, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		command, err := mdm.DecodeCommand(b)
		if err != nil {
			logger.Info("msg", "decoding command", "err", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		nopush := r.URL.Query().Get("nopush") != ""
		output := apiResult{
			Status:      make(enrolledAPIResults),
			NoPush:      nopush,
			CommandUUID: command.CommandUUID,
			RequestType: command.Command.RequestType,
		}
		logger = logger.With(
			"command_uuid", command.CommandUUID,
			"request_type", command.Command.RequestType,
		)
		logs := []interface{}{
			"msg", "enqueue",
		}
		idErrs, err := enqueuer.EnqueueCommand(ctx, ids, command)
		if err != nil {
			logs = append(logs, "err", err)
			output.CommandError = err.Error()
		}
		logs = append(logs, "count", len(ids)-len(idErrs))
		if len(idErrs) > 0 {
			logs = append(logs, "errs", len(idErrs))
		}
		if err != nil || len(idErrs) > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		pushResp := make(map[string]*push.Response)
		if !nopush {
			pushResp, err = pusher.Push(ctx, ids)
			if err != nil {
				logger.Info("msg", "push", "err", err)
				output.PushError = err.Error()
			}
		} else {
			err = nil
		}
		// loop through our command errors, if any, and add to output
		for id, err := range idErrs {
			if err != nil {
				output.Status[id] = &enrolledAPIResult{
					CommandError: err.Error(),
				}
			}
		}
		// loop through our push errors, if any, and add to output
		var pushCt, pushErrCt int
		for id, resp := range pushResp {
			if _, ok := output.Status[id]; ok {
				output.Status[id].PushResult = resp.Id
			} else {
				output.Status[id] = &enrolledAPIResult{
					PushResult: resp.Id,
				}
			}
			if resp.Err != nil {
				output.Status[id].PushError = resp.Err.Error()
				pushErrCt++
			} else {
				pushCt++
			}
		}
		logs = []interface{}{
			"msg", "push",
			"count", pushCt,
		}
		if err != nil {
			logs = append(logs, "err", err)
		}
		if pushErrCt > 0 {
			logs = append(logs, "errs", pushErrCt)
		}
		if err != nil || pushErrCt > 0 {
			logger.Info(logs...)
		} else {
			logger.Debug(logs...)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}

// StorePushCertHandler reads a PEM-encoded certificate and private
// key from the HTTP body and saves it to storage. This effectively
// enables us to do something like:
// "% cat push.pem push.key | curl -T - http://api.example.com/" to
// upload our push certs.
func StorePushCertHandler(storage storage.PushCertStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		b, err := ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		// if the PEM blocks are mushed together with no newline then add one
		b = bytes.ReplaceAll(b, []byte("----------"), []byte("-----\n-----"))
		var pemCert []byte
		var pemKey []byte
		var topic string
		var block *pem.Block
		for {
			block, b = pem.Decode(b)
			if block == nil {
				break
			}
			switch block.Type {
			case "CERTIFICATE":
				pemCert = pem.EncodeToMemory(block)
				var cert *x509.Certificate
				cert, err = x509.ParseCertificate(block.Bytes)
				if err == nil {
					topic, err = cryptoutil.TopicFromCert(cert)
				}
			case "RSA PRIVATE KEY", "PRIVATE KEY":
				pemKey = pem.EncodeToMemory(block)
			default:
				err = fmt.Errorf("unrecognized PEM type: %q", block.Type)
			}
			if err != nil {
				break
			}
		}
		if err == nil {
			if len(pemCert) == 0 {
				err = errors.New("cert not found")
			} else if len(pemKey) == 0 {
				err = errors.New("private key not found")
			}
		}
		if err == nil {
			err = storage.StorePushCert(r.Context(), pemCert, pemKey)
		}
		output := &struct {
			Error string `json:"error,omitempty"`
			Topic string `json:"topic,omitempty"`
		}{
			Topic: topic,
		}
		if err != nil {
			logger.Info("msg", "store push cert", "err", err)
			output.Error = err.Error()
		} else {
			logger.Info("msg", "stored push cert", "topic", topic)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
		}
		w.Header().Set("Content-type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}
