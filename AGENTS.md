# Notes for AI coding agents

This repository holds **only plugins** and what builds and releases them. The store (the client that reads an index and installs plugins), the plugin API, the SDK, the runtime and the manifest format are in [Stellatinus/Regulus](https://github.com/Stellatinus/Regulus); do not add them here.

Read before starting, in Regulus: `AGENTS.md`, `CONTRIBUTING.md`, `docs/plugins/README.md`, `docs/plugins/manifest.md`, `docs/plugins/runtime.md` and **`docs/plugins/store.md`** (the design this repository follows). Then this repository's `README.md`.

- One pull request per issue, `Closes #N`, English only, Conventional Commits. Issues about the store client belong in Regulus; issues about a plugin or about this repository's CI belong here.
- A manifest may only use keys the Regulus parser accepts today (it refuses unknown keys). Keys that are only proposed (`kind`, `audience`, `needs`, versioned `provides`) stay as comments until Regulus accepts them.
- Never delete or move a release or a tag. Do not say a part is verified unless it ran; nothing here has run on a real hub.
- If you learn something that matters and it is not written down, add it to the right file in the same pull request (design decisions go to `docs/decisions.md` in Regulus).
