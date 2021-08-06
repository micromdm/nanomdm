package http

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/log"
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

// PushHandlerFunc sends APNs push notifications to MDM enrollments.
//
// Note the whole URL path is used as the identifier to push to. This
// probably necessitates stripping the URL prefix before using. Also
// note we expose Go errors to the output as this is meant for "API"
// users.
func PushHandlerFunc(pusher push.Pusher, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var hadServerError bool
		ids := strings.Split(r.URL.Path, ",")
		output := apiResult{
			Status: make(enrolledAPIResults),
		}
		pushResp, err := pusher.Push(r.Context(), ids)
		if err != nil {
			logger.Info("msg", "push", "err", err)
			output.PushError = err.Error()
			hadServerError = true
		}
		var ct, errCt int
		for id, resp := range pushResp {
			output.Status[id] = &enrolledAPIResult{
				PushResult: resp.Id,
			}
			if resp.Err != nil {
				output.Status[id].PushError = resp.Err.Error()
				errCt += 1
				hadServerError = true
			} else {
				ct += 1
			}
		}
		logger.Debug("msg", "push", "count", ct, "errs", errCt)
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
			hadServerError = true
		}
		w.Header().Set("Content-type", "application/json")
		if hadServerError {
			w.WriteHeader(http.StatusInternalServerError)
		}
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
		ids := strings.Split(r.URL.Path, ",")
		nopush := r.URL.Query().Get("nopush") != ""
		output := apiResult{
			Status:      make(enrolledAPIResults),
			NoPush:      nopush,
			CommandUUID: command.CommandUUID,
			RequestType: command.Command.RequestType,
		}
		var hadServerError bool
		idErrs, err := enqueuer.EnqueueCommand(r.Context(), ids, command)
		if err != nil {
			logger.Info("msg", "enqueue command", "err", err)
			output.CommandError = err.Error()
			hadServerError = true
		}
		pushResp := make(map[string]*push.Response)
		if !nopush {
			pushResp, err = pusher.Push(r.Context(), ids)
			if err != nil {
				logger.Info("msg", "push", "err", err)
				output.PushError = err.Error()
				hadServerError = true
			}
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
			}
		}
		logger.Debug(
			"msg", "enqueue",
			"command_uuid", command.CommandUUID,
			"request_type", command.Command.RequestType,
			"id_count", len(ids),
			"id_first", ids[0],
		)
		logger.Debug("msg", "push", "count", len(pushResp))
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
			hadServerError = true
		}
		w.Header().Set("Content-type", "application/json")
		if hadServerError {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}

// StorePushCertHandlerFunc reads a PEM-encoded certificate and private
// key from the HTTP body and saves it to storage. This effectively
// enables us to do something like:
// "% cat push.pem push.key | curl -T - http://api.example.com/" to
// upload our push certs.
func StorePushCertHandlerFunc(storage storage.PushCertStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		var hadServerError bool
		for {
			block, b = pem.Decode(b)
			if block == nil {
				break
			}
			switch block.Type {
			case "CERTIFICATE":
				pemCert = pem.EncodeToMemory(block)
				cert, err := x509.ParseCertificate(block.Bytes)
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
			hadServerError = true
		} else {
			logger.Info("msg", "stored push cert", "topic", topic)
		}
		json, err := json.MarshalIndent(output, "", "\t")
		if err != nil {
			logger.Info("msg", "marshal json", "err", err)
			hadServerError = true
		}
		w.Header().Set("Content-type", "application/json")
		if hadServerError {
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, err = w.Write(json)
		if err != nil {
			logger.Info("msg", "writing body", "err", err)
		}
	}
}
