// Package webhook is a NanoMDM service for sending HTTP webhook events.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"net/http"
	"time"

	"github.com/micromdm/nanolib/http/trace"
	"github.com/micromdm/nanomdm/http/hashbody"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

type Doer interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

const (
	// ContentType used for all requests.
	ContentType = "application/json; charset=utf-8"

	// HTTP header name used when including HMAC signatures.
	HMACHeader = "X-Hmac-Signature"
)

// stringPtr converts s to a pointer to T.
// If s is empty a nil pointer is returned.
func stringPtr[T ~string](s string) *T {
	if s == "" {
		return nil
	}
	tmp := T(s)
	return &tmp
}

// b64 merely encodes src to [RawPayload] as base64.
// It's a helper since our JSON schema generator does not turn our
// field into a byte slice for us (and instead a typed string).
func b64(src []byte) RawPayload {
	return RawPayload(base64.StdEncoding.EncodeToString(src))
}

// ids is a helper to convert from request to schema types.
func ids(eid *mdm.EnrollID) *IDs {
	return &IDs{
		Id:       eid.ID,
		ParentId: stringPtr[string](eid.ParentID),
		Type:     IDsType(eid.Type.String()),
	}
}

// Webhook is a NanoMDM service for sending HTTP webhook events.
type Webhook struct {
	url   string
	doer  Doer
	store storage.TokenUpdateTallyStore
	nowFn func() time.Time
}

// Options configure webhook services.
type Option func(*Webhook)

// WithTokenUpdateTalley specifies a storage backend to retrieve the "token talley" from.
// This permits sending the token talley in the event to the webhook.
func WithTokenUpdateTalley(store storage.TokenUpdateTallyStore) Option {
	return func(w *Webhook) {
		w.store = store
	}
}

// WithClient configures an HTTP client to use when sending webhooks.
func WithClient(doer Doer) Option {
	return func(w *Webhook) {
		w.doer = doer
	}
}

// WithHMACSecret will add a SHA-256 HMAC of the webhook HTTP body using key.
// The HMAC is provided in the [HMACHeader] header and is Base-64 encoded.
func WithHMACSecret(key []byte) Option {
	return func(w *Webhook) {
		w.doer = hashbody.NewSetBodyHashClient(
			w.doer,
			HMACHeader,
			func() hash.Hash {
				return hmac.New(sha256.New, key)
			},
			base64.StdEncoding.EncodeToString,
		)
	}
}

// New initializes a new [Webhook] sending events to url.
func New(url string, opts ...Option) *Webhook {
	w := &Webhook{
		url:   url,
		doer:  http.DefaultClient,
		nowFn: func() time.Time { return time.Now() },
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// send the HTTP request to the webhook URL.
func (w *Webhook) send(ctx context.Context, event *EventJson) error {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", ContentType)

	resp, err := w.doer.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	return nil
}

// Authenticate sends a webhook event of the NanoMDM Authenticate check-in message.
func (w *Webhook) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmAuthenticate,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return w.send(r.Context(), ev)
}

// TokenUpdate sends a webhook event of the NanoMDM TokenUpdate check-in message.
func (w *Webhook) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmTokenUpdate,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	if w.store != nil {
		tally, err := w.store.RetrieveTokenUpdateTally(r.Context(), r.ID)
		if err != nil {
			return err
		}
		ev.CheckinEvent.TokenUpdateTally = &tally
	}
	return w.send(r.Context(), ev)
}

// CheckOut sends a webhook event of the NanoMDM CheckOut check-in message.
func (w *Webhook) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmCheckOut,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return w.send(r.Context(), ev)
}

// UserAuthenticate sends a webhook event of the NanoMDM UserAuthenticate check-in message.
func (w *Webhook) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmUserAuthenticate,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return nil, w.send(r.Context(), ev)
}

// SetBootstrapToken sends a webhook event of the NanoMDM SetBootstrapToken check-in message.
func (w *Webhook) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmSetBootstrapToken,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return w.send(r.Context(), ev)
}

// GetBootstrapToken sends a webhook event of the NanoMDM GetBootstrapToken check-in message.
func (w *Webhook) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmGetBootstrapToken,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return nil, w.send(r.Context(), ev)
}

// CommandAndReportResults sends a webhook event of the NanoMDM command results.
func (w *Webhook) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmConnect,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		AcknowledgeEvent: &AcknowledgeEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](results.EnrollmentID),
			Udid:         stringPtr[UDID](results.UDID),
			Status:       results.Status,
			CommandUuid:  stringPtr[string](results.CommandUUID),
			RawPayload:   b64(results.Raw),
			UrlParams:    r.Params,
		},
	}
	return nil, w.send(r.Context(), ev)
}

// DeclarativeManagement sends a webhook event of the NanoMDM DeclarativeManagement check-in message.
func (w *Webhook) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmDeclarativeManagement,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return nil, w.send(r.Context(), ev)
}

// GetToken sends a webhook event of the NanoMDM GetToken check-in message.
func (w *Webhook) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmGetToken,
		CreatedAt: w.nowFn(),
		EventId:   stringPtr[string](trace.GetTraceID(r.Context())),
		CheckinEvent: &CheckinEvent{
			Ids:          ids(r.EnrollID),
			EnrollmentId: stringPtr[EnrollmentID](m.EnrollmentID),
			Udid:         stringPtr[UDID](m.UDID),
			RawPayload:   b64(m.Raw),
			UrlParams:    r.Params,
		},
	}
	return nil, w.send(r.Context(), ev)
}
