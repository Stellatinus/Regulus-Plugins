#!/bin/sh
# Pins every GitHub Action in .github/workflows to a full commit SHA, keeping
# the tag it was at as a comment: `uses: actions/checkout@v4` becomes
# `uses: actions/checkout@<40 hex> # v4`. A tag can be moved by whoever owns
# the action's repository; a commit cannot.
#
# Usage: scripts/pin-actions.sh          rewrite the workflows (needs git and network)
#        scripts/pin-actions.sh --check  list what is not pinned; exit 1 if any
#
# Dependabot (package-ecosystem github-actions) keeps pinned actions current
# and understands the `# tag` comment, so run this once and review the diff.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
workflows=${WORKFLOWS_DIR:-$here/../.github/workflows}
check=0
[ "${1:-}" = "--check" ] && check=1

# unpinned prints "file:line:action@ref" for every `uses:` that is not a
# commit SHA (local actions, ./path, and docker:// are not pinned this way).
unpinned() {
	# *.yml and *.yaml, and a quoted value: uses: "a/b@v1".
	grep -n -E "^[[:space:]]*-?[[:space:]]*uses:[[:space:]]*[\"']?[A-Za-z0-9_.-]+/[^@[:space:]]+@" "$workflows"/*.yml "$workflows"/*.yaml 2>/dev/null |
		grep -v -E "@[0-9a-f]{40}[\"']?([[:space:]]|$)" || true
}

if [ "$check" = 1 ]; then
	list=$(unpinned)
	if [ -n "$list" ]; then
		printf 'not pinned to a commit:\n%s\n' "$list"
		exit 1
	fi
	echo "every action is pinned to a commit"
	exit 0
fi

# sha_of OWNER/REPO REF prints the commit of a tag or branch; for an annotated
# tag the commit it points to (the peeled ref) wins.
sha_of() {
	repo=$(printf '%s' "$1" | cut -d/ -f1-2)
	out=$(git ls-remote "https://github.com/$repo" "refs/tags/$2" "refs/tags/$2^{}" "refs/heads/$2")
	peeled=$(printf '%s\n' "$out" | awk '/\^\{\}$/ {print $1; exit}')
	[ -n "$peeled" ] && { printf '%s\n' "$peeled"; return 0; }
	printf '%s\n' "$out" | awk 'NF {print $1; exit}'
}

unpinned | while IFS= read -r line; do
	file=${line%%:*}
	rest=${line#*:}
	action=$(printf '%s' "$rest" | sed -E 's/.*uses:[[:space:]]*([^@[:space:]]+)@([^[:space:]#]+).*/\1/')
	ref=$(printf '%s' "$rest" | sed -E 's/.*uses:[[:space:]]*([^@[:space:]]+)@([^[:space:]#]+).*/\2/')
	case "$action" in
		\"* | \'*) echo "$file: a quoted uses: value cannot be rewritten; pin it by hand: $rest" >&2; exit 1 ;;
	esac
	sha=$(sha_of "$action" "$ref")
	[ -n "$sha" ] || { echo "cannot resolve $action@$ref" >&2; exit 1; }
	# Replace this action@ref wherever it appears in the file, whatever its
	# comment: one resolution per distinct line is enough.
	esc=$(printf '%s' "$action@$ref" | sed 's/[.[\*^$]/\\&/g')
	sed -i -E "s|(uses:[[:space:]]*)$esc([[:space:]]*)(#.*)?\$|\\1$action@$sha # $ref|" "$file"
done
