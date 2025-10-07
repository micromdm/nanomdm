package e2e

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/storage"
	"github.com/micromdm/nanomdm/test"
)

type pushCertUploader interface {
	PushCert(ctx context.Context, pemCert, pemKey []byte) error
}

func pushcert(t *testing.T, ctx context.Context, a pushCertUploader, store storage.PushCertStore) {
	pemCert, err := os.ReadFile("../../test/e2e/testdata/push.pem")
	if err != nil {
		t.Fatal(err)
	}

	pushTmpl, err := cryptoutil.DecodePEMCertificate(pemCert)
	if err != nil {
		t.Fatal(err)
	}
	pushTmpl.PublicKey = nil

	key, cert, err := test.SelfSignedCertRSAResigner(pushTmpl)
	if err != nil {
		t.Fatal(err)
	}
	pemCert = cryptoutil.PEMCertificate(cert.Raw)

	pemKey := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		t.Fatal(err)
	}

	err = a.PushCert(ctx, pemCert, pemKey)
	if err != nil {
		t.Fatal(err)
	}

	tlsCert, staleToken1, err := store.RetrievePushCert(ctx, topic)
	if err != nil {
		t.Fatal(err)
	}

	if have, want := len(tlsCert.Certificate), 1; have != want {
		t.Fatalf("have: %d, want: %d", have, want)
	}

	pemCert2 := cryptoutil.PEMCertificate(tlsCert.Certificate[0])

	if strings.TrimSpace(string(pemCert)) != strings.TrimSpace(string(pemCert2)) {
		t.Error("mismatched certs")
	}

	err = store.StorePushCert(ctx, pemCert, pemKey)
	if err != nil {
		t.Fatal(err)
	}

	_, staleToken2, err := store.RetrievePushCert(ctx, topic)
	if err != nil {
		t.Fatal(err)
	}

	if staleToken1 == staleToken2 {
		t.Error("stale tokens should not match after storing twice")
	}

}
