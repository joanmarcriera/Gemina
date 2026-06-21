#!/usr/bin/env bash
# PostToolUse hook: keep Go edits gofmt-clean so the verify gate's `gofmt -l`
# check never fails late. Formats only the file that was just edited.
set -euo pipefail

input="$(cat)"
file="$(printf '%s' "$input" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' \
  2>/dev/null || true)"

case "$file" in
  *.go) ;;
  *) exit 0 ;;
esac
[ -f "$file" ] || exit 0
command -v gofmt >/dev/null 2>&1 || exit 0

gofmt -w "$file"
exit 0
