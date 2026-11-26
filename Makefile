# Everything CI runs on the tooling, for a clone. The plugins themselves are
# checked with `make check`.

.PHONY: test check lint

test:
	cd tools/pluginctl && go vet ./... && go test -race -count=1 ./...
	sh scripts/test-changes.sh
	sh scripts/test-plugin-version.sh
	sh scripts/test-package-plugin.sh

# Manifest, README.md and CHANGELOG.md of every plugin.
check:
	cd tools/pluginctl && go run . check $(addprefix ../../,$(wildcard plugins/*))
	cd tools/pluginctl && go run . check-yanked ../../yanked.txt

# Needs actionlint and shellcheck on the PATH.
lint:
	sh scripts/pin-actions.sh --check
	actionlint
	shellcheck -s sh scripts/*.sh
