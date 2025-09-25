// Package webhook is a NanoMDM service for sending HTTP webhook events.
package webhook

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
)

type Doer interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

// Webhook is a NanoMDM service for sending HTTP webhook events.
type Webhook struct {
	url   string
	doer  Doer
	store storage.TokenUpdateTallyStore
	nowFn func() time.Time
}

// New initializes a new [Webhook].
// The store can be nil.
func New(url string, store storage.TokenUpdateTallyStore) *Webhook {
	return &Webhook{
		url:   url,
		doer:  http.DefaultClient,
		store: store,
		nowFn: func() time.Time { return time.Now() },
	}
}

// updateCheckinIDs type converts the enrollment IDs to their schema counterparts and assigns to the event struct.
func (ev *CheckinEvent) updateCheckinIDs(udid, enrollmentID string) {
	if udid != "" {
		id := UDID(udid)
		ev.Udid = &id
	}
	if enrollmentID != "" {
		id := EnrollmentID(enrollmentID)
		ev.EnrollmentId = &id
	}
}

// b64 merely encodes src to [RawPayload] as base64.
// It's a helper since our JSON schema generator does not turn our
// field into a byte slice for us (and instead a typed string).
func b64(src []byte) RawPayload {
	return RawPayload(base64.StdEncoding.EncodeToString(src))
}

// Authenticate sends a webhook event of the NanoMDM Authenticate check-in message.
func (w *Webhook) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmAuthenticate,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// TokenUpdate sends a webhook event of the NanoMDM TokenUpdate check-in message.
func (w *Webhook) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmTokenUpdate,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	if w.store != nil {
		tally, err := w.store.RetrieveTokenUpdateTally(r.Context(), r.ID)
		if err != nil {
			return err
		}
		ev.CheckinEvent.TokenUpdateTally = &tally
	}
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// CheckOut sends a webhook event of the NanoMDM CheckOut check-in message.
func (w *Webhook) CheckOut(r *mdm.Request, m *mdm.CheckOut) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmCheckOut,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)

	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// UserAuthenticate sends a webhook event of the NanoMDM UserAuthenticate check-in message.
func (w *Webhook) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmUserAuthenticate,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// SetBootstrapToken sends a webhook event of the NanoMDM SetBootstrapToken check-in message.
func (w *Webhook) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmSetBootstrapToken,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// GetBootstrapToken sends a webhook event of the NanoMDM GetBootstrapToken check-in message.
func (w *Webhook) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmGetBootstrapToken,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// CommandAndReportResults sends a webhook event of the NanoMDM command results.
func (w *Webhook) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmConnect,
		CreatedAt: w.nowFn(),
		AcknowledgeEvent: &AcknowledgeEvent{
			Status:      results.Status,
			CommandUuid: &results.CommandUUID,
			RawPayload:  b64(results.Raw),
			UrlParams:   r.Params,
		},
	}
	if results.UDID != "" {
		id := UDID(results.UDID)
		ev.AcknowledgeEvent.Udid = &id
	}
	if results.EnrollmentID != "" {
		id := EnrollmentID(results.EnrollmentID)
		ev.AcknowledgeEvent.EnrollmentId = &id
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// DeclarativeManagement sends a webhook event of the NanoMDM DeclarativeManagement check-in message.
func (w *Webhook) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmDeclarativeManagement,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// GetToken sends a webhook event of the NanoMDM GetToken check-in message.
func (w *Webhook) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	ev := &EventJson{
		Topic:     EventJsonTopicMdmGetToken,
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			RawPayload: b64(m.Raw),
			UrlParams:  r.Params,
		},
	}
	ev.CheckinEvent.updateCheckinIDs(m.UDID, m.EnrollmentID)
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}
