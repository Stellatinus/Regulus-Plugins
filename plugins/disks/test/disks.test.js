// Tests of the Disks tab, with node's test runner: no DOM, no network, no
// dependency. Run: node --test plugins/disks/test (make test does).
import test from "node:test";
import assert from "node:assert/strict";
import {
  PROTOCOL, TEXTS, t, formatBytes, diskState, view, html, createApp, sanitizeTheme, toDom, MOVES_LIMIT,
} from "../ui/disks.js";

const disk = (over = {}) => ({
  key: "wwn:5000c500a1b2c3d4", kind: "wwn", identifiable: true, conflict: false, contested: false,
  agent: "n1", device: "sda", vendor: "ATA", model: "ST4000VN008", serial: "ZA1ABC23", size_bytes: 4000787030016,
  present: true, ...over,
});

const ready = (over = {}) => ({
  ready: true, locale: "en", node: "n1", theme: {},
  disks: { kind: "ready", items: [disk()], message: "" },
  moves: { kind: "ready", items: [], message: "" }, ...over,
});

test("English and Italian have exactly the same keys", () => {
  assert.deepEqual(Object.keys(TEXTS.it).sort(), Object.keys(TEXTS.en).sort());
});

test("t fills {vars} and falls back to English, then to the key", () => {
  assert.equal(t("it", "state.contested", { other: "n2" }), "Conflitto: anche il server n2 segnala questo disco");
  assert.equal(t("xx", "title"), "Physical disks");
  assert.equal(t("en", "no.such.key"), "no.such.key");
});

test("formatBytes is the dashboard's", () => {
  assert.equal(formatBytes(512), "512 B");
  assert.equal(formatBytes(1536), "1.5 KiB");
  assert.equal(formatBytes(4000787030016), "3.6 TiB");
  assert.equal(formatBytes(-5), "0 B");
});

test("diskState: contested, not identifiable, absent, present", () => {
  assert.deepEqual(diskState(disk({ contested: true, contested_by: "n2" }), "n1"), { cls: "warn", key: "state.contested", vars: { other: "n2" } });
  // seen from the server that contests it, the holder is the other one
  assert.equal(diskState(disk({ contested: true, agent: "n2", contested_by: "n1" }), "n1").vars.other, "n2");
  assert.equal(diskState(disk({ identifiable: false, reason: "virtual" }), "n1").vars.reasonKey, "reason.virtual");
  assert.equal(diskState(disk({ identifiable: false, reason: "made-up" }), "n1").vars.reasonKey, "reason.other");
  assert.equal(diskState(disk({ present: false }), "n1").cls, "off");
  assert.equal(diskState(disk(), "n1").cls, "ok");
});

test("the view shows device, disk, serial, size, identity, kind and state", () => {
  const out = html(view(ready()));
  for (const want of ["sda", "ATA ST4000VN008", "ZA1ABC23", "3.6 TiB", "wwn:5000c500a1b2c3d4", "(WWN)", "present"]) assert.ok(out.includes(want), want);
});

test("a disk that is not identifiable shows the reason, in Italian too", () => {
  const st = ready({ locale: "it", disks: { kind: "ready", items: [disk({ identifiable: false, reason: "usb-bridge", key: "slot:n1:sda" })], message: "" } });
  const out = html(view(st));
  assert.ok(out.includes("non identificabile: un contenitore USB senza WWN"));
  assert.ok(!out.includes("slot:n1:sda"), "the slot key says nothing about the drive");
});

test("hub text never becomes markup", () => {
  const evil = '<img src=x onerror="alert(1)">';
  const st = ready({
    disks: { kind: "ready", items: [disk({ serial: evil, model: evil, device: evil })], message: "" },
    moves: { kind: "ready", items: [{ id: 1, at: "2026-11-26T10:00:00Z", model: evil, serial: "", from: evil, to: "n1" }], message: "" },
  });
  const out = html(view(st));
  assert.ok(!out.includes("<img"), out);
  assert.ok(out.includes("&lt;img"));
  const err = html(view(ready({ disks: { kind: "error", items: [], message: evil } })));
  assert.ok(!err.includes("<img"));
});

test("loading, empty, forbidden and error states", () => {
  const L = (d, m) => html(view(ready({ disks: d, moves: m })));
  const ok = { kind: "ready", items: [], message: "" };
  assert.ok(L({ kind: "loading", items: [] }, ok).includes("Loading the disks"));
  assert.ok(L(ok, ok).includes("This server has reported no physical disks."));
  assert.ok(L({ kind: "forbidden", items: [] }, ok).includes("You do not have permission"));
  assert.ok(L({ kind: "error", items: [], message: "boom" }, ok).includes("Could not load the disks: boom"));
  assert.ok(L(ok, { kind: "error", items: [], message: "boom" }).includes("Could not load the disk moves: boom"));
  assert.ok(html(view({ ...ready(), ready: false })).includes("Waiting for the dashboard"));
});

test("moves: only those to or from this node, newest as given", () => {
  const mv = (id, from, to) => ({ id, at: "2026-11-26T10:00:00Z", key: "k", model: "M", serial: `S${id}`, from, to });
  const st = ready({ moves: { kind: "ready", items: [mv(1, "n1", "n2"), mv(2, "n3", "n4"), mv(3, "n2", "n1")], message: "" } });
  const out = html(view(st));
  assert.ok(out.includes("S1") && out.includes("S3") && !out.includes("S2"));
  assert.ok(html(view(ready({ moves: { kind: "ready", items: [mv(2, "n3", "n4")], message: "" } }))).includes("No disk has moved"));
  assert.ok(html(view(ready({ moves: { kind: "ready", items: [{ ...mv(5, "n1", "n2"), model: "", serial: "" }], message: "" } }))).includes("(unknown disk)"));
});

function harness() {
  const sent = [];
  const app = createApp({ post: (m) => sent.push(m) });
  return { app, sent };
}
const init = (over = {}) => ({ regulus: PROTOCOL, type: "init", locale: "it", theme: { "--ink": "#fff" }, context: { node: "n1" }, ...over });

test("start says ready; init asks for the disks of the node and for the moves", () => {
  const { app, sent } = harness();
  app.start();
  assert.deepEqual(sent, [{ regulus: 1, type: "ready" }]);
  assert.equal(app.receive(init()), true);
  assert.deepEqual(sent.slice(1), [
    { regulus: 1, type: "request", id: 1, op: "inventory.disks", params: { node: "n1" } },
    { regulus: 1, type: "request", id: 2, op: "inventory.disk_moves", params: { limit: MOVES_LIMIT } },
  ]);
  assert.equal(app.state.locale, "it");
  assert.deepEqual(app.state.theme, { "--ink": "#fff" });
});

test("responses fill the state: whole route JSON or a bare list", () => {
  const { app } = harness();
  app.receive(init());
  app.receive({ regulus: 1, type: "response", id: 1, ok: true, data: { agent: "n1", disks: [disk()] } });
  app.receive({ regulus: 1, type: "response", id: 2, ok: true, data: [] });
  assert.equal(app.state.disks.kind, "ready");
  assert.equal(app.state.disks.items.length, 1);
  assert.equal(app.state.moves.kind, "ready");
});

test("403 and forbidden are shown as no permission; other errors as errors; odd data as an error", () => {
  const { app } = harness();
  app.receive(init());
  app.receive({ regulus: 1, type: "response", id: 1, ok: false, error: { code: "forbidden", message: "no" } });
  app.receive({ regulus: 1, type: "response", id: 2, ok: false, error: { code: "http", status: 500, message: "down" } });
  assert.equal(app.state.disks.kind, "forbidden");
  assert.equal(app.state.moves.kind, "error");
  assert.equal(app.state.moves.message, "down");
  const b = harness();
  b.app.receive(init());
  b.app.receive({ regulus: 1, type: "response", id: 1, ok: false, error: { status: 403, message: "x" } });
  assert.equal(b.app.state.disks.kind, "forbidden");
  b.app.receive({ regulus: 1, type: "response", id: 2, ok: true, data: { nope: 1 } });
  assert.equal(b.app.state.moves.kind, "error");
});

test("messages that are not protocol 1, or unknown, or for an unknown request, change nothing", () => {
  const { app } = harness();
  app.receive(init());
  for (const m of [null, "x", {}, { regulus: 2, type: "init" }, { regulus: 1, type: "weird" }, { regulus: 1, type: "response", id: 99, ok: true, data: [] }]) {
    assert.equal(app.receive(m), false);
  }
  assert.equal(app.state.disks.kind, "loading");
});

test("a second init (another node) drops answers still in flight", () => {
  const { app, sent } = harness();
  app.receive(init({ context: { node: "n1" } }));
  app.receive(init({ context: { node: "n2" } }));
  assert.equal(sent.at(-2).params.node, "n2");
  // the answer to the first request (id 1) must not be drawn for n2
  assert.equal(app.receive({ regulus: 1, type: "response", id: 1, ok: true, data: [disk({ device: "OLD" })] }), false);
  assert.equal(app.state.disks.kind, "loading");
  assert.equal(app.receive({ regulus: 1, type: "response", id: 3, ok: true, data: [] }), true);
});

test("theme messages replace the theme; only plain custom properties pass", () => {
  assert.deepEqual(sanitizeTheme({ "--ink": "#fff", "--bg": "url(http://x/y)", color: "red", "--a;b": "1", "--ok": "1;2", "--long": "x".repeat(300), "--n": 5 }), { "--ink": "#fff" });
  assert.deepEqual(sanitizeTheme(null), {});
  const { app } = harness();
  app.receive(init());
  assert.equal(app.receive({ regulus: 1, type: "theme", theme: { "--ink": "#000" } }), true);
  assert.deepEqual(app.state.theme, { "--ink": "#000" });
});

test("toDom builds elements and text nodes, and only sets known attributes", () => {
  const made = [];
  const doc = {
    createTextNode: (s) => ({ text: s }),
    createElement: (tag) => {
      const el = { tag, attrs: {}, kids: [], setAttribute(k, v) { this.attrs[k] = v; }, appendChild(k) { this.kids.push(k); } };
      made.push(el);
      return el;
    },
  };
  const el = toDom(doc, { tag: "p", attrs: { class: "a", onclick: "evil()" }, kids: ["<b>x</b>"] });
  assert.deepEqual(el.attrs, { class: "a" });
  assert.deepEqual(el.kids, [{ text: "<b>x</b>" }]);
});
