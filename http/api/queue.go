package api

import (
	"net/http"
	"strings"

	nanoapi "github.com/micromdm/nanomdm/api"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// QueueHandler handles GET and DELETE requests for the queue API.
// GET retrieves queued commands for an enrollment ID.
// DELETE clears queued commands for one or more enrollment IDs.
//
// Note the whole URL path is used as the identifier. This
// probably necessitates stripping the URL prefix before using.
func QueueHandler(store storage.CommandQueueAPIStore, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := ctxlog.Logger(r.Context(), logger)

		switch r.Method {
		case http.MethodGet:
			handleQueueGet(w, r, store, logger)
		case http.MethodDelete:
			handleQueueDelete(w, r, store, logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
}

// handleQueueGet retrieves queued commands for an enrollment ID.
func handleQueueGet(w http.ResponseWriter, r *http.Request, store storage.CommandQueueAPIStore, logger log.Logger) {
	id := r.URL.Path
	if id == "" {
		logAndWriteJSONError(logger, w, "missing enrollment id", nil, http.StatusBadRequest)
		return
	}

	// Parse pagination from query parameters
	pagination := parsePaginationFromQuery(r)

	// Validate pagination
	if pagination != nil {
		if err := pagination.ValidErr(); err != nil {
			logAndWriteJSONError(logger, w, "invalid pagination", err, http.StatusBadRequest)
			return
		}
	}

	query := &storage.QueueQuery{
		ID:         id,
		Pagination: pagination,
	}

	result, err := store.RetrieveQueuedCommands(r.Context(), query)
	if err != nil {
		logAndWriteJSONError(logger, w, "retrieving queued commands", err, http.StatusInternalServerError)
		return
	}

	writeJSON(w, result, http.StatusOK, logger)
}

// handleQueueDelete clears queued commands for one or more enrollment IDs.
func handleQueueDelete(w http.ResponseWriter, r *http.Request, store storage.CommandQueueAPIStore, logger log.Logger) {
	idsRaw := r.URL.Path
	if idsRaw == "" {
		logAndWriteJSONError(logger, w, "missing enrollment id(s)", nil, http.StatusBadRequest)
		return
	}

	ids := strings.Split(idsRaw, ",")

	result := &nanoapi.QueueAPIResult{
		Status: make(map[string]*nanoapi.Error),
	}

	var failCount int
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if err := store.ClearQueueByID(r.Context(), id); err != nil {
			result.Status[id] = nanoapi.NewError(err)
			failCount++
		}
	}

	// Determine HTTP status based on results
	header := http.StatusNoContent
	if failCount > 0 && failCount < len(ids) {
		// Some failed
		header = http.StatusMultiStatus
	} else if failCount > 0 {
		// All failed
		header = http.StatusInternalServerError
	}

	if header == http.StatusNoContent {
		w.WriteHeader(header)
		return
	}

	writeJSON(w, result, header, logger)
}
