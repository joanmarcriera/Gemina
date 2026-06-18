#!/usr/bin/env sh
set -eu

manifest="research/upstream-manifest.yaml"

if [ ! -f "$manifest" ]; then
	echo "missing $manifest" >&2
	exit 1
fi

if ! grep -q '^sources:' "$manifest"; then
	echo "$manifest does not define sources" >&2
	exit 1
fi

if grep -RIl 'GNU General Public License' apps cmd internal pkg bridge api db deploy observability tests 2>/dev/null | grep . >/dev/null 2>&1; then
	echo "GPL text found in product directories; verify no GPL source was copied" >&2
	exit 1
fi

for path in docs/legal/upstream-licences.md docs/legal/code-provenance.md docs/legal/dependency-inventory.md NOTICE; do
	if [ ! -f "$path" ]; then
		echo "missing legal/provenance file: $path" >&2
		exit 1
	fi
done

if grep -nE 'pin_status: pending|TODO-pin' "$manifest" >/tmp/continuity-vpn-pending-pins 2>/dev/null; then
	echo "upstream manifest still contains pending pins:" >&2
	cat /tmp/continuity-vpn-pending-pins >&2
	exit 1
fi

if grep -nE '^[[:space:]]*commit: [0-9a-f]{1,39}$|^[[:space:]]*commit: [0-9a-f]{41,}$|^[[:space:]]*commit: [0-9a-f]*[^0-9a-f[:space:]][^[:space:]]*' "$manifest" >/tmp/continuity-vpn-invalid-pins 2>/dev/null; then
	echo "upstream manifest contains invalid commit pins:" >&2
	cat /tmp/continuity-vpn-invalid-pins >&2
	exit 1
fi

echo "licence/provenance structure ok"
