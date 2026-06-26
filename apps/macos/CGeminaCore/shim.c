/*
 * shim.c — a single translation unit so SwiftPM treats CGeminaCore as a C
 * target and exposes its module map. The header only declares the bridge ABI;
 * the symbols are provided at link time by the Go c-archive
 * (bridge/geminacore, built with `go build -buildmode=c-archive`).
 *
 * The header here mirrors the canonical bridge/include/geminacore.h; keep
 * the two in sync.
 */
#include "geminacore.h"
