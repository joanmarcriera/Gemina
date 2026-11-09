# Swift/Go Bridge

The C-compatible boundary between the Swift app and the Go transport core.

* `geminacore/` — the cgo bridge (`package main`, `//export`), built as a static
  archive with `go build -buildmode=c-archive -o apps/macos/build/libgeminacore.a
  ./bridge/geminacore`. The Xcode project (`apps/macos/project.yml`) runs that
  build for the `GeminaTunnel` extension. It exports `cc_session_new`,
  `cc_handshake_begin`, `cc_handshake_complete`, `cc_handshake_cancel`,
  `cc_outbound`, `cc_inbound` and `cc_session_free`.
* `include/geminacore.h` — the hand-authored header Swift imports; it mirrors
  the exports and states the memory-ownership contract (caller owns every
  buffer, input is copied, sessions are opaque `uint64` handles, no Go pointer
  crosses the boundary).
* `build/`, `darwin/` — placeholders.

`scripts/build-apple-bridge.sh` is still a stub; build the archive with the
command above. See ADR-0002, ADR-0005 and `docs/dev/xcode-signing.md`.
