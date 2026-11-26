#!/bin/sh
# The versions of one plugin: what the next one is, and whether a release is
# allowed. Adapted from Regulus' scripts/next-version.sh, for tags of the form
# plugin/<id>/v<version>. The tags are read on standard input, one per line
# (git tag -l 'plugin/<id>/v*').
#
# Usage: git tag -l 'plugin/ID/v*' | scripts/plugin-version.sh next ID CHANNEL BUMP [VERSION]
#        git tag -l 'plugin/ID/v*' | scripts/plugin-version.sh check ID VERSION
#
#   next   prints the version to write in plugin.yaml for the next release
#   check  exits 0 if VERSION is exactly the next release of its channel and
#          says what is expected otherwise (the tag of VERSION itself may be in
#          the list: it is left out)
#
#   CHANNEL  stable, beta or alpha
#   BUMP     patch, minor or major: which part to raise when a new version
#            starts (ignored while a version is in alpha or beta, or with VERSION)
#   VERSION  X.Y.Z[-alpha.N|-beta.N]
#
# Rules, so that the order of the tags matches SemVer:
#   - A version starts from the latest stable release, raised by BUMP. While a
#     newer version has alphas or betas, it continues: 0.3.0-alpha.1,
#     0.3.0-alpha.2, 0.3.0-beta.1, ..., then the release 0.3.0.
#   - Pre-releases are numbered per channel from 1, one at a time.
#   - No alpha once the version has a beta (it would sort before the beta), and
#     no pre-release of a version already released.
#   - The only pre-releases are alpha.N and beta.N. There is no dev channel
#     (docs/plugins/store.md in Regulus): internal builds live in the
#     development repository, and rc is not used for plugins.
#
# Set PLUGIN_VERSION_SOURCED=1 to only define the functions (for tests).

semver_sort() {
	sed 's/-/~/' | sort -V | sed 's/~/-/'
}

newer() {
	[ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | semver_sort | tail -n 1)" = "$1" ]
}

is_core() {
	printf '%s\n' "$1" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'
}

bump() {
	major=${1%%.*}
	rest=${1#*.}
	minor=${rest%%.*}
	patch=${rest#*.}
	case "$2" in
		patch) echo "$major.$minor.$((patch + 1))" ;;
		minor) echo "$major.$((minor + 1)).0" ;;
		major) echo "$((major + 1)).0.0" ;;
		*) return 1 ;;
	esac
}

valid_id() {
	printf '%s\n' "$1" | grep -Eq '^[a-z][a-z0-9]*(-[a-z0-9]+)*$' && [ "${#1}" -ge 2 ] && [ "${#1}" -le 40 ]
}

# versions_of ID reads tags on stdin and prints the versions of that plugin that
# follow the rules above, in SemVer order. Other tags are ignored.
versions_of() {
	sed -n "s|^plugin/$1/v||p" | grep -E '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta)\.[1-9][0-9]*)?$' | semver_sort || true
}

# next_version ID CHANNEL BUMP [VERSION] reads the tags and prints the next version.
next_version() {
	id=$1
	channel=$2
	part=$3
	wanted=${4:-}
	valid_id "$id" || {
		echo "error: '$id' is not a plugin id" >&2
		return 1
	}
	case "$channel" in stable | beta | alpha) ;; *)
		echo "error: channel must be stable, beta or alpha, not '$channel'" >&2
		return 1
		;;
	esac
	case "$part" in patch | minor | major) ;; *)
		echo "error: bump must be patch, minor or major, not '$part'" >&2
		return 1
		;;
	esac

	versions=$(versions_of "$id")
	stable=$(printf '%s\n' "$versions" | grep -Ev -e '-' | grep . | tail -n 1 || true)
	[ -n "$stable" ] || stable=0.0.0
	newest=$(printf '%s\n' "$versions" | grep . | tail -n 1 || true)

	if [ -n "$wanted" ]; then
		is_core "$wanted" || {
			echo "error: version must be X.Y.Z, such as 0.3.0, not '$wanted'" >&2
			return 1
		}
		core=$wanted
	elif [ -n "$newest" ] && [ "${newest%%-*}" != "$newest" ] && newer "${newest%%-*}" "$stable"; then
		core=${newest%%-*} # a version in alpha or beta continues
	else
		core=$(bump "$stable" "$part")
	fi

	if printf '%s\n' "$versions" | grep -qx "$core"; then
		echo "error: $core is already released; choose a newer version" >&2
		return 1
	fi
	newer "$core" "$stable" || {
		echo "error: $core is not newer than the latest release, $stable" >&2
		return 1
	}

	if [ "$channel" = stable ]; then
		echo "$core"
		return 0
	fi
	esc=$(printf '%s' "$core" | sed 's/\./\\./g')
	if [ "$channel" = alpha ] && printf '%s\n' "$versions" | grep -Eq "^$esc-beta\."; then
		echo "error: $core already has a beta, and an alpha would sort before it; publish a beta, or a newer version with VERSION" >&2
		return 1
	fi
	last=$(printf '%s\n' "$versions" | sed -n "s/^$esc-$channel\.//p" | sort -n | tail -n 1)
	echo "$core-$channel.$((${last:-0} + 1))"
}

# check_version ID VERSION reads the tags and succeeds when VERSION is the
# release that comes next in its channel.
check_version() {
	id=$1
	ver=$2
	core=${ver%%-*}
	case "$ver" in
		*-*) pre=${ver#*-} ;;
		*) pre= ;;
	esac
	is_core "$core" || {
		echo "error: $ver is not a version like 0.3.0, 0.3.0-alpha.1 or 0.3.0-beta.2" >&2
		return 1
	}
	case "$pre" in
		"") channel=stable ;;
		alpha.[1-9] | alpha.[1-9][0-9]*) channel=alpha ;;
		beta.[1-9] | beta.[1-9][0-9]*) channel=beta ;;
		*)
			echo "error: the pre-release of $ver must be alpha.N or beta.N: there is no dev channel, and internal builds live in the development repository" >&2
			return 1
			;;
	esac
	case "${pre#*.}" in "$pre") ;; *[!0-9]*)
		echo "error: $ver: the number after alpha. or beta. must be digits only" >&2
		return 1
		;;
	esac
	tags=$(grep -vx "plugin/$id/v$ver" || true)
	want=$(printf '%s\n' "$tags" | next_version "$id" "$channel" minor "$core") || return 1
	if [ "$want" != "$ver" ]; then
		echo "error: the next $channel release of $id is $want, not $ver" >&2
		return 1
	fi
}

if [ "${PLUGIN_VERSION_SOURCED:-0}" != 1 ]; then
	set -eu
	cmd=${1:-}
	case "$cmd" in
		next)
			[ $# -ge 4 ] && [ $# -le 5 ] || {
				sed -n '7,8p' "$0" | sed 's/^# \{0,1\}//' >&2
				exit 2
			}
			shift
			next_version "$@"
			;;
		check)
			[ $# -eq 3 ] || {
				sed -n '7,8p' "$0" | sed 's/^# \{0,1\}//' >&2
				exit 2
			}
			valid_id "$2" || {
				echo "error: '$2' is not a plugin id" >&2
				exit 1
			}
			check_version "$2" "$3"
			;;
		*)
			sed -n '7,8p' "$0" | sed 's/^# \{0,1\}//' >&2
			exit 2
			;;
	esac
fi
