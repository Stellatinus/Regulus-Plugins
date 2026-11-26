# disks

The first real plugin of Regulus (issue [#10](https://github.com/Stellatinus/Regulus-Plugins/issues/10), label `disks`). It is the plugin side of Stellatinus/Regulus#497.

**What it does:** adds a **Disks** tab (**Dischi** in Italian) to the node window, between General and SSH terminal. It shows what the **Physical disks** panel of the core shows (`docs/disks.md` in Regulus): for each disk its device, model, serial, size, identity key and kind, whether it is present, why it cannot be identified, a conflict when another server reports the same disk, and the **moves of disks between servers** to and from this server. It is **not a disk manager**: it shows statistics and identity, it changes nothing.

**What it is:** a UI-only plugin: no program, one module, `ui/disks.js`, which the dashboard runs in a sandboxed frame with no cookie and no network. It reads only through the dashboard (postMessage protocol v1, two read-only operations, `inventory.disks` and `inventory.disk_moves`), so what a viewer sees is decided by the hub's `inventory.read`. The core keeps collecting the identities and keeps the history; this plugin only presents them. The reasons are in [`decisions.md`](decisions.md).

**What it needs:** the permission `disks.read`, granted to whoever holds the core's `inventory.read`. **It needs no other plugin** (`needs` is empty on purpose). It runs on the hubs only. English and Italian.

**How to try it:** `make test` runs its tests (`node --test`, no install). To try it on a hub, install it from the Regulus store once it is released (an alpha: the hub must follow `alpha`).

**What is verified:** the module's state machine, its view and its escaping of hub text, in English and Italian, with node's test runner; the manifest, the package and the index entry (`make check`, the release chain on this plugin). **On 2026-11-26 it also ran in Chromium (Playwright), on the real hub binary of Regulus v0.4.1-alpha.1 (built from `f1ddaf7`), copied by hand into the hub's plugins directory** (not installed from the store): the Disks tab appears in the node window, the module loads in the sandboxed frame, reads through the real bridge (`inventory.disks`, `inventory.disk_moves`) and shows, in English, a disk present, one not identifiable (virtual), one not reported now and a move, with no error in the browser console; with no rows it shows the empty texts. The disk rows were written into the hub's database by hand, because the agent of that run was in a container and reports no physical disks. **What is not:** installation from the store, a real server with real disks, the conflict (`contested`) state, Italian and the light theme in a real frame, Firefox and Safari, and a real phone; the Italian of the texts the core's panel did not have is a draft. A check that stays in CI is Stellatinus/Regulus-Plugins#15.

**Version `0.1.0-alpha.1`**: an alpha, offered only to hubs that follow `alpha`. It is a draft until it has a tag.
