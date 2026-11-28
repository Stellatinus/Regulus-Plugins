# Changelog

## 0.1.0-alpha.1

First alpha. Adds a **Disks** tab (**Dischi** in Italian) to the node window,
with what the **Physical disks** panel of the core shows: for each disk its
device, model, serial, size, identity key and kind (WWN, EUI-64, NGUID, or model
and serial), and its state (present, not reported now, not identifiable with the
reason, or in conflict with another server that reports the same disk); and the
moves of disks to and from the server.

- Each disk is a **card** as the maintainer's design v3.2 draws it (device and
  how it is attached, model, serial, state, size, first and last seen, identity),
  with a note that says why the **S.M.A.R.T.** values of the design (health,
  temperature, wear, hours, reallocated sectors) and its test buttons are not
  there: the hub has no such data (Regulus#149) and no command can be sent to a
  node. Nothing is shown rather than an invented value.
- UI only: no program. The core still collects the identities and stores the
  history; the tab reads it through the dashboard, read-only.
- Needs the permission `disks.read`, granted to whoever holds `inventory.read`.
  Needs no other plugin.
- English and Italian. The Italian of the texts that the core's panel did not
  have (no permission, waiting for the dashboard) is a draft.
- **Not tried on a real hub yet.** An alpha: offered only to hubs that follow
  `alpha`.
