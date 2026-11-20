package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// ErrNotBlocked is returned when retrieving a block for an enrollment that is not blocked.
var ErrNotBlocked = errors.New("enrollment not blocked")

// blockRequest creates an MDM request for the enrollment ID in the URL path of r.
func blockRequest(r *http.Request) (*mdm.Request, error) {
	if r.URL.Path == "" {
		return nil, errors.New("empty path")
	}
	m := mdm.NewRequestWithContext(r.Context(), nil)
	m.EnrollID = &mdm.EnrollID{ID: r.URL.Path}
	return m, nil
}

// BlockHandler retrieves (GET), stores (PUT), and removes (DELETE)
// the block of an enrollment.
//
// Note the whole URL path is used as the enrollment ID. This
// probably necessitates stripping the URL prefix before using. The
// enrollment ID should be a device channel ID.
func BlockHandler(store storage.BlockStore, logger log.Logger) http.HandlerFunc {
	if store == nil {
		panic("nil store")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)

		m, err := blockRequest(r)
		if err != nil {
			logAndWriteJSONError(logger, w, "getting enrollment id", err, http.StatusBadRequest)
			return
		}
		logger = logger.With("id", m.ID)

		switch r.Method {
		case http.MethodGet:
			block, err := store.RetrieveBlock(m)
			if err != nil {
				logAndWriteJSONError(logger, w, "retrieving block", err, http.StatusInternalServerError)
				return
			} else if block == nil {
				logAndWriteJSONError(logger, w, "retrieving block", ErrNotBlocked, http.StatusNotFound)
				return
			}
			writeJSON(w, block, http.StatusOK, logger)
		case http.MethodPut:
			block := new(storage.Block)
			if err = json.NewDecoder(r.Body).Decode(block); err != nil {
				err = fmt.Errorf("decoding block: %w", err)
				logAndWriteJSONError(logger, w, "decoding block", err, http.StatusBadRequest)
				return
			}
			if err = block.Validate(); err != nil {
				logAndWriteJSONError(logger, w, "validating block", err, http.StatusBadRequest)
				return
			}
			if err = store.StoreBlock(m, block); err != nil {
				logAndWriteJSONError(logger, w, "storing block", err, http.StatusInternalServerError)
				return
			}
			logger.Info("msg", "stored block", "style", block.Style, "clear_on_authenticate", block.ClearOnAuthenticate)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if err = store.RemoveBlock(m); err != nil {
				logAndWriteJSONError(logger, w, "removing block", err, http.StatusInternalServerError)
				return
			}
			logger.Info("msg", "removed block")
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}
