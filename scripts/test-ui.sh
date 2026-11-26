#!/bin/sh
# Runs the tests of the UI modules: for each plugin given (default: all) that
# has plugins/<id>/test/*.test.js, `node --test` on them. Node's own test runner
# only: a plugin's tests need no dependency and no install. Run it from the root
# of the repository.
#
# Usage: scripts/test-ui.sh [ID...]
set -eu

command -v node >/dev/null 2>&1 || { echo "node is needed to test the UI modules" >&2; exit 1; }

if [ "$#" -eq 0 ]; then
	set --
	for d in plugins/*/; do
		set -- "$@" "$(basename "$d")"
	done
fi

status=0
for id in "$@"; do
	set -- "plugins/$id"/test/*.test.js
	[ -e "$1" ] || continue
	echo "== $id"
	node --test "$@" || status=1
done
exit $status
