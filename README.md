# Regulus plugins

The plugins of [Regulus](https://github.com/Stellatinus/Regulus), and **nothing else**: their source, their CI, their releases, and the CI that writes and signs the official plugin index. The **store** (the dashboard page that reads an index, checks it and installs plugins) is part of Regulus, not of this repository.

> **Status: design only.** No plugin has been released and the index is not published. The design is [`docs/plugins/store.md`](https://github.com/Stellatinus/Regulus/blob/main/docs/plugins/store.md) in Regulus; this file only says what that design means for this repository. If the two disagree, the design in Regulus wins, and one of them is a bug.

## Layout

```
plugins/
  <id>/               one directory per plugin, named after its id
    plugin.yaml       the manifest (docs/plugins/manifest.md in Regulus)
    README.md         what it does, what it needs, how to try it
    CHANGELOG.md      one section per released version
    ...               source, ui/, tests
  hello-world/        the test plugin: does nothing useful, to be removed later
scripts/              package a plugin, check a manifest, write the index   (to write)
.github/workflows/    ci, release-plugin, publish-index                     (to write)
```

## Rules

- **A directory is a plugin.** No code shared between plugins: each builds, is versioned and is released alone. Shared code belongs in Regulus' SDK.
- **`main` is the only long-lived branch.** Work goes in by pull request, one issue each, in English, with Conventional Commits.
- **Pushing is not publishing. Only a tag publishes.** A plugin with no tag is in no index: that is where a draft or an idea lives. The code is public, but no hub is offered it.
- **A tag is `plugin/<id>/v<version>`** and must equal `version` in that plugin's `plugin.yaml`, on a commit of `main` where CI passed.
- **The channel comes from the version**, never from a file:

  | Version | Channel | Offered to |
  |---|---|---|
  | `1.2.0` | stable | everyone |
  | `1.3.0-beta.2`, `-rc.1` | beta | hubs on beta or alpha |
  | `1.3.0-alpha.1` | alpha | hubs on alpha |
  | `1.3.0-dev.7` | dev | **only** a hub whose own configuration turns developer builds on for this source; listed in `dev/index.json`, never in `index.json` |

- **A yanked version stays downloadable.** Never delete a release or a tag: rollbacks and pinned hubs depend on them.
- **`audience` defaults to `admin`.** A plugin that does not say who it is for is never shown to ordinary users. `audience` is a hint for the dashboard; what a user may do is decided by permissions, checked by the hub.

## Your own plugins, your own repository

This layout and the index format are the contract for third-party sources: copy the layout, publish a signed index, and give its address to a hub as a source. See "Other sources and trust" in the design: adding a source is an act of trust, because a plugin is code that runs on the servers.

## What blocks the first plugin

A plugin outside the Regulus repository cannot import its SDK today (it is `internal/`). Until the SDK and the plugin contract are importable, `plugins/hello-world` has a manifest and a description but no program. See "What blocks the work" in the design.
