package escrowkeyunlock

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	nanohttp "github.com/micromdm/nanomdm/http"
	"github.com/micromdm/nanomdm/storage"
)

const EscrowKeyUnlockURL = "https://deviceservices-external.apple.com/deviceservicesworkers/escrowKeyUnlock"

// DoEscrowKeyUnlock sends an "escrow key unlock" request to Apple.
// Mutual TLS authentication against the endpoint uses the APNs TLS keypair identified by topic in store.
// Required parameters must be supplied in queryParams and formParams: see [EscrowKeyUnlockParams].
// Caller is responsible for reading and closing HTTP response.
// See https://developer.apple.com/documentation/devicemanagement/creating-and-using-bypass-codes
func DoEscrowKeyUnlock(ctx context.Context, store storage.PushCertStore, topic string, client *http.Client, queryParams, formParams url.Values) (*http.Response, error) {
	if store == nil {
		panic("nil store")
	}
	if queryParams == nil || formParams == nil {
		panic("nil params")
	}

	pemCert, pemKey, _, err := store.RetrievePushCert(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("retrieve push cert for topic: %s: %w", topic, err)
	}

	tlsCert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, fmt.Errorf("parse key pair: %w", err)
	}

	client, err = nanohttp.ClientWithCert(client, &tlsCert)
	if err != nil {
		return nil, fmt.Errorf("adapting client for mTLS: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		EscrowKeyUnlockURL+"?"+queryParams.Encode(),
		strings.NewReader(formParams.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return client.Do(req)
}
