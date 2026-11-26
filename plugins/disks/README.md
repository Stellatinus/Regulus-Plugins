# disks

The first real plugin of Regulus (issue [#10](https://github.com/Stellatinus/Regulus-Plugins/issues/10), label `disks`). It is the plugin side of Stellatinus/Regulus#497.

**What it will do:** add a **Disks** tab (**Dischi** in Italian) to the node window, between General and SSH terminal. It shows what the **Physical disks** panel of the core shows today (`docs/disks.md` in Regulus): for each disk its device, kind, size, serial and identity key, whether it is present, why it cannot be identified, a conflict when another server reports the same disk, and the **moves of disks between servers**. It is **not a disk manager**: it shows statistics and identity, it changes nothing.

**What it needs:** the permission `disks.read`, mapped to the core's `inventory.read`; the plugin runs on the hubs only (Regulus#496). English and Italian texts.

**What exists today:** the manifest only, a **draft with no tag**: it is in no index. **There is no program and no UI module**, and nothing here has been built or run. It is blocked by Regulus#463 (the SDK importable from here), #261 (what a plugin can read and call), #21 (the tab extension point and the widget loader) and #496. The cut between what stays in the core (the agent's collection of identities, `internal/diskid`, the tables `disks` and `disk_moves`) and what moves here is a design in `docs/disks.md` of Regulus, not yet agreed.

**Version `0.1.0-alpha.1`**: an alpha, offered only to hubs that follow `alpha` once it is released.
