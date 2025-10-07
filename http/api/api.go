package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/micromdm/nanomdm/api"
	"github.com/micromdm/nanomdm/cryptoutil"
	mdmhttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// writeAPIResult encodes r to JSON to w, logging errors to logger if necessary.
func writeAPIResult(logger log.Logger, w http.ResponseWriter, r *api.APIResult, header int) {
	if header < 1 {
		header = http.StatusInternalServerError
	}

	if r == nil {
		nilErr := api.NewError(errors.New("nil API result"))
		r = &api.APIResult{
			EnqueueError: nilErr,
			PushError:    nilErr,
		}
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(header)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")

	err := enc.Encode(r)
	if err != nil && logger != nil {
		logger.Info("msg", "encoding json", "err", err)
	}
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
			writeAPIResult(logger, w, pr, header)
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
			writeAPIResult(logger, w, er, header)
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

// readPEMCertAndKey reads a PEM-encoded certificate and non-encrypted
// private key from input bytes and returns the separate PEM certificate
// and private key in cert and key respectively.
func readPEMCertAndKey(input []byte) (cert []byte, key []byte, err error) {
	// if the PEM blocks are mushed together with no newline then add one
	input = bytes.ReplaceAll(input, []byte("----------"), []byte("-----\n-----"))
	var block *pem.Block
	for {
		block, input = pem.Decode(input)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			cert = pem.EncodeToMemory(block)
		} else if block.Type == "PRIVATE KEY" || strings.HasSuffix(block.Type, " PRIVATE KEY") {
			if x509.IsEncryptedPEMBlock(block) {
				err = errors.New("private key PEM appears to be encrypted")
				break
			}
			key = pem.EncodeToMemory(block)
		} else {
			err = fmt.Errorf("unrecognized PEM type: %q", block.Type)
			break
		}
	}
	return
}

// StorePushCertHandler reads a PEM-encoded certificate and private
// key from the HTTP body and saves it to storage. This effectively
// enables us to do something like:
// "% cat push.pem push.key | curl -T - http://api.example.com/" to
// upload our push certs.
func StorePushCertHandler(storage storage.PushCertStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		b, err := mdmhttp.ReadAllAndReplaceBody(r)
		if err != nil {
			logger.Info("msg", "reading body", "err", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		certPEM, keyPEM, err := readPEMCertAndKey(b)
		if err == nil {
			// sanity check the provided cert and key to make sure they're usable as a pair.
			_, err = tls.X509KeyPair(certPEM, keyPEM)
		}
		var cert *x509.Certificate
		if err == nil {
			cert, err = cryptoutil.DecodePEMCertificate(certPEM)
		}
		var topic string
		if err == nil {
			topic, err = cryptoutil.TopicFromCert(cert)
		}
		if err == nil {
			err = storage.StorePushCert(r.Context(), certPEM, keyPEM)
		}
		output := &struct {
			Error    string    `json:"error,omitempty"`
			Topic    string    `json:"topic,omitempty"`
			NotAfter time.Time `json:"not_after,omitempty"`
		}{
			Topic: topic,
		}
		if cert != nil {
			output.NotAfter = cert.NotAfter
		}
		if err != nil {
			logger.Info("msg", "store push cert", "err", err)
			output.Error = err.Error()
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			logger.Debug("msg", "stored push cert", "topic", topic)
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

// logAndJSONError is a helper for both logging and outputting errors in JSON.
func logAndJSONError(logger log.Logger, w http.ResponseWriter, msg string, inErr error, header int) {
	logger.Info("msg", msg, "err", inErr)

	if header < 1 {
		header = http.StatusInternalServerError
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(header)

	type jsonError struct {
		Error string `json:"error"`
	}

	out := &jsonError{Error: inErr.Error()}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	err := enc.Encode(out)
	if err != nil && logger != nil {
		logger.Info("msg", "encoding json", "err", err)
	}
}
