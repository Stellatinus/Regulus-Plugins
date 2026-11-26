# Regulus plugins

The plugins of [Regulus](https://github.com/Stellatinus/Regulus), and **nothing else**: their source, their CI, their releases, and the CI that writes and signs the official plugin index. The **store** (the dashboard page that reads an index, checks it and installs plugins) is part of Regulus, not of this repository.

> **Status (2026-11-26): the index workflow has run once on GitHub and its signed, empty index is published and accepted by a hub (Regulus v0.4.1-alpha.1 showed the official source as verified, with no plugin).** No plugin has been released yet: `ci.yml` has run, but `publish-plugin.yml` and `release-plugin.yml` have not. The first plugin to release is [`disks`](plugins/disks/README.md) (UI only, an alpha). What was checked, and how, is in "What is verified" below. The design is [`docs/plugins/store.md`](https://github.com/Stellatinus/Regulus/blob/main/docs/plugins/store.md) in Regulus; this file only says what that design means for this repository. If the two disagree, the design in Regulus wins, and one of them is a bug.

## Layout

```
plugins/
  README.md           the list of all plugins, drafts included
  <id>/               one directory per plugin, named after its id
    plugin.yaml       the manifest (docs/plugins/manifest.md in Regulus)
    README.md         what it does, what it needs, how to try it
    CHANGELOG.md      one section per released version
    decisions.md      the decisions of this plugin's development
    ...               source, ui/, tests
  hello-world/        the test plugin: does nothing useful, to be removed later
tools/pluginctl/      the Go tool CI uses: check manifests, print release notes, write and check the index
docs/                 README.md (index) and decisions.md: the decisions that concern every plugin
scripts/              change detection, versions, packaging, pinned actions, and their tests
.github/workflows/    ci, publish-plugin, release-plugin, publish-index
yanked.txt            versions that are yanked (hidden from installs, still listed and downloadable)
```

## Rules

- **A directory is a plugin.** Every plugin, a draft included, is listed in [`plugins/README.md`](plugins/README.md) and has its own `README.md` and `decisions.md` (CI checks the list and the files). No code shared between plugins: each builds, is versioned and is released alone. Shared code belongs in Regulus' SDK.
- **`main` is the only long-lived branch.** Work goes in by pull request, one issue each, in English, with Conventional Commits.
- **Decisions are written down where they apply.** A decision about one plugin goes in that plugin's `decisions.md`; one that concerns several plugins or this repository goes in [`docs/decisions.md`](docs/decisions.md) (which also lists what is still open). Why, and what follows, not only what.
- **Pushing is not publishing. Only a tag publishes.** A plugin with no tag is in no index: that is where a draft or an idea lives. The code is public, but no hub is offered it.
- **Every plugin has an issue label named after its id, from the day its directory is created, a draft too.** An issue about a plugin carries that label, so it is found and cited by the plugin it is about. Whoever adds `plugins/<id>/` creates the label `<id>` in the same pull request's work (labels are not files; the pull request says it was created).
- **A tag is `plugin/<id>/v<version>`** and must equal `version` in that plugin's `plugin.yaml`, on a commit of `main` where CI passed. The version is chosen in a pull request (it is a line of `plugin.yaml`); the workflows check that it is exactly the next one of its channel.
- **Pre-releases are `-alpha.N` and `-beta.N`, numbered from 1, one at a time.** No `-rc`, no `-dev`: the release workflow refuses them (the Regulus client would still read `-rc` as a beta).
- **The channel comes from the version**, never from a file:

  | Version | Channel | Offered to |
  |---|---|---|
  | `1.2.0` | stable | everyone |
  | `1.3.0-beta.2`, `-rc.1` | beta | hubs on beta or alpha |
  | `1.3.0-alpha.1` | alpha | hubs on alpha |

  There is no `dev` channel. Work not ready for anyone is a directory with no tag; internal builds live in the development repository, `regulus-plugins-dev` *(proposed)*, which has this same layout and which only the maintainer adds as a source. A plugin that is ready is moved here.

- **A yanked version stays downloadable.** Never delete a release or a tag: rollbacks and pinned hubs depend on them.
- **`audience` defaults to `admin`.** A plugin that does not say who it is for is never shown to ordinary users. `audience` is a hint for the dashboard; what a user may do is decided by permissions, checked by the hub.

## What the workflows do

| Workflow | When | What |
|---|---|---|
| `ci.yml` | every pull request, every push to `main`, by hand | Works out what changed (`scripts/changes.sh`) and runs only what that needs. A change to documents only skips everything heavy. **Tooling**: tests of `scripts/` and `tools/`, actionlint, every action pinned to a commit. **Manifests**: `plugin.yaml`, `README.md`, `CHANGELOG.md` and `decisions.md` of each plugin that changed, the list of plugins, and the `node --test` tests of each plugin's UI module. **Build**: `go vet`, `go test -race` and a linux amd64/arm64 package of each plugin that has code and changed. **All checks** is the one result to require: it fails if any job failed or was cancelled. Superseded pull request runs are cancelled; runs on `main` never are |
| `publish-plugin.yml` ("Release a plugin") | by hand, on `main`, with the plugin id | Waits for CI on the commit, checks that the version in `plugin.yaml` is the next one of its channel and that its CHANGELOG has notes, tags `plugin/<id>/v<version>` and starts `release-plugin.yml` on the tag |
| `release-plugin.yml` | a `plugin/*/v*` tag | **verify** (commit on `main`, CI green, manifest, notes, version order), **build** (linux amd64 and arm64 packages, checksums), **publish** (checks the files against `checksums.txt`, attests the provenance, creates the GitHub release; pre-release when the version has one), then asks for a new index. A plugin with no program for a command of its manifest is **refused**: Regulus could not start it, so it must not reach the index |
| `publish-index.yml` | after a release, every Monday, by hand on `main` | Writes `index.json` from the plugin releases and `yanked.txt`, checks it, signs it with Sigstore (keyless) and publishes it, with its signature, on GitHub Pages. The index is valid for 21 days |

To know what to write in `plugin.yaml` for the next release, from a clone with the tags:

```sh
git tag -l 'plugin/<id>/v*' | sh scripts/plugin-version.sh next <id> alpha minor
```

To yank a version, add a line to `yanked.txt` by pull request and run **Publish index**. Nothing is ever deleted.

Run what CI runs on the tooling and on the UI modules (needs `node`) with `make test`, the checks of the plugins with `make check` (which also checks the list of plugins), and the linters (they need actionlint and shellcheck) with `make lint`.

## Repository settings these need (not files; Stellatinus/Regulus-Plugins#7)

None of these is set, and some are not verifiable from a file:

- **Protect `main`** (pull requests, and **All checks** as the required check) and restrict who can create `plugin/*` tags. The workflows repeat the checks a tag needs, but the workflow that runs is the one at the tagged commit, so whoever can push a tag can also remove them.
- **The environment `plugin-release`** with required reviewers, used by the publishing job and by the signing job. Without it the environment name changes nothing and nobody approves anything. The weekly index run waits for the same approval.
- **Pages** with "GitHub Actions" as its source, and the environment `github-pages`.
- **The signature names `publish-index.yml` at `refs/heads/main`.** Regulus' store client pins that identity (Stellatinus/Regulus#465). Whoever can push to `main`, or approve the environment, can sign for every hub that has this source.
- GitHub switches off scheduled workflows of a repository with no activity for 60 days: after that the index expires unless someone runs **Publish index**.
- Actions pinned to a commit are kept current by Dependabot (`.github/dependabot.yml`); run `scripts/pin-actions.sh` for a new action.

## What is verified

Run locally, on Go 1.24 and Linux: the unit tests of `tools/pluginctl` (manifest rules, release notes, the index, `yanked.txt`, the index check, under `-race`), the tests of `scripts/changes.sh` (in a temporary git repository), `scripts/plugin-version.sh` and `scripts/package-plugin.sh`; `actionlint` v1.7.7 on the four workflows and ShellCheck on `scripts/` and on the `run` steps; `scripts/pin-actions.sh --check`; the `jq` expressions of the workflows against a real index; the `node --test` tests of the Disks UI module (`scripts/test-ui.sh`); and the chain check, package (amd64 and arm64), notes, checksums, index, verify-index on `plugins/disks` and on throw-away plugins (the package of a UI-only plugin is byte for byte the same for both architectures).

**Not verified, because it cannot run here:** the Disks module in a real browser frame with the dashboard's host side, on a real hub (the tests drive its state machine and its view, not a browser); the workflows themselves on GitHub (triggers, permissions, environments, `gh api` output shape, the `gh release create` and `gh workflow run` calls), cosign signing, the build provenance attestation, GitHub Pages, and the whole thing against a real release. **A manifest rule copied from Regulus** (`tools/pluginctl/manifest.go`) can drift from the parser: the real fix is a `regulus plugin check` command in Regulus that CI runs. The Regulus store client has read the empty index made here and verified its signature (2026-11-26); it has not read an index that lists a plugin.

## Your own plugins, your own repository

This layout and the index format are the contract for third-party sources: copy the layout, publish a signed index, and give its address to a hub as a source. See "Other sources and trust" in the design: adding a source is an act of trust, because a plugin is code that runs on the servers.

## What blocks the plugins

- **A plugin with a program** (such as `hello-world`) cannot import the Regulus SDK today (it is `internal/`). Until the SDK and the plugin contract are importable (Stellatinus/Regulus#463), `plugins/hello-world` has a manifest and a description but no program, and **cannot be released**: `release-plugin.yml` refuses it. See "What blocks the work" in the design.
- **A UI-only plugin** (`plugins/disks`) needs no SDK, so it is not blocked by that. What it needs is in Regulus: the loader that hosts its module in a sandboxed frame, the tab in the node window, the store that installs it (all in the pull request that adds them), and the repository settings below the table of workflows.
