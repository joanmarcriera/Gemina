# Bridge Headers

`geminacore.h` is the public C header for `bridge/geminacore`: the hand-authored
contract the Swift side imports (the cgo-generated header is a build artefact and
is not checked in). Keep it in step with the `//export` functions in
`bridge/geminacore/bridge.go` and `handshake.go`.
