# disks

The first real plugin of Regulus (issue [#10](https://github.com/Stellatinus/Regulus-Plugins/issues/10), label `disks`). It is the plugin side of Stellatinus/Regulus#497.

**What it does:** adds a **Disks** tab (**Dischi** in Italian) to the node window, between General and SSH terminal. It shows what the **Physical disks** panel of the core shows (`docs/disks.md` in Regulus): for each disk its device, model, serial, size, identity key and kind, whether it is present, why it cannot be identified, a conflict when another server reports the same disk, and the **moves of disks between servers** to and from this server. Each disk is a card as design v3.2 draws it. The design's **S.M.A.R.T.** panel (health, temperature, wear, hours of use, reallocated sectors) and its test buttons are **not built**: the hub does not have that data (Regulus#149) and cannot send a command to a node, so the tab says so in a note instead of showing an invented value. It is **not a disk manager**: it shows statistics and identity, it changes nothing.

**What it is:** a UI-only plugin: no program, one module, `ui/disks.js`, which the dashboard runs in a sandboxed frame with no cookie and no network. It reads only through the dashboard (postMessage protocol v1, two read-only operations, `inventory.disks` and `inventory.disk_moves`), so what a viewer sees is decided by the hub's `inventory.read`. The core keeps collecting the identities and keeps the history; this plugin only presents them. The reasons are in [`decisions.md`](decisions.md).

**What it needs:** the permission `disks.read`, granted to whoever holds the core's `inventory.read`. **It needs no other plugin** (`needs` is empty on purpose). It runs on the hubs only. English and Italian.

**How to try it:** `make test` runs its tests (`node --test`, no install). To try it on a hub, install it from the Regulus store once it is released (an alpha: the hub must follow `alpha`).

**What is verified:** the module's state machine, its view and its escaping of hub text, in English and Italian, with node's test runner; the manifest, the package and the index entry (`make check`, the release chain on this plugin). **What is not:** it has never run in a real browser frame, with the dashboard's host side, or on a real hub; the Italian of the texts the core's panel did not have is a draft.

**Version `0.1.0-alpha.1`**: an alpha, offered only to hubs that follow `alpha`. It is a draft until it has a tag.
