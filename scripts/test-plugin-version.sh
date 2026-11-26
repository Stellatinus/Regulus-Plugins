#!/bin/sh
# Tests scripts/plugin-version.sh, the rules of the versions of a plugin.
#
# Usage: scripts/test-plugin-version.sh
set -eu

here=$(cd "$(dirname "$0")" && pwd)
failures=0

PLUGIN_VERSION_SOURCED=1
export PLUGIN_VERSION_SOURCED
# shellcheck disable=SC1091
. "$here/plugin-version.sh"

nl='
'
p=plugin/demo/v
# next NAME TAGS CHANNEL BUMP VERSION WANT: WANT is the printed version, or "error".
next() {
	if got=$(printf '%s' "$2" | next_version demo "$3" "$4" "$5" 2>/dev/null); then :; else got=error; fi
	if [ "$got" = "$6" ]; then printf 'ok   next: %s\n' "$1"; else
		printf 'FAIL next: %s (got %s, want %s)\n' "$1" "$got" "$6"
		failures=$((failures + 1))
	fi
}
# check NAME TAGS VERSION WANT: WANT is ok or error.
check() {
	if printf '%s' "$2" | check_version demo "$3" >/dev/null 2>&1; then got=ok; else got=error; fi
	if [ "$got" = "$4" ]; then printf 'ok   check: %s\n' "$1"; else
		printf 'FAIL check: %s (got %s, want %s)\n' "$1" "$got" "$4"
		failures=$((failures + 1))
	fi
}

next "first alpha" "" alpha minor "" 0.1.0-alpha.1
next "first beta" "" beta minor "" 0.1.0-beta.1
next "first stable" "" stable minor "" 0.1.0
next "next alpha" "${p}0.1.0-alpha.1${nl}${p}0.1.0-alpha.2" alpha minor "" 0.1.0-alpha.3
next "alpha numbers compare as numbers" "${p}0.1.0-alpha.9${nl}${p}0.1.0-alpha.10" alpha minor "" 0.1.0-alpha.11
next "beta after alphas" "${p}0.1.0-alpha.2" beta minor "" 0.1.0-beta.1
next "release after betas" "${p}0.1.0-alpha.2${nl}${p}0.1.0-beta.3" stable major "" 0.1.0
next "no alpha after a beta" "${p}0.1.0-beta.1" alpha minor "" error
next "new minor after a release" "${p}0.1.0" beta minor "" 0.2.0-beta.1
next "new patch after a release" "${p}0.1.0" stable patch "" 0.1.1
next "new major after a release" "${p}0.1.0" alpha major "" 1.0.0-alpha.1
next "bump ignored while a version is in beta" "${p}0.1.0${nl}${p}0.2.0-beta.1" beta major "" 0.2.0-beta.2
next "explicit version" "${p}0.1.0${nl}${p}0.2.0-beta.1" alpha minor 1.0.0 1.0.0-alpha.1
next "explicit version already released" "${p}0.1.0" stable minor 0.1.0 error
next "explicit version older than the release" "${p}0.2.0" stable minor 0.1.5 error
next "explicit version not X.Y.Z" "" stable minor 1.0 error
next "unknown channel" "" nightly minor "" error
next "unknown bump" "" stable huge "" error
next "tags of other plugins and of Regulus are ignored" "plugin/other/v1.0.0${nl}v9.9.9${nl}${p}0.1.0" stable minor "" 0.2.0
next "an id that is a prefix of another is not confused" "plugin/demo-two/v5.0.0${nl}${p}0.1.0" stable minor "" 0.2.0
next "a dev or rc tag is ignored" "${p}0.2.0-dev.1${nl}${p}0.2.0-rc.1${nl}${p}0.1.0" stable minor "" 0.2.0
next "a bad plugin id" "" stable minor "" 0.1.0
if printf '' | next_version "Bad_Id" stable minor "" >/dev/null 2>&1; then
	echo "FAIL next: a bad id was accepted"
	failures=$((failures + 1))
else echo "ok   next: a bad id is refused"; fi

check "the first alpha" "" 0.1.0-alpha.1 ok
check "the tag of the release itself is left out" "${p}0.1.0-alpha.1" 0.1.0-alpha.1 ok
check "the second alpha" "${p}0.1.0-alpha.1" 0.1.0-alpha.2 ok
check "an alpha number skipped" "${p}0.1.0-alpha.1" 0.1.0-alpha.3 error
check "an alpha repeated" "${p}0.1.0-alpha.1${nl}${p}0.1.0-alpha.2" 0.1.0-alpha.2 ok
check "a beta after alphas" "${p}0.1.0-alpha.2" 0.1.0-beta.1 ok
check "an alpha after a beta" "${p}0.1.0-beta.1" 0.1.0-alpha.2 error
check "the release after betas" "${p}0.1.0-beta.1" 0.1.0 ok
check "a release that is already out" "${p}0.1.0" 0.1.0 ok
check "a release older than the latest" "${p}0.2.0" 0.1.5 error
check "a beta of a released version" "${p}0.1.0" 0.1.0-beta.2 error
check "dev is refused" "" 0.1.0-dev.1 error
check "rc is refused" "" 0.1.0-rc.1 error
check "alpha.0 is refused" "" 0.1.0-alpha.0 error
check "alpha.01 is refused" "" 0.1.0-alpha.01 error
check "not a version" "" 1.0 error
check "a leading v is refused" "" v0.1.0 error

if [ "$failures" -gt 0 ]; then
	echo "$failures failed" >&2
	exit 1
fi
echo "all passed"
