#!/usr/bin/env sh
set -eu

manifest="research/upstream-manifest.yaml"
dest_root=".research-src"

if [ "${1:-}" = "--check" ]; then
	[ -f "$manifest" ] || { echo "missing $manifest" >&2; exit 1; }
	grep -q '^sources:' "$manifest" || { echo "$manifest does not define sources" >&2; exit 1; }
	echo "research manifest present"
	exit 0
fi

mkdir -p "$dest_root"

awk '
  /^[[:space:]]*-[[:space:]]*name:/ { name=$3 }
  /^[[:space:]]*repository:/ { repo=$2 }
  /^[[:space:]]*commit:/ {
    commit=$2
    gsub("\"", "", name)
    gsub("\"", "", repo)
    gsub("\"", "", commit)
    if (name != "" && repo != "" && commit != "" && commit !~ /^TODO/) {
      print name "\t" repo "\t" commit
    }
    name=""; repo=""; commit=""
  }
' "$manifest" | while IFS='	' read -r name repo commit; do
	case "$commit" in
		????????????????????????????????????????)
			case "$commit" in
				*[!0123456789abcdef]*)
					echo "invalid commit for $name: $commit" >&2
					exit 1
					;;
			esac
			;;
		*)
			echo "invalid commit for $name: $commit" >&2
			exit 1
			;;
	esac

	target="$dest_root/$name"
	if [ -d "$target/.git" ]; then
		env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" remote set-url origin "$repo"
	else
		mkdir -p "$target"
		env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" init -q
		env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" remote add origin "$repo"
	fi
	env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" fetch --depth 1 --no-tags origin "$commit"
	env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" checkout --detach "$commit"

	actual="$(env GIT_CONFIG_GLOBAL=/dev/null git -C "$target" rev-parse HEAD)"
	if [ "$actual" != "$commit" ]; then
		echo "checked out $actual for $name, expected $commit" >&2
		exit 1
	fi
done

echo "research sources fetched into $dest_root"
