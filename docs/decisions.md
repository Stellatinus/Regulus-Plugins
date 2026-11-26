# Decisions about the plugins

Decisions that concern **every plugin** of this repository, or the repository itself. A decision about **one** plugin is in that plugin's own `plugins/<id>/decisions.md`. The design of the plugin system and the store, and the decisions of the core, are in Regulus (`docs/plugins/`, `docs/decisions.md`); this file only records what was decided *here*. If a decision here contradicts Regulus, one of them is a bug: say so and fix the wrong one.

Each entry: **what was decided, why, and what follows from it.** Newest at the end of each section. A date is the day it was decided; "maintainer" means Giacomo Marra, who decides for the project.

## The repository

### Only plugins are here (2026-11-25, maintainer)
**Decision:** the repository holds plugins and what builds and releases them: their CI, their releases and the CI that writes and signs the official index. **Why:** the core's updates are trusted by the signature of the core's release workflow; plugin tags would fill the list of core releases every hub reads; each plugin needs releases of its own. **Consequence:** the store (the Store page, the index reader, the installer, the root helper), the plugin API, the SDK, the runtime and the manifest format stay in Regulus. Issues about the store go there; issues about a plugin or this repository's CI come here.

### `main` is the only long-lived branch; one pull request per issue (2026-11-25)
**Decision:** work goes in by pull request, one issue each, English only, Conventional Commits. **Why:** the same rules as Regulus. **Consequence:** a cloud session's branch counts as the issue's branch.

### Pushing is not publishing: only a tag publishes (2026-11-25)
**Decision:** a plugin with no tag is in no index, however much code it has. A tag is `plugin/<id>/v<version>`, must equal `version` in `plugin.yaml`, and must be on a commit of `main` where CI passed. **Why:** a draft must be able to live in the repository, in the open, without any hub being offered it. **Consequence:** a draft is a directory with no tag; it still has the full set of files below.

### No `dev` channel; a development repository instead (2026-11-25)
**Decision:** the channel comes from the version (`-alpha.N`, `-beta.N`, stable); there is no `dev` channel. Internal builds live in `regulus-plugins-dev` *(proposed)*, which has this same layout. **Why:** no special case in the client and nothing in the public index to hide.

### Never delete a release or a tag (2026-11-25)
**Decision:** a bad version is *yanked* (a line of `yanked.txt`), never deleted. **Why:** rollbacks and pinned hubs depend on the files and their checksums.

### Pre-releases are `-alpha.N` and `-beta.N`, from 1, one at a time (2026-11-25)
**Decision:** no `-rc`, no `-dev`; the release workflow refuses them. **Why:** the Regulus client would read `-rc` as a beta.

## What every plugin has

### A directory is a plugin, and it is listed (2026-11-26, maintainer)
**Decision:** every directory under `plugins/`, a draft included, is a row of the table in `plugins/README.md`, which links the plugin's own `README.md`. **Why:** one place to see every plugin and where it stands. **Consequence:** `scripts/check-plugin-list.sh` (in `make check` and CI) fails when a directory is missing from the table or the table names one that does not exist.

### One issue label per plugin, named after its id (2026-11-26, maintainer)
**Decision:** from the day its directory is created, a plugin has a GitHub label with its id (`disks`, `hello-world`), a draft too, and every issue about the plugin carries it. **Why:** issues can cite the plugin they are about, and a search by label gives its whole history. **Consequence:** labels are not files, so whoever adds the directory creates the label and says so in the pull request; a script cannot check it.

### Every plugin has `plugin.yaml`, `README.md`, `CHANGELOG.md` and `decisions.md` (2026-11-26, maintainer)
**Decision:** `decisions.md` in each plugin directory records the decisions of that plugin's development, in the format of this file; decisions that concern more than one plugin go in `docs/decisions.md`. **Why:** it lets the next person (or agent) continue quickly without rediscovering why. **Consequence:** `pluginctl check` fails a plugin with no `decisions.md`, as for the other three.

### A manifest uses only keys Regulus accepts (2026-11-25)
**Decision:** proposed keys (`kind`, `audience`, `needs`, versioned `provides`) stay as comments until Regulus accepts them. **Why:** Regulus reads the manifest strictly and would refuse the plugin. **Consequence:** `tools/pluginctl/manifest.go` copies the rules of Regulus' parser and is changed only when Regulus changes them (it follows `components.ui.node_tab` since 2026-11-26).

## Plugins that are only a UI

### A plugin may have no program (2026-11-26)
**Decision:** a plugin whose `components` is only `ui` is valid, packed and released like any other: its package holds `plugin.yaml` and the module. **Why:** the first real plugin (Disks) only presents data the core already collects; a program would be dead weight, and the core's rule that plugins run on the hubs only (Regulus#496) leaves nothing for a program to do there. **Consequence:** `scripts/package-plugin.sh` accepts a manifest with no `command`; it still refuses a `command` with no program.

### A UI-only package is offered for linux amd64 and arm64 with the same content (2026-11-26)
**Decision:** the release workflow packs every plugin for `linux/amd64` and `linux/arm64`, so a UI-only plugin gets two identical archives (`scripts/test-package-plugin.sh` checks they are byte for byte equal). **Why:** the index design has the client pick the artifact by os and arch; keeping that rule means no special case in the client or the index format, and no change to the workflow. **Consequence:** two files of the same size in the release; if an `any/any` artifact is ever added, it is a change of the index format (`format`), decided in Regulus.

### A plugin's UI is a dependency-free ES module, tested with `node --test` (2026-11-26)
**Decision:** a UI module is one file with no imports and no build step; pure functions (texts, state, view) are separate from the thin DOM layer, and `plugins/<id>/test/*.test.js` runs with node's own test runner (`scripts/test-ui.sh`, `make test`, CI). **Why:** the module runs in a sandboxed frame with no network, so it cannot import anything at run time; no install means CI cannot break on a package registry. **Consequence:** a plugin that really needs a build chain must say so in its own `decisions.md` and in the pull request, and the CI must learn to run it.

### The first real plugin is Disks, and it needs nothing (2026-11-26, maintainer)
**Decision:** `disks` is the first plugin that is meant to be released; it has **no `needs`**. **Why:** it did not need anything when it was part of the core, and a requirement check on a plugin that needs nothing is the case that must not fail. `hello-world`, the test plugin, is removed once `disks` is released. **Consequence:** see `plugins/disks/decisions.md`.

## Open

- **The importable SDK for plugins that have a program** (Regulus#463): the SDK is `internal/` in Regulus, and the Go module path there is still the old organization's. How a plugin builds against it (a public module with a version in the plugin's `go.mod`, compiled in this repository's CI, released as a ready-made binary per os and arch) is the maintainer's to decide; nothing here depends on it while the only real plugin is UI-only.
- **The development repository** `regulus-plugins-dev` is proposed and does not exist.
- **Repository settings** (protected `main`, the `plugin-release` environment, Pages, tag rules) are not files and not set (Stellatinus/Regulus-Plugins#7); the first release needs them.
- **`plugins/README.md` alone**: a change to it only is treated as a document change and skips the check of the list (`scripts/changes.sh`).
- **A `regulus plugin check` command in Regulus**, so the manifest rules live in one place (Regulus#500).
