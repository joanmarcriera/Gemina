# Security Policy

## Supported Versions

No released product version exists yet.

## Reporting

Report security concerns directly to the repository owner until a public security contact is approved.

Do not include secrets, raw access keys, customer identifiers or packet captures containing private traffic in issue trackers, logs or support attachments.

## Current Security Boundaries

The repository is pre-release. It contains a gateway runtime (`cmd/gateway`,
probe and data modes), an encrypted and authenticated data plane (ADR-0006,
ADR-0007) and a Wi-Fi-only macOS packet tunnel that is code-complete but not yet
verified on hardware. It has no payment flow, no hosted entitlement service and
no released client. The gateway session and data-plane threat model is
[`docs/security/stage-2-session-security.md`](docs/security/stage-2-session-security.md).

Future work must update concrete threat models before implementing
security-sensitive behaviour.
