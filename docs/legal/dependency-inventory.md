# Dependency Inventory

No production dependencies have been added yet.

## Go

The root Go module currently uses only the standard library.

## Swift

The Stage 0 Swift package currently uses only Swift standard libraries.

## Infrastructure

OpenTofu configuration is a placeholder and declares no providers or managed resources.

## GitHub Actions

| Action | Version | Workflows | Purpose |
| --- | --- | --- | --- |
| `actions/checkout` | `v7.0.0` | Go CI, Infrastructure CI, Licence Scan, macOS CI | Check out the repository in CI. |
| `actions/setup-go` | `v6.4.0` | Go CI | Install the Go toolchain from `go.mod`. |
| `opentofu/setup-opentofu` | `v2.0.1` | Infrastructure CI | Install OpenTofu for placeholder configuration validation. |

## CI Binary Tools

| Tool | Source | Workflows | Purpose |
| --- | --- | --- | --- |
| `swiftlint` | Homebrew formula | macOS CI | Strict Swift linting. |

## Required Future Updates

Update this file when adding any Go module, Swift package, provider, action, binary tool or vendored source.
