package block

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/storage"
)

const (
	testUDID   = "test-udid"
	testUserID = "test-user-id"
)

// errNext is returned by every method of nextService so tests can tell
// whether a request was forwarded.
var errNext = errors.New("forwarded to next")

type nextService struct{}

func (s *nextService) Authenticate(*mdm.Request, *mdm.Authenticate) error { return errNext }
func (s *nextService) TokenUpdate(*mdm.Request, *mdm.TokenUpdate) error   { return errNext }
func (s *nextService) CheckOut(*mdm.Request, *mdm.CheckOut) error         { return errNext }
func (s *nextService) SetBootstrapToken(*mdm.Request, *mdm.SetBootstrapToken) error {
	return errNext
}
func (s *nextService) GetBootstrapToken(*mdm.Request, *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	return nil, errNext
}
func (s *nextService) UserAuthenticate(*mdm.Request, *mdm.UserAuthenticate) ([]byte, error) {
	return nil, errNext
}
func (s *nextService) DeclarativeManagement(*mdm.Request, *mdm.DeclarativeManagement) ([]byte, error) {
	return nil, errNext
}
func (s *nextService) GetToken(*mdm.Request, *mdm.GetToken) (*mdm.GetTokenResponse, error) {
	return nil, errNext
}
func (s *nextService) CommandAndReportResults(*mdm.Request, *mdm.CommandResults) (*mdm.Command, error) {
	return nil, errNext
}

// memStore is a simple in-memory BlockStore.
type memStore struct {
	blocks map[string]*storage.Block
	err    error
}

func newMemStore() *memStore {
	return &memStore{blocks: make(map[string]*storage.Block)}
}

func (s *memStore) RetrieveBlock(r *mdm.Request) (*storage.Block, error) {
	return s.blocks[r.ID], s.err
}

func (s *memStore) StoreBlock(r *mdm.Request, block *storage.Block) error {
	s.blocks[r.ID] = block
	return s.err
}

func (s *memStore) RemoveBlock(r *mdm.Request) error {
	delete(s.blocks, r.ID)
	return s.err
}

// call invokes the svc method named by kind with an enrollment e.
func call(svc service.CheckinAndCommandService, kind string, e mdm.Enrollment) error {
	r := mdm.NewRequestWithContext(context.Background(), nil)
	var err error
	switch kind {
	case "Authenticate":
		err = svc.Authenticate(r, &mdm.Authenticate{Enrollment: e})
	case "TokenUpdate":
		err = svc.TokenUpdate(r, &mdm.TokenUpdate{Enrollment: e})
	case "CheckOut":
		err = svc.CheckOut(r, &mdm.CheckOut{Enrollment: e})
	case "UserAuthenticate":
		_, err = svc.UserAuthenticate(r, &mdm.UserAuthenticate{Enrollment: e})
	case "SetBootstrapToken":
		err = svc.SetBootstrapToken(r, &mdm.SetBootstrapToken{Enrollment: e})
	case "GetBootstrapToken":
		_, err = svc.GetBootstrapToken(r, &mdm.GetBootstrapToken{Enrollment: e})
	case "DeclarativeManagement":
		_, err = svc.DeclarativeManagement(r, &mdm.DeclarativeManagement{Enrollment: e})
	case "GetToken":
		_, err = svc.GetToken(r, &mdm.GetToken{Enrollment: e})
	case "CommandAndReportResults":
		_, err = svc.CommandAndReportResults(r, &mdm.CommandResults{Enrollment: e})
	default:
		panic("unknown kind: " + kind)
	}
	return err
}

var kinds = []string{
	"Authenticate",
	"TokenUpdate",
	"CheckOut",
	"UserAuthenticate",
	"SetBootstrapToken",
	"GetBootstrapToken",
	"DeclarativeManagement",
	"GetToken",
	"CommandAndReportResults",
}

// expectStatus checks that err is an HTTP status error of status,
// or was forwarded to next if status is zero.
func expectStatus(t *testing.T, err error, status int) {
	t.Helper()
	if status == 0 {
		if !errors.Is(err, errNext) {
			t.Errorf("expected forward to next, have: %v", err)
		}
		return
	}
	var httpErr *service.HTTPStatusError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected HTTP status error, have: %v", err)
	}
	if have, want := httpErr.Status, status; have != want {
		t.Errorf("status: have: %d, want: %d", have, want)
	}
	if !errors.Is(err, ErrBlocked) {
		t.Errorf("expected ErrBlocked, have: %v", err)
	}
}

func TestNotBlocked(t *testing.T) {
	svc := New(&nextService{}, newMemStore())
	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			expectStatus(t, call(svc, kind, mdm.Enrollment{UDID: testUDID}), 0)
		})
	}
}

func TestStyles(t *testing.T) {
	for _, tc := range []struct {
		style storage.BlockStyle
		// want returns the expected status for kind; 0 is forwarded
		want func(kind string) int
	}{
		{
			style: storage.StyleUnenroll,
			want: func(kind string) int {
				if kind == "TokenUpdate" {
					return 0
				}
				return http.StatusUnauthorized
			},
		},
		{
			style: storage.StyleServiceUnavailable,
			want: func(kind string) int {
				if kind == "TokenUpdate" {
					return 0
				}
				return http.StatusServiceUnavailable
			},
		},
		{
			style: storage.StyleUnrecognizedDevice,
			want: func(kind string) int {
				if kind == "TokenUpdate" {
					return 0
				}
				return http.StatusForbidden
			},
		},
		{
			style: storage.StyleCommandQueue,
			want: func(kind string) int {
				if kind == "CommandAndReportResults" {
					return http.StatusServiceUnavailable
				}
				return 0
			},
		},
	} {
		t.Run(string(tc.style), func(t *testing.T) {
			store := newMemStore()
			store.blocks[testUDID] = &storage.Block{Style: tc.style}
			svc := New(&nextService{}, store)
			for _, kind := range kinds {
				t.Run(kind, func(t *testing.T) {
					expectStatus(t, call(svc, kind, mdm.Enrollment{UDID: testUDID}), tc.want(kind))
				})
			}
			if store.blocks[testUDID] == nil {
				t.Error("block should not have been cleared")
			}
		})
	}
}

func TestUserChannel(t *testing.T) {
	store := newMemStore()
	store.blocks[testUDID] = &storage.Block{Style: storage.StyleUnenroll}
	svc := New(&nextService{}, store)

	// a user channel of a blocked device is blocked, too
	err := call(svc, "CommandAndReportResults", mdm.Enrollment{UDID: testUDID, UserID: testUserID})
	expectStatus(t, err, http.StatusUnauthorized)

	// another device is not blocked
	err = call(svc, "CommandAndReportResults", mdm.Enrollment{UDID: "other-udid", UserID: testUserID})
	expectStatus(t, err, 0)
}

func TestClearOnAuthenticate(t *testing.T) {
	store := newMemStore()
	store.blocks[testUDID] = &storage.Block{Style: storage.StyleUnenroll, ClearOnAuthenticate: true}
	svc := New(&nextService{}, store)

	// blocked until re-enrollment
	expectStatus(t, call(svc, "CommandAndReportResults", mdm.Enrollment{UDID: testUDID}), http.StatusUnauthorized)

	// Authenticate clears the block and is forwarded
	expectStatus(t, call(svc, "Authenticate", mdm.Enrollment{UDID: testUDID}), 0)
	if store.blocks[testUDID] != nil {
		t.Error("block should have been cleared")
	}

	// no longer blocked
	expectStatus(t, call(svc, "CommandAndReportResults", mdm.Enrollment{UDID: testUDID}), 0)
}

// TestStoreErrorFailsOpen checks that a store error forwards every request,
// even for a blocked enrollment: a storage outage must not block the fleet.
func TestStoreErrorFailsOpen(t *testing.T) {
	store := newMemStore()
	store.blocks[testUDID] = &storage.Block{Style: storage.StyleUnenroll}
	store.err = errors.New("store error")
	svc := New(&nextService{}, store)

	for _, kind := range kinds {
		t.Run(kind, func(t *testing.T) {
			expectStatus(t, call(svc, kind, mdm.Enrollment{UDID: testUDID}), 0)
		})
	}
}

// TestClearOnAuthenticateRemoveErrorFailsOpen checks that Authenticate is
// still forwarded when the block can't be removed.
func TestClearOnAuthenticateRemoveErrorFailsOpen(t *testing.T) {
	store := &removeErrStore{memStore: newMemStore()}
	store.blocks[testUDID] = &storage.Block{Style: storage.StyleUnenroll, ClearOnAuthenticate: true}
	svc := New(&nextService{}, store)

	expectStatus(t, call(svc, "Authenticate", mdm.Enrollment{UDID: testUDID}), 0)
}

// removeErrStore fails only RemoveBlock.
type removeErrStore struct{ *memStore }

func (s *removeErrStore) RemoveBlock(*mdm.Request) error { return errors.New("remove error") }

// TestInvalidEnrollmentFailsOpen checks that an enrollment that can't be
// normalized is forwarded; validating it is left to the next service.
func TestInvalidEnrollmentFailsOpen(t *testing.T) {
	svc := New(&nextService{}, newMemStore())
	expectStatus(t, call(svc, "CommandAndReportResults", mdm.Enrollment{}), 0)
}
