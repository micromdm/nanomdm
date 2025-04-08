package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanomdm/cryptoutil"
	mdmhttp "github.com/micromdm/nanomdm/http"
	httpapi "github.com/micromdm/nanomdm/http/api"
	httpmdm "github.com/micromdm/nanomdm/http/mdm"
	"github.com/micromdm/nanomdm/service"
	"github.com/micromdm/nanomdm/service/certauth"
	"github.com/micromdm/nanomdm/service/nanomdm"
	"github.com/micromdm/nanomdm/storage"
)

const (
	serverURL   = "/mdm"
	apiPrefix   = "/test/v1"
	enqueueURL  = apiPrefix + "/enqueue/"
	pushCertURl = apiPrefix + "/pushcert"
)

// setupNanoMDM configures normal-ish NanoMDM HTTP server handlers for testing.
func setupNanoMDM(logger log.Logger, store storage.AllStorage) (http.Handler, error) {
	// begin with the primary NanoMDM service
	var svc service.CheckinAndCommandService = nanomdm.New(store, nanomdm.WithLogger(logger))

	// chain the certificate auth middleware
	svc = certauth.New(svc, store, certauth.WithLogger(logger))

	mux := http.NewServeMux()
	mdmMux := mdmhttp.NewMWMux(mux)

	// setup certificate extraction
	// note missing auth for tests
	mdmMux.Use(func(h http.Handler) http.Handler {
		return httpmdm.CertExtractMdmSignatureMiddleware(h, httpmdm.MdmSignatureVerifierFunc(cryptoutil.VerifyMdmSignature))
	})

	// setup MDM (check-in and command) handlers
	// note missing auth for tests
	mdmMux.Handle(
		serverURL,
		httpmdm.CheckinAndCommandHandler(svc, logger.With("handler", "mdm")),
	)

	// setup API handlers
	httpapi.HandleAPIv1("/test/v1", mux, logger, store, nil)

	return mux, nil
}

type IDer interface {
	ID() string
}

func TestE2E(t *testing.T, ctx context.Context, store storage.AllStorage) {
	var logger log.Logger = log.NopLogger // stdlogfmt.New(stdlogfmt.WithDebugFlag(true))

	mux, err := setupNanoMDM(logger, store)
	if err != nil {
		t.Fatal(err)
	}

	// create a fake HTTP client that dispatches to our raw handlers
	c := NewHandlerClient(mux)

	t.Run("pushcert", func(t *testing.T) { pushcert(t, ctx, &api{doer: c, urlPushCert: pushCertURl}, store) })

	// create our new device for testing
	d, err := newDeviceFromCheckins(
		c,
		serverURL,
		"../../mdm/testdata/Authenticate.2.plist",
		"../../mdm/testdata/TokenUpdate.2.plist",
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("certauth", func(t *testing.T) { certAuth(t, ctx, store) })
	t.Run("certauth-retro", func(t *testing.T) { certAuthRetro(t, ctx, store) })

	// regression test for retrieving push info of missing devices.
	t.Run("invalid-pushinfo", func(t *testing.T) {
		_, err := store.RetrievePushInfo(ctx, []string{"INVALID"})
		if err != nil {
			// should NOT recieve a "global" error for an enrollment that
			// is merely invalid (or not enrolled yet, or not fully enrolled)
			t.Errorf("should NOT have errored: %v", err)
		}
	})

	t.Run("enroll", func(t *testing.T) { enroll(t, ctx, d, store) })

	t.Run("tally", func(t *testing.T) { tally(t, ctx, d, store, 1) })

	t.Run("bstoken", func(t *testing.T) { bstoken(t, ctx, d.Enrollment) })

	// re-enroll device
	// this is to try and catch any leftover crud that a storage backend didn't
	// clean up (like the tally count, BS token, etc.)
	t.Run("re-enroll", func(t *testing.T) {
		err = d.DoEnroll(ctx)
		if err != nil {
			t.Fatal(fmt.Errorf("re-enrolling device %s: %w", d.ID(), err))
		}
	})

	t.Run("tally-after-reenroll", func(t *testing.T) { tally(t, ctx, d, store, 1) })

	t.Run("bstoken-after-reenroll", func(t *testing.T) { bstoken(t, ctx, d.Enrollment) })

	t.Run("clear-queue", func(t *testing.T) {
		err = store.ClearQueue(d.NewMDMRequest(ctx))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("queue", func(t *testing.T) { queue(t, ctx, d, &api{doer: c}, store) })

	t.Run("migrate", func(t *testing.T) { migrate(t, ctx, store, d) })
}
