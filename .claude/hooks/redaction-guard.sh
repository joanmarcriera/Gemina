#!/usr/bin/env bash
# PostToolUse guard: stop raw host identifiers (MAC / IPv4 dotted-quad) from
# landing in product source, diagnostics, research or docs. Mirrors the runtime
# redaction invariant asserted by .claude/skills/run-continuityctl/smoke.sh, but
# at edit time instead of gate time.
#
# Scope: files under internal/platform/darwin, internal/diagnostics, research/,
# or any *.md / *.txt — EXCLUDING *_test.go and testdata/, where sample
# identifiers are legitimate fixtures the project's own tests depend on.
#
# Exit 2 feeds the message back to Claude (the edit stays on disk; nothing is
# destroyed). If a match is an intentional documentation example, Claude may
# proceed; otherwise redact to coarse tokens or move data into testdata/.
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' \
  2>/dev/null || true)"

[ -n "$file" ] && [ -f "$file" ] || exit 0

case "$file" in
  *_test.go|*/testdata/*) exit 0 ;;
esac
case "$file" in
  */internal/platform/darwin/*|*/internal/diagnostics/*|*/research/*|*.md|*.txt) ;;
  *) exit 0 ;;
esac

mac='([0-9a-f]{2}:){5}[0-9a-f]{2}'
ipv4='[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}'
# Allowed dotted-quads — not host leaks: RFC1918 private (10/8, 172.16/12,
# 192.168/16), TEST-NET docs, loopback/any/broadcast. Anchored at ^ because each
# match is extracted onto its own line below. Keeps prepare-public.sh in step.
ipv4_allow='^(10|192\.168|172\.(1[6-9]|2[0-9]|3[01])|192\.0\.2|198\.51\.100|203\.0\.113|127\.|0\.0\.0\.0|255\.255\.255\.255)\.?'

hits=""
grep -qiE "$mac" "$file" && hits="MAC address"
# Flag only if some dotted-quad in the file is NOT in the allowlist above.
grep -oE "$ipv4" "$file" | grep -qvE "$ipv4_allow" && hits="${hits:+$hits and }IPv4 dotted-quad"

[ -n "$hits" ] || exit 0

{
  echo "Redaction invariant: $file contains a raw $hits."
  echo "This repo must never store raw host identifiers in source, diagnostics,"
  echo "research or docs (AGENTS.md; smoke.sh asserts the same on runtime output)."
  echo "Replace with coarse tokens (present/absent/wifi/android-rndis), or move"
  echo "sample hardware data into a testdata/ fixture. If this is a deliberate"
  echo "documentation example (e.g. a TEST-NET or loopback address), you may proceed."
  echo "Offending lines:"
  grep -niE "$mac|$ipv4" "$file" | head -5
} >&2
exit 2
