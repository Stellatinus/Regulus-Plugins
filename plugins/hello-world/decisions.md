# Decisions: hello-world

Decisions about this plugin only. Decisions about every plugin are in [`docs/decisions.md`](../../docs/decisions.md).

### It does nothing useful, on purpose (2026-11-25)
**Decision:** hello-world counts its starts in its data directory, is healthy while that directory is writable, and later *provides* `hello.greet@1`. **Why:** the store has to be tried end to end (index, install, update, rollback, removal, a requirement between two plugins, a yanked version) before a real plugin depends on it.

### A second test plugin, `hello-guest`, tests requirements (2026-11-25)
**Decision:** `hello-guest` *needs* `hello.greet@1`. It is not created yet, because the requirement check is not built. **Consequence:** it will be a directory like this one, with its own label.

### It stays a draft with no tag (2026-11-26)
**Decision:** no program and no release: a plugin outside Regulus cannot import the SDK yet (Regulus#463), and `release-plugin.yml` refuses a command with no program. **Why:** the first real release is `disks`, which needs no SDK. **Consequence:** hello-world is listed in `plugins/README.md` as a draft and is removed when `disks` is released; its past versions, if any, stay in the index as yanked.
