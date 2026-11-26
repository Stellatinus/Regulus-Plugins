package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)

const sum1 = "1111111111111111111111111111111111111111111111111111111111111111"
const sum2 = "2222222222222222222222222222222222222222222222222222222222222222"

// fixture writes the small files of one release into the assets directory and
// returns the release as the GitHub API would list it.
func fixture(t *testing.T, assets, id, ver, manifestYAML string, arches ...string) release {
	t.Helper()
	tag := "plugin/" + id + "/v" + ver
	dir := assetDir(assets, tag)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%s_%s_plugin.yaml", id, ver)), []byte(manifestYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	r := release{Tag: tag, Prerelease: strings.Contains(ver, "-"), Published: "2026-11-30T10:00:00Z"}
	for i, arch := range arches {
		name := fmt.Sprintf("%s_%s_linux_%s.tar.gz", id, ver, arch)
		sum := []string{sum1, sum2}[i%2]
		if err := os.WriteFile(filepath.Join(dir, name+".sha256"), []byte(sum+"  "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		r.Assets = append(r.Assets, asset{Name: name, Size: int64(1000 + i), URL: "https://example.test/" + name})
		r.Assets = append(r.Assets, asset{Name: name + ".sha256", Size: 90, URL: "https://example.test/" + name + ".sha256"})
	}
	return r
}

func manifestFor(id, ver, extra string) string {
	return fmt.Sprintf("id: %s\nversion: %s\nname: %s name\ndescription: About %s.\napi: 1\nrequires: \">=0.4.0\"\ncomponents:\n  agent:\n    command: bin/%s\npermissions:\n  - id: %s.use\n    description: Use it.\n%s", id, ver, id, id, id, id, extra)
}

func opts(assets string, rels ...release) buildOptions {
	return buildOptions{releases: rels, assets: assets, yanked: map[string]string{}, repo: "Stellatinus/Regulus-Plugins", now: t0, validFor: 21 * 24 * time.Hour}
}

func TestBuildIndex(t *testing.T) {
	a := t.TempDir()
	rels := []release{
		fixture(t, a, "hello-world", "0.1.0-alpha.1", manifestFor("hello-world", "0.1.0-alpha.1", ""), "amd64", "arm64"),
		fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.0", "provides:\n  - hello.greet@1\n"), "amd64", "arm64"),
		fixture(t, a, "hello-world", "0.2.0-beta.1", manifestFor("hello-world", "0.2.0-beta.1", "kind: integration\naudience: everyone\nneeds:\n  - capability: mail.send@1\n    optional: true\n"), "amd64"),
		fixture(t, a, "hello-guest", "0.1.0-alpha.1", manifestFor("hello-guest", "0.1.0-alpha.1", ""), "amd64"),
		{Tag: "v1.0.0"}, // a Regulus tag: not ours
		{Tag: "plugin/hello-world/v9.9.9", Draft: true}, // a draft is not published
	}
	ix, err := buildIndex(opts(a, rels...))
	if err != nil {
		t.Fatal(err)
	}
	if ix.Format != 1 || ix.Source != "regulus-plugins" || ix.IssuedAt != "2026-12-01T09:00:00Z" || ix.Expires != "2026-12-22T09:00:00Z" {
		t.Fatalf("header = %+v", ix)
	}
	if len(ix.Plugins) != 2 || ix.Plugins[0].ID != "hello-guest" || ix.Plugins[1].ID != "hello-world" {
		t.Fatalf("plugins are not sorted by id: %+v", ix.Plugins)
	}
	hw := ix.Plugins[1]
	var got []string
	for _, v := range hw.Versions {
		got = append(got, v.Version+"/"+v.Channel)
	}
	if want := "0.2.0-beta.1/beta,0.1.0/stable,0.1.0-alpha.1/alpha"; strings.Join(got, ",") != want {
		t.Fatalf("versions = %v, want newest first: %s", got, want)
	}
	// The plugin's own fields come from its newest version.
	if hw.Kind != "integration" || hw.Audience != "everyone" || hw.Name != "hello-world name" {
		t.Fatalf("plugin = %+v", hw)
	}
	if hw.Homepage != "https://github.com/Stellatinus/Regulus-Plugins/tree/main/plugins/hello-world" {
		t.Fatalf("homepage = %s", hw.Homepage)
	}
	// A plugin that says nothing about its audience is for administrators.
	if ix.Plugins[0].Audience != "admin" || ix.Plugins[0].Kind != "" {
		t.Fatalf("defaults = %+v", ix.Plugins[0])
	}
	beta := hw.Versions[0]
	if len(beta.Needs) != 1 || beta.Needs[0].Capability != "mail.send@1" || !beta.Needs[0].Optional {
		t.Fatalf("needs = %+v", beta.Needs)
	}
	stable := hw.Versions[1]
	if len(stable.Provides) != 1 || stable.Provides[0] != "hello.greet@1" || len(stable.Needs) != 0 || stable.Yanked {
		t.Fatalf("stable = %+v", stable)
	}
	if len(stable.Artifacts) != 2 || stable.Artifacts[0].Arch != "amd64" || stable.Artifacts[0].SHA256 != sum1 || stable.Artifacts[1].SHA256 != sum2 ||
		stable.Artifacts[0].OS != "linux" || stable.Artifacts[0].Size != 1000 || !strings.HasPrefix(stable.Artifacts[0].URL, "https://") {
		t.Fatalf("artifacts = %+v", stable.Artifacts)
	}
	if got := strings.Join(stable.Components, ","); got != "agent" || stable.API != 1 || stable.Requires != ">=0.4.0" || len(stable.Permissions) != 1 {
		t.Fatalf("manifest fields = %+v", stable)
	}

	// The JSON has every field of the design, with [] and not null.
	data, _ := json.Marshal(ix)
	for _, want := range []string{`"needs":[]`, `"provides":[]`, `"yanked":false`, `"format":1`} {
		if !bytes.Contains(data, []byte(want)) {
			t.Errorf("index.json lacks %s", want)
		}
	}
	if err := verifyIndex(data, t0.Add(time.Minute), time.Hour); err != nil {
		t.Fatalf("a fresh index is refused: %v", err)
	}
}

func TestYankedVersionsStayListed(t *testing.T) {
	a := t.TempDir()
	r1 := fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.0", ""), "amd64")
	o := opts(a, r1)
	o.yanked = map[string]string{"hello-world 0.1.0": "removed from the catalog"}
	ix, err := buildIndex(o)
	if err != nil {
		t.Fatal(err)
	}
	v := ix.Plugins[0].Versions[0]
	if !v.Yanked || v.YankReason != "removed from the catalog" || len(v.Artifacts) != 1 {
		t.Fatalf("a yanked version must stay listed and downloadable: %+v", v)
	}
	// A typo in yanked.txt is an error, not a silently ignored line.
	o = opts(a, r1)
	o.yanked = map[string]string{"hello-world 0.1.1": ""}
	if _, err := buildIndex(o); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("an unknown yanked version: %v", err)
	}
}

func TestBuildIndexRefusesInconsistentReleases(t *testing.T) {
	good := func(a string) release {
		return fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.0", ""), "amd64")
	}
	t.Run("manifest of another version", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.1", ""), "amd64")
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "plugin.yaml says") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("prerelease flag disagrees with the version", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		r.Prerelease = true
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "prerelease") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("no package", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		r.Assets = nil
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "no package") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("missing checksum", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		os.Remove(filepath.Join(assetDir(a, r.Tag), "hello-world_0.1.0_linux_amd64.tar.gz.sha256"))
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "no checksum") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("checksum of another file", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		os.WriteFile(filepath.Join(assetDir(a, r.Tag), "hello-world_0.1.0_linux_amd64.tar.gz.sha256"), []byte(sum1+"  other.tar.gz\n"), 0o644)
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), ".sha256 is not") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("missing manifest asset", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		os.Remove(filepath.Join(assetDir(a, r.Tag), "hello-world_0.1.0_plugin.yaml"))
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "plugin.yaml") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("bad package name", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		r.Assets = append(r.Assets, asset{Name: "hello-world_0.1.0_linux.tar.gz", URL: "https://x"})
		if _, err := buildIndex(opts(a, r)); err == nil || !strings.Contains(err.Error(), "not named") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("two releases of one version", func(t *testing.T) {
		a := t.TempDir()
		r := good(a)
		if _, err := buildIndex(opts(a, r, r)); err == nil || !strings.Contains(err.Error(), "two releases") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("a tag that is not a version", func(t *testing.T) {
		a := t.TempDir()
		if _, err := buildIndex(opts(a, release{Tag: "plugin/hello-world/vlatest"})); err == nil || !strings.Contains(err.Error(), "not a version") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestEmptyIndexIsValid(t *testing.T) {
	ix, err := buildIndex(opts(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(ix)
	if !bytes.Contains(data, []byte(`"plugins":[]`)) {
		t.Fatalf("an index with no plugin must say plugins: [] and not null: %s", data)
	}
}

func TestVerifyIndex(t *testing.T) {
	a := t.TempDir()
	ix, _ := buildIndex(opts(a, fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.0", ""), "amd64")))
	data, _ := json.Marshal(ix)
	now := t0.Add(10 * time.Minute)
	if err := verifyIndex(data, now, time.Hour); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		now time.Time
		mod func(string) string
		max time.Duration
	}{
		"issued too long ago":  {now: t0.Add(3 * time.Hour), max: time.Hour},
		"issued in the future": {now: t0.Add(-time.Hour), max: time.Hour},
		"expired":              {now: t0.Add(22 * 24 * time.Hour), max: 30 * 24 * time.Hour},
		"valid for too long":   {now: now, max: time.Hour, mod: func(s string) string { return strings.Replace(s, "2026-12-22T09:00:00Z", "2027-12-22T09:00:00Z", 1) }},
		"wrong format":         {now: now, max: time.Hour, mod: func(s string) string { return strings.Replace(s, `"format":1`, `"format":2`, 1) }},
		"no source":            {now: now, max: time.Hour, mod: func(s string) string { return strings.Replace(s, `"source":"regulus-plugins"`, `"source":""`, 1) }},
		"insecure url":         {now: now, max: time.Hour, mod: func(s string) string { return strings.Replace(s, "https://example.test", "http://example.test", 1) }},
		"not json":             {now: now, max: time.Hour, mod: func(string) string { return "{" }},
	} {
		d := string(data)
		if c.mod != nil {
			d = c.mod(d)
		}
		if err := verifyIndex([]byte(d), c.now, c.max); err == nil {
			t.Errorf("%s: the index was accepted", name)
		}
	}
}

func TestReadYanked(t *testing.T) {
	dir := t.TempDir()
	write := func(s string) string {
		f := filepath.Join(dir, "y.txt")
		os.WriteFile(f, []byte(s), 0o644)
		return f
	}
	got, err := readYanked(write("# comment\n\nhello-world 0.1.0 removed, test only\nother 1.0.0-beta.1\n"))
	if err != nil || got["hello-world 0.1.0"] != "removed, test only" || got["other 1.0.0-beta.1"] != "" || len(got) != 2 {
		t.Fatalf("got %v, %v", got, err)
	}
	for _, bad := range []string{"hello-world\n", "Hello 0.1.0\n", "hello-world v0.1.0\n", "hello-world 1.0\n", "a-b 0.1.0\na-b 0.1.0\n"} {
		if _, err := readYanked(write(bad)); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
	if got, err := readYanked(filepath.Join(dir, "absent.txt")); err != nil || len(got) != 0 {
		t.Fatalf("an absent file means nothing yanked: %v %v", got, err)
	}
}

func TestRunCommands(t *testing.T) {
	root := t.TempDir()
	var out, errb bytes.Buffer
	dir := writePlugin(t, root, "hello-world", goodManifest, map[string]string{"README.md": "x", "CHANGELOG.md": "## 0.1.0-alpha.1\n\nFirst.\n"})
	if code := run([]string{"check", dir}, &out, &errb); code != 0 {
		t.Fatalf("check = %d: %s", code, errb.String())
	}
	if code := run([]string{"check", "-release", "0.1.0-alpha.1", dir}, &out, &errb); code != 0 {
		t.Fatalf("check -release = %d: %s", code, errb.String())
	}
	out.Reset()
	if code := run([]string{"notes", dir, "0.1.0-alpha.1"}, &out, &errb); code != 0 || strings.TrimSpace(out.String()) != "First." {
		t.Fatalf("notes = %d %q", code, out.String())
	}
	errb.Reset()
	bad := writePlugin(t, root, "bad", "id: bad\n", map[string]string{})
	if code := run([]string{"check", dir, bad}, &out, &errb); code != 1 || !strings.Contains(errb.String(), "bad:") {
		t.Fatalf("a bad plugin must fail the run: %d %s", code, errb.String())
	}
	if code := run(nil, &out, &errb); code != 2 {
		t.Fatalf("no arguments = %d", code)
	}
	if code := run([]string{"nothing"}, &out, &errb); code != 2 {
		t.Fatalf("unknown command = %d", code)
	}

	// index end to end through the command line, with a fixed time.
	a := t.TempDir()
	r := fixture(t, a, "hello-world", "0.1.0", manifestFor("hello-world", "0.1.0", ""), "amd64")
	rj, _ := json.Marshal([]release{r})
	rf := filepath.Join(t.TempDir(), "releases.json")
	os.WriteFile(rf, rj, 0o644)
	yf := filepath.Join(t.TempDir(), "yanked.txt")
	os.WriteFile(yf, []byte("# nothing\n"), 0o644)
	of := filepath.Join(t.TempDir(), "index.json")
	if code := run([]string{"index", "-releases", rf, "-assets", a, "-repo", "Stellatinus/Regulus-Plugins", "-yanked", yf, "-now", "2026-12-01T09:00:00Z", "-o", of}, &out, &errb); code != 0 {
		t.Fatalf("index = %d: %s", code, errb.String())
	}
	data, _ := os.ReadFile(of)
	if err := verifyIndex(data, t0.Add(time.Minute), time.Hour); err != nil {
		t.Fatalf("the index the command wrote is refused: %v", err)
	}
	if code := run([]string{"check-yanked", yf}, &out, &errb); code != 0 {
		t.Fatalf("check-yanked = %d", code)
	}
}
