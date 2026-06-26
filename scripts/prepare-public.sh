#!/usr/bin/env bash
#
# prepare-public.sh — read-only pre-publication audit for Continuity VPN.
#
# Checks the TRACKED tree for things that must not be published before the
# repository goes public, and prints a go/no-go checklist. It is strictly
# non-destructive: it only reads, greps and reports. It never deletes, moves,
# stages or commits anything.
#
# See docs/dev/repository-strategy.md for the full release runbook. Run this
# from the repository root; it refuses to run anywhere else.
#
# Exit status: 0 = GO (all checks passed), 1 = NO-GO (one or more failures),
# 2 = could not run the audit (wrong directory, missing git, etc.).

set -Eeuo pipefail

trap 'printf "\n[%s] ERROR: audit aborted on line %s\n" "$SCRIPT_NAME" "$LINENO" >&2; exit 2' ERR

SCRIPT_NAME="$(basename -- "${BASH_SOURCE[0]}")"

# --- output helpers -------------------------------------------------------

pass()  { printf '  \033[32mPASS\033[0m  %s\n' "$*"; }
fail()  { printf '  \033[31mFAIL\033[0m  %s\n' "$*"; }
note()  { printf '        %s\n' "$*"; }
head2() { printf '\n== %s ==\n' "$*"; }

FAILURES=0
record_fail() { FAILURES=$((FAILURES + 1)); }

# --- preconditions --------------------------------------------------------

require_repo_root() {
	if ! command -v git >/dev/null 2>&1; then
		printf 'ERROR: git is not installed; cannot audit the tracked tree.\n' >&2
		exit 2
	fi

	local top
	if ! top="$(git rev-parse --show-toplevel 2>/dev/null)"; then
		printf 'ERROR: not inside a git working tree.\n' >&2
		exit 2
	fi

	if [[ "$top" != "$PWD" ]]; then
		printf 'ERROR: run this from the repository root.\n' >&2
		printf '       repo root: %s\n' "$top" >&2
		printf '       current  : %s\n' "$PWD" >&2
		exit 2
	fi
}

# Print the tracked file list once; reuse it everywhere.
load_tracked_files() {
	mapfile -t TRACKED_FILES < <(git ls-files)
	if [[ ${#TRACKED_FILES[@]} -eq 0 ]]; then
		printf 'ERROR: git reports no tracked files; nothing to audit.\n' >&2
		exit 2
	fi
}

# --- individual checks ----------------------------------------------------

# 1. No third-party AI/agent or editor scratch directories tracked.
check_junk_dirs() {
	head2 "Tool / scratch directories"
	local -a junk=(".agents" ".codebuddy" ".continue" ".junie" ".kiro" ".codex")
	local dir hits any=0
	for dir in "${junk[@]}"; do
		hits="$(printf '%s\n' "${TRACKED_FILES[@]}" | grep -E "^${dir}/" || true)"
		if [[ -n "$hits" ]]; then
			fail "tracked files under ${dir}/ (must not be published):"
			printf '%s\n' "$hits" | sed 's/^/          /'
			any=1
		fi
	done
	if [[ $any -eq 0 ]]; then
		pass "no tracked tool/scratch directories"
	else
		record_fail
		note "These are local assistant/editor dirs. Add them to .gitignore and"
		note "untrack with: git rm -r --cached <dir>   (do this yourself; not here)."
	fi
}

# 2. No tooling lockfile or built binaries tracked.
check_junk_files() {
	head2 "Lockfiles and built binaries"
	local -a junk=("skills-lock.json" "gateway" "continuityctl")
	local f hits any=0
	for f in "${junk[@]}"; do
		# Match the file at repo root only (leading-anchored, no slash before).
		hits="$(printf '%s\n' "${TRACKED_FILES[@]}" | grep -E "^${f}$" || true)"
		if [[ -n "$hits" ]]; then
			fail "tracked: ${f} (build artefact / lockfile — must not be published)"
			any=1
		fi
	done
	if [[ $any -eq 0 ]]; then
		pass "no tracked lockfiles or built binaries at repo root"
	else
		record_fail
		note "Rebuild binaries from source; do not commit them. Untrack with:"
		note "  git rm --cached <file>   (do this yourself; not here)."
	fi
}

# 3. Required licence / notice files present AND tracked.
check_licence_files() {
	head2 "Licence and notice files"
	local -a required=(
		"LICENSE"
		"NOTICE"
		"LICENSES/AGPL-3.0.txt"
		"LICENSES/Apache-2.0.txt"
	)
	local f any=0
	for f in "${required[@]}"; do
		if [[ -f "$f" ]] && printf '%s\n' "${TRACKED_FILES[@]}" | grep -qxF "$f"; then
			pass "present and tracked: ${f}"
		else
			fail "missing or untracked: ${f}"
			any=1
		fi
	done
	[[ $any -eq 0 ]] || record_fail
}

# 4. No obvious secrets in the tracked tree.
check_secrets() {
	head2 "Obvious secrets"
	# Patterns for clearly secret material. Licence texts and the SECURITY.md
	# guidance are excluded so their prose does not trip the scan.
	local -a patterns=(
		'-----BEGIN [A-Z ]*PRIVATE KEY-----'
		'AKIA[0-9A-Z]{16}'
		'-----BEGIN OPENSSH PRIVATE KEY-----'
		'xox[baprs]-[0-9A-Za-z-]{10,}'
		'gh[pousr]_[0-9A-Za-z]{20,}'
	)
	local pat hits any=0
	for pat in "${patterns[@]}"; do
		# -I skips binary files; restrict to tracked files via git grep.
		# This script and SECURITY.md are excluded so their literal pattern
		# strings / guidance prose do not match themselves.
		hits="$(git grep -nIE -- "$pat" \
			-- ':(exclude)LICENSES/*' ':(exclude)LICENSE' \
			   ':(exclude)scripts/prepare-public.sh' ':(exclude)SECURITY.md' 2>/dev/null || true)"
		if [[ -n "$hits" ]]; then
			fail "possible secret matching /${pat}/:"
			printf '%s\n' "$hits" | sed 's/^/          /'
			any=1
		fi
	done
	if [[ $any -eq 0 ]]; then
		pass "no obvious secret material in tracked files"
	else
		record_fail
		note "Investigate each hit. Real secrets must be rotated AND purged from"
		note "history before publishing — removing the file now is not enough."
	fi
}

# 5. No raw dotted-quad IPv4 in tracked source/docs.
# Documentation placeholders, TEST-NET ranges and licence texts are allowed.
check_raw_ipv4() {
	head2 "Raw IPv4 addresses"
	# Build the dotted-quad pattern from a non-literal octet so this script does
	# not itself contain an example IP (the project bans raw IPs in docs/source).
	local octet='[0-9]{1,3}'
	local ipv4="${octet}\\.${octet}\\.${octet}\\.${octet}"

	# Exclusions:
	#  - licence texts (verbatim upstream);
	#  - test files and testdata (fixtures legitimately carry sample identifiers);
	#  - TEST-NET documentation ranges (192.0.2.x, 198.51.100.x, 203.0.113.x);
	#  - link-local / loopback noise is left visible on purpose.
	local hits
	hits="$(git grep -nIE -- "$ipv4" \
		-- ':(exclude)LICENSES/*' \
		   ':(exclude)LICENSE' \
		   ':(exclude)*_test.go' \
		   ':(exclude)*/testdata/*' \
		   ':(exclude)**/testdata/*' \
		   ':(exclude)tests/*' 2>/dev/null \
		| grep -Ev '192\.0\.2\.|198\.51\.100\.|203\.0\.113\.' \
		| grep -Ev '127\.0\.0\.1|0\.0\.0\.0|255\.255\.255\.255' \
		|| true)"

	if [[ -z "$hits" ]]; then
		pass "no raw dotted-quad IPv4 in tracked source/docs"
	else
		fail "raw IPv4 found (use placeholders or TEST-NET ranges):"
		printf '%s\n' "$hits" | sed 's/^/          /'
		record_fail
		note "The repo bans raw host IPs in docs/source. Replace with a hostname"
		note "placeholder (gateway.example.com) or a TEST-NET range."
	fi
}

# 6. Advisory: warn if known-ignored paths are present but somehow tracked.
check_research_src() {
	head2 "Research / local-state directories (advisory)"
	local -a advisory=(".research-src" ".netcheck")
	local dir hits any=0
	for dir in "${advisory[@]}"; do
		hits="$(printf '%s\n' "${TRACKED_FILES[@]}" | grep -E "^${dir}/" || true)"
		if [[ -n "$hits" ]]; then
			fail "tracked files under ${dir}/ — these must never be published:"
			printf '%s\n' "$hits" | sed 's/^/          /'
			any=1
		fi
	done
	if [[ $any -eq 0 ]]; then
		pass ".research-src/ and .netcheck/ are not tracked"
	else
		record_fail
	fi
}

# --- main -----------------------------------------------------------------

main() {
	printf '%s — read-only pre-publication audit (non-destructive)\n' "$SCRIPT_NAME"
	require_repo_root
	load_tracked_files
	printf 'Auditing %d tracked files under %s\n' "${#TRACKED_FILES[@]}" "$PWD"

	check_junk_dirs
	check_junk_files
	check_licence_files
	check_secrets
	check_raw_ipv4
	check_research_src

	head2 "Verdict"
	if [[ $FAILURES -eq 0 ]]; then
		printf '  \033[32mGO\033[0m — automated checks passed.\n'
		note "Still do by hand before publishing (see docs/dev/repository-strategy.md):"
		note "  - make test && make lint && make licence-check && scripts/docs-check.sh"
		note "  - confirm the client never imports gateway packages (CI invariant)"
		note "  - audit git HISTORY, not just the current tree, before flipping public"
		note "  - update CONTRIBUTING.md licence wording to final"
		exit 0
	fi

	printf '  \033[31mNO-GO\033[0m — %d check(s) failed; resolve them and re-run.\n' "$FAILURES"
	note "Nothing was changed. This script only reports."
	exit 1
}

main "$@"
