#!/bin/bash
# run.sh — run all three bundles, print exit codes.
cd "$(dirname "$0")"; for a in unsandboxed sandboxed_usb sandboxed_nousb; do echo "== $a"; ./$a.app/Contents/MacOS/sbclaim; echo "exit=$?"; done
