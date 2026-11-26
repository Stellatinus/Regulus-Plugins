#!/bin/sh
# Checks that plugins/README.md, the list of plugins, names every plugin
# directory exactly as `| [<id>](<id>/README.md) |` and names nothing else.
# Run it from the root of the repository.
set -eu

list=plugins/README.md
[ -f "$list" ] || { echo "$list is missing" >&2; exit 1; }
status=0

for d in plugins/*/; do
	id=$(basename "$d")
	if ! grep -q "^| \[$id\]($id/README.md) |" "$list"; then
		echo "$list does not list the plugin $id" >&2
		status=1
	fi
done

grep -o '^| \[[a-z0-9-]*\]([a-z0-9-]*/README.md) |' "$list" | sed 's/^| \[\([a-z0-9-]*\)\].*/\1/' | while read -r id; do
	[ -d "plugins/$id" ] || { echo "$list lists $id but plugins/$id does not exist" >&2; exit 1; }
done || status=1

exit $status
