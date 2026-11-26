#!/bin/sh
# Decides what CI has to do for a change, in seconds, from the list of files
# that changed. It writes key=value lines (to $GITHUB_OUTPUT in a workflow, to
# standard output otherwise):
#
#   docs_only     true when every changed file is a document: nothing heavy runs
#   tooling       true when scripts, tools, workflows or anything that is not a
#                 plugin or a document changed (or when this is not a pull
#                 request): the tooling is tested and every plugin is checked
#   plugins       JSON array of the ids of the plugins to check: those that
#                 changed, or all of them when tooling is true
#   code_plugins  JSON array of those that have a go.mod and whose code changed
#                 (all of them when tooling is true): they are built and tested
#   has_plugins, has_code   "true" when the arrays are not empty, for `if:`
#
# Usage: scripts/changes.sh EVENT BASE HEAD
#   EVENT  the GitHub event name; anything but pull_request means "everything"
#   BASE   the commit the change starts from (the pull request's base)
#   HEAD   the commit where it ends (default: HEAD)
#
# A document is any *.md, LICENSE*, .gitignore, the issue templates and the pull
# request template. A *.md inside a plugin still makes that plugin be checked,
# because its README.md, CHANGELOG.md and decisions.md are part of the check, but it is not
# code. Anything unknown counts as tooling, so a new kind of file is never
# skipped by mistake. Run it from the root of the repository.
set -eu

event=${1:?usage: scripts/changes.sh EVENT BASE HEAD}
base=${2:-}
head=${3:-HEAD}
nl='
'

plugin_ids() {
	for d in plugins/*/; do
		[ -f "${d}plugin.yaml" ] && basename "$d"
	done
	return 0
}

keep_existing() { # ids on stdin: those that still have a plugin.yaml
	while IFS= read -r id; do
		[ -n "$id" ] && [ -f "plugins/$id/plugin.yaml" ] && echo "$id"
	done
	return 0
}

keep_go() { # ids on stdin: those with a go.mod
	while IFS= read -r id; do
		[ -n "$id" ] && [ -f "plugins/$id/go.mod" ] && echo "$id"
	done
	return 0
}

json_array() { # ids on stdin, one per line
	printf '['
	first=1
	while IFS= read -r id; do
		[ -n "$id" ] || continue
		[ "$first" = 1 ] || printf ','
		printf '"%s"' "$id"
		first=0
	done
	printf ']'
}

docs_only=false
tooling=false
changed=
code_changed=

if [ "$event" != pull_request ] || [ -z "$base" ] || [ "$base" = 0000000000000000000000000000000000000000 ]; then
	tooling=true # a push to main, a manual run: check everything
else
	files=$(git diff --name-only "$base...$head")
	docs_only=true
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		case "$f" in
			plugins/*/*.md) # (a * in a case pattern also matches a /, so this includes deeper files)
				id=${f#plugins/}
				changed="$changed${id%%/*}$nl"
				;;
			plugins/*/*)
				id=${f#plugins/}
				changed="$changed${id%%/*}$nl"
				code_changed="$code_changed${id%%/*}$nl"
				docs_only=false
				;;
			*.md | LICENSE | LICENSE.* | .gitignore | .github/ISSUE_TEMPLATE/*) ;; # (*.md includes the pull request template)
			*)
				tooling=true
				docs_only=false
				;;
		esac
	done <<EOF2
$files
EOF2
fi

if [ "$tooling" = true ]; then
	plugins=$(plugin_ids)
	code=$(plugin_ids | keep_go)
else
	plugins=$(printf '%s' "$changed" | sort -u | keep_existing)
	code=$(printf '%s' "$code_changed" | sort -u | keep_existing | keep_go)
fi

plugins_json=$(printf '%s\n' "$plugins" | json_array)
code_json=$(printf '%s\n' "$code" | json_array)
has_plugins=true
[ "$plugins_json" = '[]' ] && has_plugins=false
has_code=true
[ "$code_json" = '[]' ] && has_code=false

{
	echo "docs_only=$docs_only"
	echo "tooling=$tooling"
	echo "plugins=$plugins_json"
	echo "code_plugins=$code_json"
	echo "has_plugins=$has_plugins"
	echo "has_code=$has_code"
} >>"${GITHUB_OUTPUT:-/dev/stdout}"
