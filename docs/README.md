# Documents of the plugin repository

| Document | What it is |
|---|---|
| [Decisions](decisions.md) | what was decided about the repository and about every plugin: why, and what follows |
| [`../plugins/README.md`](../plugins/README.md) | the list of all plugins, drafts included, with the state of each |
| `../plugins/<id>/README.md` | what one plugin does, what it needs, how to try it |
| `../plugins/<id>/decisions.md` | the decisions of that plugin's development |
| `../plugins/<id>/CHANGELOG.md` | one section per released version |
| [`../README.md`](../README.md) | the rules and what the workflows do |
| [`../AGENTS.md`](../AGENTS.md) | the same rules, for AI agents |

The design of the plugin system and the store, the manifest, the runtime and the core's decisions are in Regulus ([`docs/plugins/`](https://github.com/Stellatinus/Regulus/tree/main/docs/plugins), [`docs/decisions.md`](https://github.com/Stellatinus/Regulus/blob/main/docs/decisions.md)). A decision about a plugin goes in that plugin's `decisions.md`; one that concerns several plugins or this repository goes in [`decisions.md`](decisions.md) here.
