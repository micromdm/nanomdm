# NanoMDM

NanoMDM is a minimalist [Apple MDM server](https://support.apple.com/business/enrollment-deployment) heavily inspired by MicroMDM.

# Features

- Horizontal scaling: zero/minimal local state, persistence in storage layers. MySQL backend provided in the box.
- Multiple APNs topics: potentially multi-tenant.
- Multi-command targeting: send the same command (or pushes) to multiple enrollments without individually queuing commands.
- Otherwise we share many features between MicroMDM and NanoMDM, such as:
  - A MicroMDM-emulating HTTP webhook/callback.
  - Enrollment-certificate authorization

## $x not included

If you've used [MicroMDM](https://github.com/micromdm/micromdm) before you might be interested to know what NanoMDM does *not* include:

- TLS. You'll need to provide your own reverse proxy that terminates TLS (an MDM protocol requirement). [ssl-proxy](https://github.com/suyashkumar/ssl-proxy) may be a quick & easy development solution.
- SCEP. Spin up your own [scep](https://github.com/micromdm/scep) server. Or bring your own.
- ADE (DEP) access. While ADE/DEP *enrollments* are supported there is no DEP API access.
- Enrollment. You'll need to create and serve your own enrollment profiles.
- Blueprints. No 'automatic' command sending.
- JSON command API. Commands are submitted in raw Plist form only.
  - The `micro2nano` project provides an API translation layer.
- VPP.

# Architecture Overview

NanoMDM, at its core, is a thin composable layer between HTTP handlers and a set of storage abstractions.

- The "front-end" is a set of standard Golang HTTP handlers that handle MDM and API requests. The core MDM handlers adapt the requests to the service layer. These handlers exist in the `http` package.
- The service layer is a composable set of interfaces that process and handle MDM requests. The service layer dispatches to the storage layer. These services exist under the `service` package.
- The storage layer is a set of interfaces and implementations that store & retrieve MDM enrollment and command data. These exist under the `storage` package.

