# NanoMDM Admin Console

NanoMDM includes a small, embedded administration console at `/admin/`. It is
served by the same HTTP server and requires no additional frontend build step.

Start NanoMDM with the API enabled, then open `http://localhost:9000/admin/`.
Enter the API key configured with `-api` (or `NANOMDM_API`) in **Settings**.
The browser sends it as Basic authentication with the fixed NanoMDM API
username `nanomdm`. The key is held in session storage and is not sent
anywhere except the current server.

## Included functionality

The console uses the existing HTTP API only:

- `/version` for the server health and version card.
- `/v1/enrollments` for live enrollment inventory when using MySQL or
  PostgreSQL storage. The endpoint returns enrollment metadata without push
  tokens or identity certificates.
- `/v1/enrollments/{id}` for one enrollment's persisted metadata.
- `/v1/push/{id}` for a push notification.
- `/v1/enqueue/{id}` for real raw-plist commands: device information, profile
  list, restart, shutdown, lock, and wipe.
- `/v1/pushcert?topic=...` as the configured APNs topic health surface.
- The **Enrollment** page generates a configured `.mobileconfig` starting
  profile from the repository template. It intentionally leaves the SCEP
  challenge placeholder and does not replace a real SCEP service or profile
  signing/distribution workflow.

With MySQL or PostgreSQL storage, the **Devices** page uses live enrollment
inventory from `/v1/enrollments`. If the configured storage backend does not
implement that optional reader (for example, file or in-memory storage),
enrollment IDs can still be entered as operator-managed local targets and are
stored in the browser's local storage.

## Deliberate backend limitations

The current NanoMDM API does not expose Apple inventory detail/result
history, command-history reads, server log reads, profile hosting or
management, or users and roles. The console labels these capabilities as
unavailable rather than showing fabricated data. SCEP, ADE/DEP API access,
and enrollment profile delivery remain external integrations as described in
the operations guide.
