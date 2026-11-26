#!/bin/sh
# Tests scripts/package-plugin.sh with plugins made on the fly: one written in
# Go, one that is a script, and the ones that must be refused.
#
# Usage: scripts/test-package-plugin.sh
# shellcheck disable=SC2015 # ok always succeeds, so `A && ok || bad` is an if-else here
set -eu

here=$(cd "$(dirname "$0")" && pwd)
failures=0
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$work"

ok() { printf 'ok   %s\n' "$1"; }
bad() {
	printf 'FAIL %s\n' "$1"
	failures=$((failures + 1))
}

manifest() { # id version command-line-block
	printf 'id: %s\nversion: %s\nname: %s\napi: 1\ncomponents:\n  agent:\n    command: %s\n' "$1" "$2" "$1" "$3"
}

# A plugin in Go, with its own go.mod.
mkdir -p plugins/gopl
manifest gopl 0.1.0 bin/gopl >plugins/gopl/plugin.yaml
printf 'module example.test/gopl\n\ngo 1.21\n' >plugins/gopl/go.mod
printf 'package main\n\nfunc main() {}\n' >plugins/gopl/main.go

# A plugin that is a script, with a ui module.
mkdir -p plugins/scriptpl/bin plugins/scriptpl/ui
{ manifest scriptpl 1.2.3-beta.1 bin/run; printf '  ui:\n    module: ui/w.js\n'; } >plugins/scriptpl/plugin.yaml
printf '#!/bin/sh\nexit 0\n' >plugins/scriptpl/bin/run
chmod +x plugins/scriptpl/bin/run
echo "export default {}" >plugins/scriptpl/ui/w.js

# A plugin with only a manifest: must be refused.
mkdir -p plugins/onlymanifest
manifest onlymanifest 0.1.0 bin/onlymanifest >plugins/onlymanifest/plugin.yaml

# A script that is not executable, a path that leaves the directory, a directory
# that is not named after the id.
mkdir -p plugins/noexec/bin plugins/escape plugins/wrongdir
manifest noexec 0.1.0 bin/run >plugins/noexec/plugin.yaml
echo '#!/bin/sh' >plugins/noexec/bin/run
manifest escape 0.1.0 ../outside >plugins/escape/plugin.yaml
manifest other 0.1.0 bin/x >plugins/wrongdir/plugin.yaml

GOOS=linux GOARCH=amd64 "$here/package-plugin.sh" plugins/gopl dist >/dev/null 2>&1 && ok "a Go plugin is packed" || bad "a Go plugin is packed"
a=dist/gopl_0.1.0_linux_amd64.tar.gz
[ -f "$a" ] && [ -f "$a.sha256" ] && [ -f dist/gopl_0.1.0_plugin.yaml ] && ok "archive, checksum and manifest copy are written" || bad "archive, checksum and manifest copy are written"
(cd dist && sha256sum -c gopl_0.1.0_linux_amd64.tar.gz.sha256 >/dev/null) && ok "the checksum file is in the form sha256sum -c reads" || bad "the checksum file is valid"
tar -tzf "$a" | sort | tr '\n' ' ' | grep -q '^gopl/ gopl/bin/ gopl/bin/gopl gopl/plugin.yaml $' && ok "the archive holds the plugin directory, the program and the manifest" || bad "the archive layout ($(tar -tzf "$a" | tr '\n' ' '))"
tar -tvzf "$a" | grep -q 'gopl/bin/gopl' && tar -tvzf "$a" | grep 'gopl/bin/gopl' | grep -q '^-rwx' && ok "the program is executable" || bad "the program is executable"

# The same source gives the same archive.
cp "$a" first.tar.gz
GOOS=linux GOARCH=amd64 "$here/package-plugin.sh" plugins/gopl dist2 >/dev/null 2>&1
[ "$(sha256sum <first.tar.gz)" = "$(sha256sum <dist2/gopl_0.1.0_linux_amd64.tar.gz)" ] && ok "the archive is reproducible" || bad "the archive is reproducible"

# Another architecture has its own name.
GOOS=linux GOARCH=arm64 "$here/package-plugin.sh" plugins/gopl dist >/dev/null 2>&1 && [ -f dist/gopl_0.1.0_linux_arm64.tar.gz ] && ok "an arm64 package is named for its architecture" || bad "an arm64 package is named for its architecture"

GOOS=linux GOARCH=amd64 "$here/package-plugin.sh" plugins/scriptpl dist >/dev/null 2>&1 && ok "a script plugin is packed" || bad "a script plugin is packed"
tar -tzf dist/scriptpl_1.2.3-beta.1_linux_amd64.tar.gz | sort | tr '\n' ' ' | grep -q 'scriptpl/bin/run scriptpl/plugin.yaml scriptpl/ui/ scriptpl/ui/w.js' && ok "a script and a ui module are in the archive" || bad "a script and a ui module are in the archive"

for p in onlymanifest noexec escape wrongdir; do
	if GOOS=linux GOARCH=amd64 "$here/package-plugin.sh" "plugins/$p" refused >/dev/null 2>err.txt; then
		bad "$p is refused"
	else
		ok "$p is refused: $(head -n 1 err.txt)"
	fi
	[ -z "$(ls refused 2>/dev/null)" ] || bad "$p left files behind"
done
rm -rf refused

if [ "$failures" -gt 0 ]; then
	echo "$failures failed" >&2
	exit 1
fi
echo "all passed"
