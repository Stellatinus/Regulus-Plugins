#!/bin/sh
# Tests scripts/changes.sh in a temporary git repository.
#
# Usage: scripts/test-changes.sh
set -eu

here=$(cd "$(dirname "$0")" && pwd)
failures=0
n=0
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
cd "$work"
git init -q -b main .
git config user.email t@example.test
git config user.name test

mkdir -p plugins/alpha plugins/beta-go/cmd scripts docs .github/workflows
echo "id: alpha" >plugins/alpha/plugin.yaml
echo "id: beta-go" >plugins/beta-go/plugin.yaml
echo "module x" >plugins/beta-go/go.mod
echo "package main" >plugins/beta-go/main.go
echo x >plugins/alpha/README.md
echo x >README.md
echo x >scripts/s.sh
git add -A
git commit -q -m base
base=$(git rev-parse HEAD)

# expect NAME EVENT "files to change" WANT, where WANT is the six output lines
# joined with ";".
expect() {
	n=$((n + 1))
	git checkout -q "$base"
	git checkout -q -b "t$n"
	printf '%s\n' "$3" | while IFS= read -r f; do
		[ -n "$f" ] || continue
		case "$f" in
			-*) git rm -q -f -r -- "${f#-}" ;;
			*) mkdir -p "$(dirname "$f")"; echo "change $n" >>"$f"; git add -- "$f" ;;
		esac
	done
	git commit -q --allow-empty -m change
	got=$("$here/changes.sh" "$2" "$base" HEAD | tr '\n' ';')
	if [ "$got" = "$4;" ]; then
		printf 'ok   %s\n' "$1"
	else
		printf 'FAIL %s\n  got  %s\n  want %s;\n' "$1" "$got" "$4"
		failures=$((failures + 1))
	fi
}

all='tooling=true;plugins=["alpha","beta-go"];code_plugins=["beta-go"];has_plugins=true;has_code=true'
expect "a README only" pull_request "README.md" 'docs_only=true;tooling=false;plugins=[];code_plugins=[];has_plugins=false;has_code=false'
expect "documents only" pull_request "README.md
docs/a.md
LICENSE" 'docs_only=true;tooling=false;plugins=[];code_plugins=[];has_plugins=false;has_code=false'
expect "a plugin's README is checked but is not code" pull_request "plugins/beta-go/README.md" 'docs_only=true;tooling=false;plugins=["beta-go"];code_plugins=[];has_plugins=true;has_code=false'
expect "a plugin's code" pull_request "plugins/beta-go/main.go" 'docs_only=false;tooling=false;plugins=["beta-go"];code_plugins=["beta-go"];has_plugins=true;has_code=true'
expect "a manifest-only plugin has no code to build" pull_request "plugins/alpha/plugin.yaml" 'docs_only=false;tooling=false;plugins=["alpha"];code_plugins=[];has_plugins=true;has_code=false'
expect "two plugins" pull_request "plugins/alpha/plugin.yaml
plugins/beta-go/main.go" 'docs_only=false;tooling=false;plugins=["alpha","beta-go"];code_plugins=["beta-go"];has_plugins=true;has_code=true'
expect "a script is tooling: everything is checked" pull_request "scripts/s.sh" "docs_only=false;$all"
expect "a workflow is tooling" pull_request ".github/workflows/ci.yml" "docs_only=false;$all"
expect "an unknown file is tooling" pull_request "something.bin" "docs_only=false;$all"
expect "a document and a script" pull_request "README.md
scripts/s.sh" "docs_only=false;$all"
expect "a deleted plugin has nothing to check" pull_request "-plugins/alpha" 'docs_only=false;tooling=false;plugins=[];code_plugins=[];has_plugins=false;has_code=false'
expect "a push checks everything" push "README.md" "docs_only=false;$all"
expect "a manual run checks everything" workflow_dispatch "README.md" "docs_only=false;$all"

# A pull request whose base moved on: only what the branch changed counts.
git checkout -q "$base"
git checkout -q -b side
echo more >>README.md
git commit -q -am "docs on the branch"
head=$(git rev-parse HEAD)
git checkout -q main
echo more >>scripts/s.sh
git commit -q -am "tooling on main, after the branch started"
moved=$(git rev-parse HEAD)
got=$("$here/changes.sh" pull_request "$moved" "$head" | tr '\n' ';')
if [ "$got" = 'docs_only=true;tooling=false;plugins=[];code_plugins=[];has_plugins=false;has_code=false;' ]; then
	echo "ok   changes on the base branch are not the pull request's"
else
	printf 'FAIL base moved on\n  got %s\n' "$got"
	failures=$((failures + 1))
fi

if [ "$failures" -gt 0 ]; then
	echo "$failures failed" >&2
	exit 1
fi
echo "all passed"
