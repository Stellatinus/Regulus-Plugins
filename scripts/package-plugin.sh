#!/bin/sh
# Packs one plugin into the archive Regulus unpacks in its plugins directory
# (docs/plugins/writing-a-plugin.md in Regulus). Adapted from Regulus'
# scripts/package-plugin.sh: a plugin here is its own Go module (when it has
# Go code), not part of the repository's.
#
# Usage: scripts/package-plugin.sh plugins/<id> [output-directory]
#
# It writes, in the output directory (default: dist):
#   <id>_<version>_<os>_<arch>.tar.gz          the package
#   <id>_<version>_<os>_<arch>.tar.gz.sha256   "<sha256>  <file name>"
#   <id>_<version>_plugin.yaml                 a copy of the manifest, which the
#                                              plugin index is built from
# The archive holds one directory named after the plugin id with plugin.yaml,
# the program of each component and the ui module. os and arch come from GOOS
# and GOARCH (default: this machine).
#
# The program of a component (components.<role>.command) is, in this order:
#   1. a file of that name in the plugin directory (a script, or a program
#      built before), which must be executable;
#   2. built from the plugin's Go code: ./cmd/<name> if it exists, otherwise the
#      main package of the plugin directory, when the plugin has a go.mod.
# A plugin with no program for a command is REFUSED, not packed without it:
# Regulus would refuse to start it, so it must never reach the index. (This is
# why `hello-world`, which has only a manifest so far, cannot be released yet.)
#
# Nothing is signed here: the index is signed (publish-index.yml).
set -eu

src=${1:?usage: scripts/package-plugin.sh plugins/<id> [output-directory]}
out=${2:-dist}
src=${src%/}
[ -f "$src/plugin.yaml" ] || { echo "no $src/plugin.yaml" >&2; exit 1; }

# The manifest is strict YAML with one key per line; pluginctl check has
# already validated it.
id=$(sed -n 's/^id:[[:space:]]*//p' "$src/plugin.yaml" | head -n 1)
version=$(sed -n 's/^version:[[:space:]]*//p' "$src/plugin.yaml" | head -n 1)
[ -n "$id" ] && [ -n "$version" ] || { echo "$src/plugin.yaml has no id or version" >&2; exit 1; }
[ "$(basename "$src")" = "$id" ] || { echo "the directory $src is not named after the id $id" >&2; exit 1; }

here=$(pwd)
goos=${GOOS:-$(go env GOOS)}
goarch=${GOARCH:-$(go env GOARCH)}
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
stage="$work/$id"
mkdir -p "$stage"
cp "$src/plugin.yaml" "$stage/"

# The archive and the build must stay inside the plugin's own directory.
safe_path() {
	case "$1" in
		/* | .. | ../* | */../* | */..) echo "the manifest names the path $1, which leaves the plugin directory" >&2; exit 1 ;;
	esac
}

commands=$(sed -n 's/^[[:space:]]*command:[[:space:]]*//p' "$src/plugin.yaml" | sort -u)
for cmd in $commands; do
	safe_path "$cmd"
	name=$(basename "$cmd")
	mkdir -p "$stage/$(dirname "$cmd")"
	if [ -f "$src/$cmd" ]; then
		[ -x "$src/$cmd" ] || { echo "$src/$cmd is not executable" >&2; exit 1; }
		cp -p "$src/$cmd" "$stage/$cmd"
	elif [ -f "$src/go.mod" ]; then
		pkg=.
		[ -d "$src/cmd/$name" ] && pkg="./cmd/$name"
		(cd "$src" && CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build -trimpath -o "$work/$id/$cmd" "$pkg")
	else
		echo "$src has no program for the command $cmd: no such file, and no go.mod to build it from" >&2
		exit 1
	fi
done

module=$(sed -n 's/^[[:space:]]*module:[[:space:]]*//p' "$src/plugin.yaml" | head -n 1)
if [ -n "$module" ]; then
	safe_path "$module"
	[ -f "$src/$module" ] || { echo "the manifest names $module, which is not in $src" >&2; exit 1; }
	mkdir -p "$stage/$(dirname "$module")"
	cp "$src/$module" "$stage/$module"
fi

mkdir -p "$out"
out=$(cd "$out" && pwd)
archive="$out/${id}_${version}_${goos}_${goarch}.tar.gz"
# A fixed order and time make the same source give the same archive. It is
# written aside and renamed, so a failed tar leaves no truncated archive.
if ! (cd "$work" && find "$id" | LC_ALL=C sort | tar --no-recursion --mtime='UTC 1970-01-01' --owner=0 --group=0 --numeric-owner -czf - -T -) >"$archive.tmp"; then
	rm -f "$archive.tmp"
	echo "tar failed (this script needs GNU tar)" >&2
	exit 1
fi
mv "$archive.tmp" "$archive"
(cd "$out" && sha256sum "$(basename "$archive")" >"$(basename "$archive").sha256")
cp "$here/$src/plugin.yaml" "$out/${id}_${version}_plugin.yaml"
echo "$archive"
