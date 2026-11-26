// Disks: the "Disks" tab of the node window (Regulus plugin `disks`).
//
// ONE dependency-free ES module. It runs in a sandboxed iframe with an opaque
// origin (no cookie, no storage, no network): the only door is postMessage to
// the dashboard, protocol v1 (docs/plugins/README.md in Regulus, "How widget
// code is isolated from the session"):
//
//   dashboard -> module  {regulus:1, type:"init", locale:"en"|"it", theme:{"--var":"value"}, context:{node}}
//                        {regulus:1, type:"response", id, ok:true, data}
//                        {regulus:1, type:"response", id, ok:false, error:{code, message}}
//                        {regulus:1, type:"theme", theme:{...}}
//   module -> dashboard  {regulus:1, type:"ready"}
//                        {regulus:1, type:"request", id, op, params}
//                        {regulus:1, type:"resize", height}
//
// Ops used, both read-only: `inventory.disks` {node} (the JSON of
// GET /api/agents/{id}/disks) and `inventory.disk_moves` {limit} (the JSON of
// GET /api/disks/moves). The dashboard decides whether the viewer may read them
// (inventory.read); a refusal arrives as an error and is shown as such.
//
// Shape of the file: pure functions first (texts, formatting, the state, the
// view as a tree of plain nodes), tested with node's test runner; then a thin
// DOM layer that is started only inside a browser frame. Text from the hub
// (serials, models, server ids) is NEVER put in markup: it is always a text
// node, and `html()` below escapes it for tests.

export const PROTOCOL = 1;
export const MOVES_LIMIT = 200;

const KINDS = ["wwn", "eui64", "nguid", "model-serial", "none"];
const REASONS = ["no-identifier", "virtual", "usb-bridge", "placeholder-serial", "duplicate"];

// The texts are those of the Physical disks panel of the core (servers.disks.*),
// plus the ones a frame needs that the page did not.
export const TEXTS = {
  en: {
    title: "Physical disks",
    intro: "The identity follows the drive, not the device name: it stays the same when the disk is moved to another server.",
    loading: "Loading the disks…",
    loadError: "Could not load the disks: {message}",
    none: "This server has reported no physical disks.",
    forbidden: "You do not have permission to see the disks of this server.",
    waiting: "Waiting for the dashboard…",
    "column.device": "Device",
    "column.disk": "Disk",
    "column.serial": "Serial",
    "column.size": "Size",
    "column.identity": "Identity",
    "column.state": "State",
    "kind.wwn": "WWN",
    "kind.eui64": "EUI-64",
    "kind.nguid": "NGUID",
    "kind.model-serial": "model and serial",
    "kind.none": "none",
    "state.ok": "present",
    "state.absent": "not reported now",
    "state.contested": "Conflict: server {other} reports this disk too",
    "state.notIdentifiable": "not identifiable: {reason}",
    "reason.no-identifier": "the drive reports no identifier",
    "reason.virtual": "a virtual disk (its serial is not the drive's)",
    "reason.usb-bridge": "a USB enclosure without a WWN",
    "reason.placeholder-serial": "the serial is a filler",
    "reason.duplicate": "two disks report the same identifier",
    "reason.other": "no reliable identifier",
    "moves.title": "Moves of disks",
    "moves.loadError": "Could not load the disk moves: {message}",
    "moves.loading": "Loading the moves…",
    "moves.none": "No disk has moved to or from this server.",
    "moves.column.when": "When",
    "moves.column.disk": "Disk",
    "moves.column.from": "From",
    "moves.column.to": "To",
    "moves.unknownDisk": "(unknown disk)",
  },
  it: {
    title: "Dischi fisici",
    intro: "L'identità segue il disco, non il nome del dispositivo: resta la stessa quando il disco passa a un altro server.",
    loading: "Caricamento dei dischi…",
    loadError: "Impossibile caricare i dischi: {message}",
    none: "Questo server non ha segnalato dischi fisici.",
    forbidden: "Non hai il permesso di vedere i dischi di questo server.",
    waiting: "In attesa della dashboard…",
    "column.device": "Dispositivo",
    "column.disk": "Disco",
    "column.serial": "Seriale",
    "column.size": "Dimensione",
    "column.identity": "Identità",
    "column.state": "Stato",
    "kind.wwn": "WWN",
    "kind.eui64": "EUI-64",
    "kind.nguid": "NGUID",
    "kind.model-serial": "modello e seriale",
    "kind.none": "nessuna",
    "state.ok": "presente",
    "state.absent": "non segnalato ora",
    "state.contested": "Conflitto: anche il server {other} segnala questo disco",
    "state.notIdentifiable": "non identificabile: {reason}",
    "reason.no-identifier": "il disco non segnala alcun identificativo",
    "reason.virtual": "un disco virtuale (il seriale non è quello del disco)",
    "reason.usb-bridge": "un contenitore USB senza WWN",
    "reason.placeholder-serial": "il seriale è un valore segnaposto",
    "reason.duplicate": "due dischi segnalano lo stesso identificativo",
    "reason.other": "nessun identificativo affidabile",
    "moves.title": "Spostamenti di dischi",
    "moves.loadError": "Impossibile caricare gli spostamenti dei dischi: {message}",
    "moves.loading": "Caricamento degli spostamenti…",
    "moves.none": "Nessun disco è passato da o verso questo server.",
    "moves.column.when": "Quando",
    "moves.column.disk": "Disco",
    "moves.column.from": "Da",
    "moves.column.to": "A",
    "moves.unknownDisk": "(disco sconosciuto)",
  },
};

export function normalizeLocale(locale) {
  return locale === "it" ? "it" : "en";
}

// t returns the text for a key in a language (English when the Italian one is
// missing), with {name} replaced by vars.name. A value is inserted as text.
export function t(locale, key, vars = {}) {
  const table = TEXTS[normalizeLocale(locale)];
  const text = table[key] ?? TEXTS.en[key] ?? key;
  return text.replace(/\{(\w+)\}/g, (_, name) => (name in vars ? String(vars[name]) : `{${name}}`));
}

const BYTE_UNITS = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];

// Same rule as the dashboard's formatBytes: "512 B", "1.5 KiB", "15.3 GiB".
export function formatBytes(bytes) {
  let value = Math.max(0, Number(bytes) || 0);
  let unit = 0;
  while (value >= 1024 && unit < BYTE_UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const digits = unit === 0 || value >= 100 ? 0 : 1;
  return `${value.toFixed(digits)} ${BYTE_UNITS[unit]}`;
}

export function formatWhen(iso, locale) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return String(iso ?? "-");
  try {
    return d.toLocaleString(normalizeLocale(locale));
  } catch {
    return d.toISOString();
  }
}

// diskState decides the state of one disk as seen from `node` (a server id):
// on the page of the server that contests a disk, the other server is the one
// that holds it. Returns {cls, key, vars}: a severity class and a text key.
export function diskState(disk, node) {
  if (disk.contested) {
    const other = disk.agent === node ? disk.contested_by : disk.agent;
    return { cls: "warn", key: "state.contested", vars: { other: other || "?" } };
  }
  if (!disk.identifiable) {
    const reason = REASONS.includes(disk.reason) ? disk.reason : "other";
    return { cls: "warn", key: "state.notIdentifiable", vars: { reasonKey: `reason.${reason}` } };
  }
  if (!disk.present) return { cls: "off", key: "state.absent", vars: {} };
  return { cls: "ok", key: "state.ok", vars: {} };
}

// ---- The view: a tree of {tag, attrs, kids}; a kid is a node or a string. ----

const h = (tag, attrs, ...kids) => ({ tag, attrs: attrs || {}, kids: kids.flat().filter((k) => k !== null && k !== undefined && k !== false) });

function section(state, title, body) {
  return h("section", { class: "panel" }, h("h3", {}, title), ...body);
}

function diskRow(state, disk) {
  const L = state.locale;
  const st = diskState(disk, state.node);
  const stateText = st.vars.reasonKey
    ? t(L, st.key, { reason: t(L, st.vars.reasonKey) })
    : t(L, st.key, st.vars);
  const model = [disk.vendor, disk.model].filter(Boolean).join(" ") || "-";
  const kind = KINDS.includes(disk.kind) ? t(L, `kind.${disk.kind}`) : String(disk.kind ?? "");
  return h(
    "tr",
    {},
    h("td", { class: "mono" }, String(disk.device ?? "")),
    h("td", {}, model),
    h("td", { class: "mono" }, disk.serial || "-"),
    h("td", {}, disk.size_bytes ? formatBytes(disk.size_bytes) : "-"),
    h(
      "td",
      {},
      disk.identifiable
        ? [h("span", { class: "mono" }, String(disk.key ?? "")), " ", h("span", { class: "muted" }, `(${kind})`)]
        : "-",
    ),
    h("td", {}, h("span", { class: `status ${st.cls}` }, stateText)),
  );
}

function disksTable(state) {
  const L = state.locale;
  const heads = ["device", "disk", "serial", "size", "identity", "state"].map((c) => h("th", { scope: "col" }, t(L, `column.${c}`)));
  return h("table", { class: "cards" }, h("thead", {}, h("tr", {}, ...heads)), h("tbody", {}, ...state.disks.items.map((d) => diskRow(state, d))));
}

function movesTable(state, mine) {
  const L = state.locale;
  const heads = ["when", "disk", "from", "to"].map((c) => h("th", { scope: "col" }, t(L, `moves.column.${c}`)));
  const rows = mine.map((m) =>
    h(
      "tr",
      {},
      h("td", {}, formatWhen(m.at, L)),
      h("td", {}, m.model || m.serial ? [m.model, m.serial].filter(Boolean).join(" ") : t(L, "moves.unknownDisk")),
      h("td", { class: "mono" }, String(m.from ?? "")),
      h("td", { class: "mono" }, String(m.to ?? "")),
    ),
  );
  return h("table", { class: "cards" }, h("thead", {}, h("tr", {}, ...heads)), h("tbody", {}, ...rows));
}

// view turns the state into the tree to draw. No data is ever markup.
export function view(state) {
  const L = state.locale;
  if (!state.ready) return h("main", {}, h("p", { class: "muted", role: "status" }, t(L, "waiting")));

  const body = [h("p", { class: "muted" }, t(L, "intro"))];
  const d = state.disks;
  if (d.kind === "loading") body.push(h("p", { role: "status" }, t(L, "loading")));
  else if (d.kind === "forbidden") body.push(h("p", { role: "alert" }, t(L, "forbidden")));
  else if (d.kind === "error") body.push(h("p", { role: "alert" }, t(L, "loadError", { message: d.message })));
  else if (d.items.length === 0) body.push(h("p", {}, t(L, "none")));
  else body.push(disksTable(state));

  const m = state.moves;
  body.push(h("h4", {}, t(L, "moves.title")));
  if (m.kind === "loading") body.push(h("p", { role: "status" }, t(L, "moves.loading")));
  else if (m.kind === "forbidden") body.push(h("p", { role: "alert" }, t(L, "forbidden")));
  else if (m.kind === "error") body.push(h("p", { role: "alert" }, t(L, "moves.loadError", { message: m.message })));
  else {
    const mine = m.items.filter((x) => x.from === state.node || x.to === state.node);
    body.push(mine.length === 0 ? h("p", {}, t(L, "moves.none")) : movesTable(state, mine));
  }
  return h("main", {}, section(state, t(L, "title"), body));
}

// html serializes a tree with every text escaped. The frame does not use it
// (it builds DOM nodes); the tests do, to check that no data becomes markup.
export function html(node) {
  if (typeof node === "string") return escapeHtml(node);
  const attrs = Object.entries(node.attrs)
    .map(([k, v]) => ` ${k}="${escapeHtml(String(v))}"`)
    .join("");
  return `<${node.tag}${attrs}>${node.kids.map(html).join("")}</${node.tag}>`;
}

export function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

// ---- The state machine: messages in, requests out. No DOM, no window. ----

const THEME_NAME = /^--[a-z0-9-]{1,40}$/;

// sanitizeTheme keeps only CSS custom properties with short plain values: the
// theme is the dashboard's, but the module does not trust a message with
// anything that could load a resource.
export function sanitizeTheme(theme) {
  const out = {};
  if (!theme || typeof theme !== "object") return out;
  for (const [name, value] of Object.entries(theme)) {
    if (!THEME_NAME.test(name) || typeof value !== "string" || value.length > 200) continue;
    if (/url\s*\(|@import|expression\s*\(|[;{}]/i.test(value)) continue;
    out[name] = value;
  }
  return out;
}

const idle = () => ({ kind: "loading", items: [], message: "" });

export function initialState() {
  return { ready: false, locale: "en", node: "", theme: {}, disks: idle(), moves: idle() };
}

function failure(error) {
  const code = error && typeof error === "object" ? String(error.code ?? "") : "";
  const status = error && typeof error === "object" ? Number(error.status) : NaN;
  const message = error && typeof error === "object" && error.message ? String(error.message) : "?";
  if (status === 403 || code === "forbidden" || code === "permission_denied") return { kind: "forbidden", items: [], message };
  return { kind: "error", items: [], message };
}

function listOf(data, field) {
  if (Array.isArray(data)) return data;
  if (data && typeof data === "object" && Array.isArray(data[field])) return data[field];
  return null;
}

// createApp holds the state and talks only through `post(message)`, which the
// frame binds to window.parent. `receive(message)` handles one message from
// the dashboard and returns true when the state changed and must be redrawn.
export function createApp({ post }) {
  let state = initialState();
  let nextId = 1;
  let pending = new Map(); // request id -> {field, listField}

  const request = (op, params, field, listField) => {
    const id = nextId++;
    pending.set(id, { field, listField });
    post({ regulus: PROTOCOL, type: "request", id, op, params });
  };

  return {
    get state() {
      return state;
    },
    start() {
      post({ regulus: PROTOCOL, type: "ready" });
    },
    receive(msg) {
      if (!msg || typeof msg !== "object" || msg.regulus !== PROTOCOL) return false;
      switch (msg.type) {
        case "init": {
          const node = msg.context && typeof msg.context.node === "string" ? msg.context.node : "";
          // A new node (or a second init) drops what is still in flight: a
          // late answer for another node must never be drawn here.
          pending = new Map();
          state = { ready: true, locale: normalizeLocale(msg.locale), node, theme: sanitizeTheme(msg.theme), disks: idle(), moves: idle() };
          request("inventory.disks", { node }, "disks", "disks");
          request("inventory.disk_moves", { limit: MOVES_LIMIT }, "moves", "moves");
          return true;
        }
        case "theme":
          state = { ...state, theme: sanitizeTheme(msg.theme) };
          return true;
        case "response": {
          const p = pending.get(msg.id);
          if (!p) return false;
          pending.delete(msg.id);
          if (msg.ok === true) {
            const items = listOf(msg.data, p.listField);
            state = { ...state, [p.field]: items ? { kind: "ready", items, message: "" } : failure({ message: "unexpected answer" }) };
          } else {
            state = { ...state, [p.field]: failure(msg.error) };
          }
          return true;
        }
        default:
          return false;
      }
    },
  };
}

// ---- The DOM layer: only runs in a browser frame, never under test. ----

const ATTRS = new Set(["class", "role", "scope", "title"]);

export function toDom(doc, node) {
  if (typeof node === "string") return doc.createTextNode(node);
  const el = doc.createElement(node.tag);
  for (const [k, v] of Object.entries(node.attrs)) if (ATTRS.has(k)) el.setAttribute(k, String(v));
  for (const kid of node.kids) el.appendChild(toDom(doc, kid));
  return el;
}

// The frame has no inline <style> (the host's policy forbids it) and no
// stylesheet to load: the look is a constructable stylesheet, which is CSSOM,
// not markup. Every colour comes from the dashboard's variables, with the
// dashboard's dark theme as the fallback.
export const STYLE = `
:root { color-scheme: dark light; }
body { margin: 0; padding: 0 2px 8px; font: 14px/1.45 var(--f-body, system-ui, sans-serif); color: var(--ink, #e6e8ee); background: transparent; }
h3 { font-size: 15px; margin: 4px 0 6px; } h4 { font-size: 13px; margin: 18px 0 6px; }
p { margin: 6px 0; } .muted { color: var(--muted, #98a0b3); }
.mono { font-family: var(--f-mono, ui-monospace, monospace); font-size: 12.5px; word-break: break-all; }
table { width: 100%; border-collapse: collapse; margin: 6px 0; }
th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid var(--line, #2a2f3d); vertical-align: top; }
th { font-weight: 600; color: var(--muted, #98a0b3); font-size: 12px; }
.status { display: inline-block; padding: 1px 8px; border-radius: 999px; border: 1px solid var(--sev, var(--line, #2a2f3d)); color: var(--sev, var(--ink, #e6e8ee)); }
.status.ok { --sev: var(--ok, #3ecf8e); } .status.warn { --sev: var(--warn, #f5a524); } .status.off { --sev: var(--off, #8a90a2); }
`;

// mount binds the state machine to a browser frame. It is the only place that
// touches window and document.
export function mount(win, doc) {
  let target = "*"; // the dashboard's origin, known from its first message
  const app = createApp({ post: (m) => win.parent.postMessage(m, target) });
  let lastHeight = -1;

  try {
    const sheet = new win.CSSStyleSheet();
    sheet.replaceSync(STYLE);
    doc.adoptedStyleSheets = [sheet];
  } catch {
    // No constructable stylesheets: the tab is plain but readable.
  }

  // report tells the dashboard how tall the tab is, when that changed.
  const report = () => {
    const height = Math.ceil(doc.documentElement.scrollHeight);
    if (height !== lastHeight) {
      lastHeight = height;
      win.parent.postMessage({ regulus: PROTOCOL, type: "resize", height }, target);
    }
  };

  const draw = () => {
    const st = app.state;
    doc.documentElement.lang = st.locale;
    for (const [name, value] of Object.entries(st.theme)) doc.documentElement.style.setProperty(name, value);
    doc.body.replaceChildren(toDom(doc, view(st)));
    report();
  };

  win.addEventListener("message", (event) => {
    if (event.source !== win.parent) return; // only the dashboard
    if (event.origin && event.origin !== "null") target = event.origin;
    if (app.receive(event.data)) draw();
  });
  if (typeof win.ResizeObserver === "function") new win.ResizeObserver(report).observe(doc.body);

  draw();
  app.start();
  return app;
}

if (typeof window !== "undefined" && typeof document !== "undefined" && window.parent !== window) {
  mount(window, document);
}
