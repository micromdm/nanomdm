package api

import (
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
	"github.com/micromdm/nanomdm/http/escrowkeyunlock"
	"github.com/micromdm/nanomdm/storage"
)

func fillUnlockParams(v url.Values) *escrowkeyunlock.EscrowKeyUnlockParams {
	return &escrowkeyunlock.EscrowKeyUnlockParams{
		Serial:      v.Get("serial"),
		IMEI:        v.Get("imei"),
		IMEI2:       v.Get("imei2"),
		MEID:        v.Get("meid"),
		ProductType: v.Get("productType"),
		OrgName:     v.Get("orgName"),
		GUID:        v.Get("guid"),
		EscrowKey:   v.Get("escrowKey"),
	}
}

// The default HTTP client will be used if client is nil.
func NewEscrowKeyUnlockHandler(store storage.PushCertStore, client *http.Client, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)
		r.ParseForm()
		values := r.PostForm
		params := fillUnlockParams(values)

		if !params.Valid() {
			err := errors.New("invalid or missing parameters")
			logAndJSONError(logger, w, "validating parameters", err, http.StatusBadRequest)
			return
		}

		topic := values.Get("topic")
		if topic == "" {
			err := errors.New("empty topic")
			logAndJSONError(logger, w, "validating parameters", err, http.StatusBadRequest)
			return
		}

		resp, err := escrowkeyunlock.DoEscrowKeyUnlock(
			r.Context(),
			store,
			topic,
			nil,
			params.QueryParams(),
			params.FormParams(),
		)
		if err != nil {
			logAndJSONError(logger, w, "escrow key unlock", err, 0)
			return
		}
		defer resp.Body.Close()

		// copy status
		w.WriteHeader(resp.StatusCode)

		// copy headers
		for k, values := range resp.Header {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}

		logger.Debug(
			"msg", "escrow key unlock",
			"serial", params.Serial,
			"http_status", resp.StatusCode,
		)

		// copy body
		_, err = io.Copy(w, resp.Body)
		if err != nil {
			logger.Info("msg", "copying body", "err", err)
		}
	}
}
