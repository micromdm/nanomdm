// Package webhook is a NanoMDM service for sending HTTP webhook events.
package webhook

import (
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

// Authenticate sends a webhook event of the NanoMDM Authenticate check-in message.
func (w *Webhook) Authenticate(r *mdm.Request, m *mdm.Authenticate) error {
	ev := &Event{
		Topic:     "mdm.Authenticate",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// TokenUpdate sends a webhook event of the NanoMDM TokenUpdate check-in message.
func (w *Webhook) TokenUpdate(r *mdm.Request, m *mdm.TokenUpdate) error {
	ev := &Event{
		Topic:     "mdm.TokenUpdate",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
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
	ev := &Event{
		Topic:     "mdm.CheckOut",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// UserAuthenticate sends a webhook event of the NanoMDM UserAuthenticate check-in message.
func (w *Webhook) UserAuthenticate(r *mdm.Request, m *mdm.UserAuthenticate) ([]byte, error) {
	ev := &Event{
		Topic:     "mdm.UserAuthenticate",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// SetBootstrapToken sends a webhook event of the NanoMDM SetBootstrapToken check-in message.
func (w *Webhook) SetBootstrapToken(r *mdm.Request, m *mdm.SetBootstrapToken) error {
	ev := &Event{
		Topic:     "mdm.SetBootstrapToken",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// GetBootstrapToken sends a webhook event of the NanoMDM GetBootstrapToken check-in message.
func (w *Webhook) GetBootstrapToken(r *mdm.Request, m *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	ev := &Event{
		Topic:     "mdm.GetBootstrapToken",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// CommandAndReportResults sends a webhook event of the NanoMDM command results.
func (w *Webhook) CommandAndReportResults(r *mdm.Request, results *mdm.CommandResults) (*mdm.Command, error) {
	ev := &Event{
		Topic:     "mdm.Connect",
		CreatedAt: w.nowFn(),
		AcknowledgeEvent: &AcknowledgeEvent{
			UDID:         results.UDID,
			EnrollmentID: results.EnrollmentID,
			Status:       results.Status,
			CommandUUID:  results.CommandUUID,
			RawPayload:   results.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// DeclarativeManagement sends a webhook event of the NanoMDM DeclarativeManagement check-in message.
func (w *Webhook) DeclarativeManagement(r *mdm.Request, m *mdm.DeclarativeManagement) ([]byte, error) {
	ev := &Event{
		Topic:     "mdm.DeclarativeManagement",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}

// GetToken sends a webhook event of the NanoMDM GetToken check-in message.
func (w *Webhook) GetToken(r *mdm.Request, m *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	ev := &Event{
		Topic:     "mdm.GetToken",
		CreatedAt: w.nowFn(),
		CheckinEvent: &CheckinEvent{
			UDID:         m.UDID,
			EnrollmentID: m.EnrollmentID,
			RawPayload:   m.Raw,
			Params:       r.Params,
		},
	}
	return nil, postWebhookEvent(r.Context(), w.doer, w.url, ev)
}
