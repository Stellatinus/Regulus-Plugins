# Decisions: disks

Decisions about this plugin only. Decisions about every plugin are in [`docs/decisions.md`](../../docs/decisions.md); the core's side of the cut is in `docs/disks.md` in Regulus (Regulus#497).

### The first real plugin, to try the plugin system on something real (2026-11-26, maintainer)
**Decision:** the **Physical disks** panel of the core becomes this plugin: a **Disks** tab (**Dischi** in Italian) of the node window, between General and SSH terminal. It is **not a disk manager**: it shows statistics and identity, it changes nothing. **Why:** the maintainer wanted the plugin system and the store tried on something real. The moves of disks between servers are part of the tab, not a separate page.

### UI-only: the core keeps the collection, the identity and the storage (2026-11-26, proposed in Regulus#497, taken as the way to proceed)
**Decision:** the plugin owns the screens only. The agent's collection of identities, `internal/diskid`, the tables `disks` and `disk_moves` and the routes `GET /api/agents/{id}/disks` and `GET /api/disks/moves` stay in the core. **Why:** a plugin has no migration mechanism and the disk history must survive the plugin being updated or removed; the collection must run on every agent, and plugins run on the hubs only (Regulus#496). **Consequence:** no program, no `hub` component, no SDK (so Regulus#463 does not block it); no data moves, so nothing is lost for an installation with disk history. Open in Regulus#497: whether the plugin should own its storage one day (then it needs a storage and migration mechanism first).

### It reads through the dashboard, with two read-only operations (2026-11-26)
**Decision:** the module runs in a sandboxed frame with an opaque origin and no network; it talks to the dashboard only by postMessage, protocol v1, and asks for two operations: `inventory.disks` {node} and `inventory.disk_moves` {limit}, which return the JSON of the existing routes. **Why:** this is the isolation of Regulus' `docs/plugins/README.md` ("How widget code is isolated from the session", #315): the module has no cookie and cannot call the hub. **Consequence:** the dashboard decides whether the viewer may read (`inventory.read`); a refusal is shown to the user as "no permission", not as an error. The module accepts the route's whole JSON or a bare list.

### The permission is `disks.read`, mapped to `inventory.read` (2026-11-26)
**Decision:** the plugin's one permission is `disks.read`; the core grants the operations only to a viewer who holds `inventory.read`. **Why:** serial numbers identify hardware, and `viewer` must still not see them. **Consequence:** until plugins' permissions are enforced (Regulus#61) the hub's check of `inventory.read` on the routes is what protects the data, as it does today. `audience: admin` stays a comment until Regulus accepts the key (#464).

### It has no requirements (2026-11-26, maintainer)
**Decision:** no `needs`. **Why:** it required nothing when it was part of the core, so the requirement check must not stop it; a first plugin that cannot be installed would test nothing. **Consequence:** do not add one without the maintainer.

### The tab is declared in the manifest: `components.ui.node_tab` (2026-11-26)
**Decision:** `node_tab: {id: disks, title: {en: Disks, it: Dischi}}`. **Why:** the dashboard has to draw the tab before it loads the frame. English and Italian are the only languages of the dashboard.

### One dependency-free module, built with DOM nodes, styled by CSSOM (2026-11-26)
**Decision:** `ui/disks.js` has no import and no build step; every text from the hub is a text node (never markup); the look is a constructable stylesheet using the dashboard's CSS variables (`--ink`, `--muted`, `--line`, `--ok`, `--warn`, `--off`, `--f-body`, `--f-mono`) with the dark theme as fallback. **Why:** the frame's policy forbids inline `<style>` and style attributes, and the module cannot import anything. **Consequence:** the host must load the module as a module script of a document that has a `<body>`; the module reports its height with `resize`.

### Texts are copied from the core's panel (2026-11-26)
**Decision:** the English and Italian texts are those of `servers.disks.*` in the dashboard, plus the few a frame needs (no permission, waiting for the dashboard). **Why:** the tab must say what the panel said until it replaces it. **Consequence:** the Italian of the new texts is a draft for the maintainer to read.

### Version `0.1.0-alpha.1`, an alpha first (2026-11-26)
**Decision:** the first release is an alpha, offered only to hubs that follow `alpha`, installed from the store. **Why:** it has never run on a real hub.
