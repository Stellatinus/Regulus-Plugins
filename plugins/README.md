# The plugins

Every directory under `plugins/` is a plugin, and every plugin is listed here, **drafts included** (a draft is a directory with no tag: it is in no index). `scripts/check-plugin-list.sh` fails CI when a directory is missing from this table or the table names one that does not exist.

Each plugin has its own `README.md` (what it does, what it needs, how to try it), `CHANGELOG.md` and `decisions.md` (the decisions of its development; the ones that concern every plugin are in [`../docs/decisions.md`](../docs/decisions.md)), and its own issue label, named after its id (see the rules in the repository's [README](../README.md)).

| Plugin | What it is | State | Label |
|---|---|---|---|
| [hello-world](hello-world/README.md) | The test plugin: does nothing useful, to try the store end to end. Removed when the first real plugin is released | draft, `0.1.0-alpha.1`, manifest only (no program: it cannot be released until Regulus#463), no tag | `hello-world` |
| [disks](disks/README.md) | The first real plugin: a **Disks** tab in the node window (physical disks, their identity and their moves between servers). Issue #10 | draft, `0.1.0-alpha.1`, UI only with tests, ready for a first alpha but never run on a real hub, no tag | `disks` |

To add a plugin: create `plugins/<id>/` with `plugin.yaml`, `README.md`, `CHANGELOG.md` and `decisions.md`, add its row here, and create its label.
