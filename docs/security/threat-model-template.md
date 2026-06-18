# Threat Model Template

## Scope

Component:

Stage:

In scope:

Out of scope:

## Assets

* Raw access keys.
* Device lease state.
* WireGuard private keys.
* Protected user traffic.
* Gateway credentials.
* Payment webhook secrets.

## Trust Boundaries

* macOS app to Packet Tunnel Extension.
* Swift to Go bridge.
* Client to gateway.
* Gateway to public Internet.
* Control API to SQLite.
* Payment provider to webhook.

## Threats

Record threats using STRIDE or another explicit model.

## Mitigations

Map each mitigation to a threat and a test or review artefact.

## Logging and Privacy

List every log field. Confirm no raw access keys, private keys or private traffic are logged.

## Residual Risk

Document accepted risk and evidence required before release.
