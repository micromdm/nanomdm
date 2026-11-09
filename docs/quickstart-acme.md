# NanoMDM ACME Quickstart Guide (with NanoCA)

This guide shows how to use [NanoCA](https://github.com/brandonweeks/nanoca) as an ACME-based alternative to SCEP for MDM device identity certificates. NanoCA issues certificates using [ACME device attestation](https://www.ietf.org/archive/id/draft-acme-device-attest-03.html), where devices cryptographically prove their identity via Apple's [Managed Device Attestation](https://support.apple.com/guide/deployment/managed-device-attestation-dep28afbde6a/web) before receiving a certificate.

This replaces the SCEP server setup in the [standard quickstart guide](quickstart.md).

> **Note:** ACME enrollment profiles require **iOS/iPadOS 16+** or **macOS 13.1+**. Hardware-bound attestation requires devices with a Secure Enclave (iPhone 8 / A11+, Apple Silicon Macs).

> **Warning:** This guide is intended to get NanoMDM *working* with NanoCA and *does not represent best practices* for running internet servers.

## Requirements

- Everything from the [standard quickstart](quickstart.md#requirements) (push certificate, command-line tools, internet access)
- A [Go toolchain](https://go.dev/doc/install) (to build the NanoCA example server)
- A device with a Secure Enclave (for hardware-bound attestation)

## Build the NanoCA + NanoMDM example server

NanoCA is a Go library. It includes an [example server](https://github.com/brandonweeks/nanoca/blob/main/examples/nanomdm.go) that runs both NanoCA (ACME CA) and NanoMDM in a single process.

```bash
$ git clone https://github.com/brandonweeks/nanoca.git
$ cd nanoca
$ go build -o nanoca-mdm ./examples/nanomdm.go
```

## Generate a CA certificate and key

You'll need a CA certificate and private key for NanoCA to issue device identity certificates with. For testing you can generate a self-signed CA:

```bash
$ openssl ecparam -name prime256v1 -genkey -noout -out rootCA.key
$ openssl pkcs8 -topk8 -nocrypt -in rootCA.key -out rootCA.pkcs8.key
$ openssl req -x509 -new -key rootCA.pkcs8.key -sha256 -days 3650 -out rootCA.crt -subj "/CN=NanoCA Root"
```

> **Note:** NanoCA requires PKCS#8 formatted private keys (the `openssl pkcs8` step above).

## Generate a TLS server certificate

The NanoCA example server terminates TLS directly (ACME requires HTTPS). You'll need a TLS certificate for the server. For testing with ngrok, you can use a self-signed certificate:

```bash
$ openssl ecparam -name prime256v1 -genkey -noout | openssl pkcs8 -topk8 -nocrypt -out server.key
$ openssl req -x509 -new -key server.key -sha256 -days 365 -out server.crt -subj "/CN=localhost"
```

For production, use a proper TLS certificate from a trusted CA or Let's Encrypt.

## Run the server

```bash
$ ./nanoca-mdm \
    -ca-cert rootCA.crt \
    -ca-key rootCA.pkcs8.key \
    -cert server.crt \
    -key server.key \
    -base-url "https://YOUR-NGROK-URL-HERE"
Starting server on :8443
ACME: https://YOUR-NGROK-URL-HERE/acme/directory
MDM: https://YOUR-NGROK-URL-HERE/mdm
```

This starts a server on port 8443 with:
- ACME CA endpoints at `/acme/`
- MDM endpoint at `/mdm`

## Run ngrok

```bash
$ ngrok http https://localhost:8443
```

Note the forwarding URL (e.g. `https://abc123.ngrok.io`). You'll need to restart the server with `-base-url` set to this URL so that ACME directory responses contain the correct URLs.

## Upload Push Certificate

Same as the [standard quickstart](quickstart.md#upload-push-certificate) — the NanoCA example server does not currently expose the push cert API, so you may need to use a separate NanoMDM instance or modify the example to include the API endpoints.

For a quick test, you can run a standard NanoMDM alongside (on a different port) with the same storage directory, or extend the example server.

## Configure enrollment profile

Take a copy of the [ACME enrollment profile example](enroll-acme.mobileconfig) and update the following values:

* `DirectoryURL` (in ACME payload): Your NanoCA ACME directory URL, e.g. `https://abc123.ngrok.io/acme/directory`
* `ServerURL` (in MDM payload): Your NanoMDM server URL, e.g. `https://abc123.ngrok.io/mdm`
* `Topic` (in MDM payload): Your push certificate topic, e.g. `com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9`

### Key differences from SCEP enrollment

The ACME payload replaces the SCEP payload. Compare:

**SCEP** ([enroll.mobileconfig](enroll.mobileconfig)):
- `PayloadType`: `com.apple.security.scep`
- Requires: `URL`, `Challenge`
- Device contacts an external SCEP server to get its certificate

**ACME** ([enroll-acme.mobileconfig](enroll-acme.mobileconfig)):
- `PayloadType`: `com.apple.security.acme`
- Requires: `DirectoryURL`, `ClientIdentifier`, `KeyType`, `KeySize`, `HardwareBound`
- Optional: `Attest` (enables device attestation)
- Device contacts the ACME server, proves its identity via attestation, and receives a certificate

In both cases, the MDM payload's `IdentityCertificateUUID` references the certificate payload's UUID. NanoMDM doesn't care which method was used — it validates the resulting device identity certificate against the CA certificate.

## Enroll a device

Install the modified enrollment profile on a device. If enrollment succeeds, you should see `Authenticate` and `TokenUpdate` messages in the server logs, just like with SCEP enrollment.

## Using NanoCA with a standalone NanoMDM

If you prefer to run the standard `nanomdm` binary separately (for its full feature set including API endpoints, webhooks, storage options, etc.), you can:

1. Run NanoCA as a separate ACME-only service (you'll need to write a small Go program or adapt the example)
2. Point your enrollment profile's `DirectoryURL` at the NanoCA service
3. Point the enrollment profile's `ServerURL` at your NanoMDM instance
4. Run NanoMDM with `-ca rootCA.crt` so it trusts certificates issued by NanoCA

The only thing NanoMDM needs is the CA certificate (`-ca` flag) to validate device identity certificates. It doesn't matter whether those certificates came from SCEP, ACME, or any other mechanism.
