#!/usr/bin/env sh
set -eu

required_files='
AGENTS.md
PROJECT_STATE.md
TASKS.md
DECISIONS.md
docs/product/project-specification.md
docs/architecture/overview.md
docs/adr/template.md
docs/adr/0001-continuity-first.md
docs/legal/upstream-licences.md
docs/legal/code-provenance.md
docs/legal/dependency-inventory.md
docs/security/threat-model-template.md
research/upstream-manifest.yaml
'

missing=0
for path in $required_files; do
	if [ ! -f "$path" ]; then
		echo "missing required file: $path" >&2
		missing=1
	fi
done

if [ "$missing" -ne 0 ]; then
	exit 1
fi

if find . -path './.git' -prune -o -path './.research-src' -prune -o -name '*.md' -print | xargs grep -n 'TODO$' >/dev/null 2>&1; then
	echo "documentation contains bare TODO markers" >&2
	exit 1
fi

echo "documentation structure ok"
