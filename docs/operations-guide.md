# NanoMDM Operations Guide

This is a brief overview of the various command-line switches and HTTP endpoints (including APIs) available to NanoMDM.

## Switches

###  -api string

* API key for API endpoints

API authorization in NanoMDM is simply HTTP Basic authentication using "nanomdm" as the username and the API key as the password. Omitting this switch turns off all API endpoints — NanoMDM in this mode will essentially just be for handling MDM client requests. It is not compatible with also specifying `-disable-mdm`.

### -ca string

* Path to CA cert for verification

NanoMDM validates that the device identity certificate is issued from specific CAs. This switch is the path to a file of PEM-encoded CAs to validate against.

### -cert-header string

* HTTP header containing URL-escaped TLS client certificate

By default NanoMDM tries to extract the device identity certificate from the HTTP request by decoding the "Mdm-Signature" header. See ["Pass an Identity Certificate Through a Proxy" section of this documentation for details](https://developer.apple.com/documentation/devicemanagement/implementing_device_management/managing_certificates_for_mdm_servers_and_devices)). This corresponds to the `SignMessage` key being set to true in the enrollment profile.

With the `-cert-header` switch you can specify the name of an HTTP header that is passed to NanoMDM to read the client identity certificate. This is ostensibly to support Nginx' [$ssl_client_escaped_cert](http://nginx.org/en/docs/http/ngx_http_ssl_module.html) in a [proxy_set_header](http://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_set_header) directive. Though any reverse proxy setting a similar header could be used, of course. The `SignMessage` key in the enrollment profile should be set appropriately.

### -checkin

* enable separate HTTP endpoint for MDM check-ins

By default NanoMDM uses a single HTTP endpoint (`/mdm` — see below) for both commands and results *and* for check-ins. If this option is specified then `/mdm` will only be for commands and results and `/checkin` will only be for MDM check-ins.

### -debug

* log debug messages

Enable additional debug logging.

### -storage & -dsn

The `-storage` and `-dsn` flags together represent how the backend storage is configured. `-storage` specifies the name of the backend while `-dsn` specifies the backend data source name (in other words the connection string). These switches are used as a pair. If neither are supplied then it is as if you specified `-storage file -dsn db` meaning we use the `file` storage backend with `db` as its DSN. In the `file` backend's case the DSN is just a directory name of the DB.

#### Supported backends:

* `-storage file`  
Configures the file storage backend. This manages enrollment data in plain filesystem directories and files and has zero dependencies. The `-dsn` switch specifies the directory for the database.  
*Example:* `-storage file -dsn /path/to/my/db`
* `-storage mysql`  
Configures the MySQL storage backend. The `-dsn` switch should be in the [format the SQL driver expects](https://github.com/go-sql-driver/mysql#dsn-data-source-name). Be sure to create your tables with the [schema.sql](../storage/mysql/schema.sql) file first.  
*Example:* `-storage mysql -dsn nanomdm:nanomdm/mymdmdb`

#### Multiple backends:

You can configure multiple storage backends. Specifying multiple sets of `-storage` and `-dsn` flags (in paired order) will configure the "multi-storage" adapter. Be aware that only the first storage backend will be used when interacting with the system, all others storage is called to, but any results are discarded. In other words consider them write-only.

Also beware that you will have very bizaare results if you change to using multiple storage backends in the midst of existing enrollments. You will receive errors about missing database rows or data. A storage backend needs to be around when a device (or all devices) initially enroll(s). There is no "sync" or backfill system with multiple storage backends (see the migration ability if you need this).

This feature is really only useful if you've always been using multiple storage backends or if you're doing some type of development or testing (perhaps a new storage backend).

For example to use both a `file` *and* `mysql` backend your command line might look like: `-storage file -dsn db -storage mysql -dsn nanomdm:nanomdm/mymdmdb`. You can also mix and match backends, or mutliple types of the same backend. Behavior is undefined (and probably very bad) if you specify two backends of the same type with the same DSN.

### -dump

* dump MDM requests and responses to stdout

Dump MDM request bodies (i.e. complete Plist requests) to standard output for each request.

### -listen string

* HTTP listen address (default ":9000")

Specifies the listen address (interface & port number) for the server to listen on.

### -disable-mdm

* disable MDM HTTP endpoint

This switch disables MDM client capability. This effecitvely turns this running instance into "API-only" mode. It is not compatible with having an empty `-api` switch.

### -dm

* URL to send Declarative Management requests to

Specifies the "base" URL to send Declarative Management requests to. The full URL is constructed from this base URL appended with the type of Declarative Management ["Endpoint" request](https://developer.apple.com/documentation/devicemanagement/declarativemanagementrequest?language=objc) such as "status" or "declaration-items". Each HTTP request includes the NanoMDM enrollment ID as the HTTP header "X-Enrollment-ID". See [this blog post](https://micromdm.io/blog/wwdc21-declarative-management/) for more details.

### -migration

* HTTP endpoint for enrollment migrations

NanoMDM supports a lossy form of MDM enrollment "migration." Essentially if a source MDM server can assemble enough of both Authenticate and TokenUpdate messages for an enrollment you can "migrate" enrollments by sending those Plist requests to the migration endpoint. Importantly this transfers the needed Push topic, token, and push magic to continue to send APNs push notifications to enrollments.

This switch turns on the migration endpoint.

### -retro

* Allow retroactive certificate-authorization association

By default NanoMDM disallows requests which did not have a certificate association setup in their Authenticate message. For new enrollments this is fine. However for enrollments that did not have a full Authenticate message (i.e. for enrollments that were migrated) they will lack such an association and be denied the ability to connect.

This switch turns on the ability for enrollments with no existing certificate association to create one, bypassing the authorization check. Note if an enrollment already has an association this will not overwrite it; only if no existing association exists.

### -version

* print version

Print version and exit.

### -webhook-url string

* URL to send requests to

NanoMDM supports a MicroMDM-compatible [webhook callback](https://github.com/micromdm/micromdm/blob/main/docs/user-guide/api-and-webhooks.md) option. This switch turns on the webhook and specifies the URL.

## HTTP endpoints & APIs

### MDM

* Endpoint: `/mdm`

The primary MDM endpoint is `/mdm` and needs to correspond to the `ServerURL` key in the enrollment profile. Both command & result handling as well as check-in handling happens on at this endpoint by default. Note that if the `-checkin` switch is turned on then this endpoint will only handle command & result requests (having assumed that you updated your enrollment profile to include a separate `CheckInURL` key). Note the `-disable-mdm` switch will turn off this endpoint.

### MDM Check-in

* Endpoint: `/checkin`

This switch enables the separate MDM check-in endpoint and if enables needs to correspond to the `CheckInURL` key in the enrollment profile. By default MDM check-ins are handled by the `/mdm` endpoint unless this switch is turned on in which case this endpoint handles them. This endpoint is disabled unless the `-checkin` switch is turned on. Note the `-disable-mdm` switch will turn off this endpoint.

### Push Cert

* Endpoint: `/v1/pushcert`

The push cert API endpoint allows for uploading an APNS push certificate. It takes a concatenated PEM-encoded APNs push certificate and private key as its HTTP body. A quick way to utilize this endpoint is to use `curl`. For example:

```bash
$ cat /path/to/push.pem /path/to/push.key | curl -T - -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/pushcert'
{
	"topic": "com.apple.mgmt.External.e3b8ceac-1f18-2c8e-8a63-dd17d99435d9"
}
```

Here the `-T -` switch to `curl` tells it to take the standard-input and use it as the body for a PUT request to `/v1/pushcert`. We're also using `-u` to specify the API key (HTTP authentication). The server responded by telling us the topic that this Push certificate corresponds to.

### Push

* Endpoint: `/v1/push/`

The push API endpoint sends APNs push notifications to enrollments (which ask the MDM client to connect to the MDM server). This is a simple 

```bash
$ curl -u nanomdm:nanomdm 'http://127.0.0.1:9000/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8'
{
	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "8B16D295-AB2C-EAB9-90FF-8615C0DFBB08"
		}
	}
}
```

Here we successfully pushed to the client and received a push_result UUID from our push provider.

We can queue multiple pushes at the same time, too (note the separating comma in the URL):

```bash
$ curl -u nanomdm:nanomdm '[::1]:9000/v1/push/99385AF6-44CB-5621-A678-A321F4D9A2C8,E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'
{
	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "5736F13F-E2A2-E8B9-E21C-3973BDAA4054"
		},
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "A70400AA-C5D8-DBA7-D66E-1296B36FA7F5"
		}
	}
}

```

### Enqueue

* Endpoint: `/v1/enqueue/`

The enqueue API endpoint allows sending of commands to enrollments. It takes a raw command Plist input as the HTTP body. The `tools/cmdr.py` script helps generate basic MDM commands. For example (the `-r` switch picks a random read-only MDM command):

```bash
$ ./tools/cmdr.py -r
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Command</key>
	<dict>
		<key>RequestType</key>
		<string>ProfileList</string>
	</dict>
	<key>CommandUUID</key>
	<string>d7408b5d-f314-461f-bc5e-4ff107c03857</string>
</dict>
</plist>
```

Then, to submit a command to a NanoMDM enrollment:

```bash
$ ./tools/cmdr.py -r | curl -T - -u nanomdm:nanomm 'http://127.0.0.1:9000/v1/enqueue/E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'
{
	"status": {
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "16C80450-B79F-E23B-F99B-0810179F244E"
		}
	},
	"command_uuid": "1ec2a267-1b32-4843-8ba0-2b06e80565c4",
	"request_type": "ProfileList"

```

Here we successfully queued a command to an enrollment ID (UDID) `E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8`  with command UUID `1ec2a267-1b32-4843-8ba0-2b06e80565c4` and we successfully sent a push request.

Note here, too, we can queue a command to multiple enrollments:

```bash
$ ./tools/cmdr.py -r | curl -T - -u nanomdm:nanomm 'http://127.0.0.1:9000/v1/enqueue/99385AF6-44CB-5621-A678-A321F4D9A2C8,E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8'

	"status": {
		"99385AF6-44CB-5621-A678-A321F4D9A2C8": {
			"push_result": "4DE6E126-CC6C-37B2-7350-3AD1871C298F"
		},
		"E9085AF6-DCCB-5661-A678-BCE8F4D9A2C8": {
			"push_result": "7B9D73CD-186B-CCF4-D585-AEE9E8E4F0F3"
		}
	},
	"command_uuid": "9b7c63eb-14b4-4739-96b0-750a5c967371",
	"request_type": "ProvisioningProfileList"
}
```

Finally you can skip sending the push notification request by appending `?nopush=1` to the URI:

```bash
$ ./tools/cmdr.py -r | curl -v -T - -u nanomdm:nanomdm '[::1]:9000/v1/enqueue/99385AF6-44CB-5621-A678-A321F4D9A2C8?nopush=1'
{
	"no_push": true,
	"command_uuid": "598544b5-b681-4ce2-8914-ba7f45ff5c02",
	"request_type": "CertificateList"
}
```

Of course the device won't check-in to retrieve this command, it will just sit in the queue until it is told to check-in using a push notification. This could be useful if you want to send a large number of commands and only want to push after the last command is sent.

### Migration

* Endpoint: `/migration`

The migration endpoint (as talked about above under the `-migration` switch) is an API endpoint that allows sending raw `TokenUpdate` and `Authenticate` messages to establish an enrollment — in particular the APNs push topic, token, and push magic. This endpoint bypasses certificate validation and certificate authentication (though still requires API HTTP authentication). In this way we enable a way to "migrate" MDM enrollments from another MDM. This is how the `llorne` tool of [the micro2nano project](https://github.com/micromdm/micro2nano) works, for example.

### Version

* Endpoint: `/version`

Returns a JSON response with the version of the running NanoMDM server.

# Enrollment Migration (nano2nano)

The `nano2nano` tool extracts migration enrollment data from a given storage backend and sends it to a NanoMDM migration endpoint. In this way you can effectively migrate between database backends. For example if you started with a `file` backend you could migrate to a `mysql` backend and vice versa. Note that MDM servers must have *exactly* the same server URL for migrations to operate.

*Note:* Enrollment migration is **lossy**. It is not intended to bring over all data related to an enrollment — just the absolute bare minimum of data to support a migrated device being able to operate with MDM. For example previous commands & responses and even inventory data will be missing.

*Note:* There are some edge cases around enrollment migration. One such case is iOS unlock tokens. If the latest `TokenUpdate` did not contain the enroll-time unlock token for iOS then this information is probably lost in the migration. Again this feature is only meant to migrate the absolute minimum of information to allow for a device to be sent APNs push requests and have an operational command-queue.

## Switches

### -debug

* log debug messages

Enable additional debug logging.

### -storage & -dsn

See the "-storage & -dsn" section, above, for NanoMDM. The syntax and capabilities are the same.

### -key string

* NanoMDM API Key

The NanoMDM API key used to authenticate to the migration endpoint.

### -url string

* NanoMDM migration URL

The URL of the NanoMDM migration endpoint. For example "http://127.0.0.1:9000/migration".

### -version

* print version

Print version and exit.

## Example usage

```bash
$ ./nano2nano-darwin-amd64 -storage file -dsn db -url 'http://127.0.0.1:9010/migration' -key nanomdm -debug
2021/06/04 14:29:54 level=info msg=storage setup storage=file
2021/06/04 14:29:54 level=info checkin=Authenticate device_id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
2021/06/04 14:29:54 level=info checkin=TokenUpdate device_id=99385AF6-44CB-5621-A678-A321F4D9A2C8 type=Device
```