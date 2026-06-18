#!/usr/bin/env sh
set -eu

source_dir="$(pwd)"
tmp_parent="${TMPDIR:-/tmp}"
clean_dir="$(mktemp -d "$tmp_parent/continuity-vpn-clean.XXXXXX")"

case "$clean_dir" in
	"$source_dir"/*)
		echo "clean workspace must not be created inside the source tree: $clean_dir" >&2
		rmdir "$clean_dir"
		exit 1
		;;
esac

cleanup() {
	if [ "${KEEP_CLEAN_WORKSPACE:-0}" != "1" ]; then
		rm -rf "$clean_dir"
	else
		echo "kept clean workspace: $clean_dir"
	fi
}
trap cleanup EXIT INT TERM

rsync -a \
	--exclude='.git/' \
	--exclude='.build/' \
	--exclude='.codex/' \
	--exclude='.codex-cycle-started' \
	--exclude='.env' \
	--exclude='.research-src/' \
	--exclude='.terraform/' \
	--exclude='*.pem' \
	--exclude='*.key' \
	--exclude='*.tfstate' \
	--exclude='*.tfstate.*' \
	--exclude='DerivedData/' \
	--exclude='apps/macos/.build/' \
	--exclude='bin/' \
	--exclude='coverage.out' \
	"$source_dir/" "$clean_dir/"

cd "$clean_dir"

scripts/docs-check.sh
scripts/licence-check.sh
make test
make lint

echo "clean workspace validation ok: $clean_dir"
