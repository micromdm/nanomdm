package api

import (
	"net/http"
	"strings"

	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"

	"github.com/micromdm/nanolib/log"
)

const (
	APIEndpointPushCert        = "/pushcert"
	APIEndpointPush            = "/push/"    // note trailing slash
	APIEndpointEnqueue         = "/enqueue/" // note trailing slash
	APIEndpointEscrowKeyUnlock = "/escrowkeyunlock"
)

// Mux can register HTTP handlers.
type Mux interface {
	// Handle registers the handler for the given pattern.
	// It is assumed pattern operates similar to http.ServeMux with
	// respect to "trailing slash" behavior.
	Handle(pattern string, handler http.Handler)
}

// APIStorage is required for the API handlers.
type APIStorage interface {
	storage.PushCertStore
	storage.PushCertStorer
	storage.CommandEnqueuer
}

func handlerName(endpoint string) string {
	return strings.Trim(endpoint, "/")
}

// HandleAPIv1 registers the various API handlers into mux.
// API endpoint paths are prepended with prefix.
// Authentication or any other layered handlers are not present.
// They are assumed to be layered with mux.
// If prefix is empty and these handlers are used in sub-paths then
// handlers should have that sub-path stripped from the request.
// The logger is adorned with a "handler" key of the endpoint name.
func HandleAPIv1(prefix string, mux Mux, logger log.Logger, store APIStorage, pusher push.Pusher) {
	// register API handlers for push cert retrieval (GET) and upload (PUT)
	pushCertLogger := logger.With("handler", handlerName(APIEndpointPushCert))
	pushCertPUT := NewStorePushCertHandler(store, pushCertLogger)
	pushCertGET := NewRetrievePushCertHandler(store, pushCertLogger)
	mux.Handle(
		prefix+APIEndpointPushCert,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				pushCertPUT.ServeHTTP(w, r)
			case http.MethodGet:
				pushCertGET.ServeHTTP(w, r)
			default:
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			}
		}),
	)

	// register API handler for sending APNs push notifications
	if pusher != nil {
		mux.Handle(
			prefix+APIEndpointPush,
			http.StripPrefix( // we strip the prefix to use the path as an id
				prefix+APIEndpointPush,
				PushHandler(
					pusher,
					logger.With("handler", handlerName(APIEndpointPush)),
				),
			),
		)
	}

	// register API handler for new command enqueueing
	mux.Handle(
		prefix+APIEndpointEnqueue,
		http.StripPrefix( // we strip the prefix to use the path as an id
			prefix+APIEndpointEnqueue,
			RawCommandEnqueueHandler(
				store,
				pusher,
				logger.With("handler", handlerName(APIEndpointEnqueue)),
			),
		),
	)

	// register API handler for escrow key unlock
	mux.Handle(
		prefix+APIEndpointEscrowKeyUnlock,
		NewEscrowKeyUnlockHandler(store, nil, logger.With("handler", handlerName(APIEndpointEscrowKeyUnlock))),
	)
}
