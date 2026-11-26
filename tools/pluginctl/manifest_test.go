package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const goodManifest = `id: hello-world
version: 0.1.0-alpha.1
name: Hello world
description: Does nothing useful.
api: 1
requires: ">=0.4.0-alpha.1"
components:
  agent:
    command: bin/hello-world
  hub:
    command: bin/hello-world
permissions:
  - id: hello-world.greet
    description: Say hello.
`

func TestManifestRules(t *testing.T) {
	cases := []struct {
		name, yaml, wantErr string
	}{
		{"good", goodManifest, ""},
		{"empty", "", "empty"},
		{"unknown key", goodManifest + "colour: red\n", "field colour not found"},
		{"proposed key kind", goodManifest + "kind: integration\n", "does not accept it yet"},
		{"proposed key audience", goodManifest + "audience: admin\n", "does not accept it yet"},
		{"proposed key needs", goodManifest + "needs: []\n", "does not accept it yet"},
		{"bad id", strings.Replace(goodManifest, "id: hello-world", "id: Hello", 1), "id "},
		{"id too short", strings.Replace(goodManifest, "id: hello-world", "id: a", 1), "id "},
		{"version with v", strings.Replace(goodManifest, "version: 0.1.0-alpha.1", "version: v0.1.0", 1), "version"},
		{"version not semver", strings.Replace(goodManifest, "version: 0.1.0-alpha.1", "version: 1.0", 1), "version"},
		{"no name", strings.Replace(goodManifest, "name: Hello world", "name: ''", 1), "name must not be empty"},
		{"no api", strings.Replace(goodManifest, "api: 1", "api: 0", 1), "api must"},
		{"range without operator", strings.Replace(goodManifest, `">=0.4.0-alpha.1"`, `"0.4.0"`, 1), "needs an operator"},
		{"empty range is any", strings.Replace(goodManifest, `requires: ">=0.4.0-alpha.1"`, `requires: ""`, 1), ""},
		{"no component", "id: hello-world\nversion: 0.1.0\nname: x\napi: 1\n", "at least one"},
		{"absolute command", strings.Replace(goodManifest, "command: bin/hello-world\n  hub", "command: /bin/sh\n  hub", 1), "relative"},
		{"command leaves the directory", strings.Replace(goodManifest, "command: bin/hello-world\n  hub", "command: ../x\n  hub", 1), "clean path"},
		{"unclean command", strings.Replace(goodManifest, "command: bin/hello-world\n  hub", "command: ./bin/x\n  hub", 1), "clean path"},
		{"ui module not js", "id: ui-only\nversion: 0.1.0\nname: x\napi: 1\ncomponents:\n  ui:\n    module: ui/w.txt\n", ".js"},
		{"permission without the id", strings.Replace(goodManifest, "hello-world.greet", "other.greet", 1), "must start with the plugin id"},
		{"permission without description", strings.Replace(goodManifest, "Say hello.", "''", 1), "needs a description"},
		{"permission twice", goodManifest + "  - id: hello-world.greet\n    description: again\n", "listed twice"},
		{"provides is accepted", goodManifest + "provides:\n  - hello.greet@1\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseManifest(strings.NewReader(c.yaml))
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)):
				t.Fatalf("error = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}

func writePlugin(t *testing.T, root, id, manifestYAML string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if manifestYAML != "" {
		files["plugin.yaml"] = manifestYAML
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCheckPlugin(t *testing.T) {
	root := t.TempDir()
	changelog := "# Changelog\n\n## 0.1.0-alpha.1\n\nFirst alpha.\n\n## 0.0.9\n\nOlder.\n"
	ok := writePlugin(t, root, "hello-world", goodManifest, map[string]string{"README.md": "x", "CHANGELOG.md": changelog})
	if p := checkPlugin(ok, ""); len(p) != 0 {
		t.Fatalf("a good plugin has problems: %v", p)
	}
	if p := checkPlugin(ok, "0.1.0-alpha.1"); len(p) != 0 {
		t.Fatalf("a good release has problems: %v", p)
	}

	// A release for another version than the manifest's, or without notes.
	if p := checkPlugin(ok, "0.1.0-alpha.2"); len(p) != 1 || !strings.Contains(p[0], "plugin.yaml says version 0.1.0-alpha.1") {
		t.Fatalf("version mismatch: %v", p)
	}
	noNotes := writePlugin(t, root, "no-notes", strings.ReplaceAll(goodManifest, "hello-world", "no-notes"), map[string]string{"README.md": "x", "CHANGELOG.md": "## 0.0.1\n\nx\n"})
	if p := checkPlugin(noNotes, "0.1.0-alpha.1"); len(p) != 1 || !strings.Contains(p[0], "no notes") {
		t.Fatalf("a release without notes: %v", p)
	}

	// Missing files and a directory not named after the id.
	bare := writePlugin(t, root, "bare", strings.ReplaceAll(goodManifest, "hello-world", "bare"), map[string]string{})
	if p := checkPlugin(bare, ""); len(p) != 2 {
		t.Fatalf("README.md and CHANGELOG.md are missing: %v", p)
	}
	wrong := writePlugin(t, root, "other-dir", goodManifest, map[string]string{"README.md": "x", "CHANGELOG.md": "x"})
	if p := checkPlugin(wrong, ""); len(p) != 1 || !strings.Contains(p[0], "not named after the id") {
		t.Fatalf("directory name: %v", p)
	}
	if p := checkPlugin(filepath.Join(root, "missing"), ""); len(p) != 1 || !strings.Contains(p[0], "no plugin.yaml") {
		t.Fatalf("no plugin: %v", p)
	}
}

func TestChangelogSection(t *testing.T) {
	file := filepath.Join(t.TempDir(), "CHANGELOG.md")
	body := "# Changelog\n\n## [1.0.0] - 2026-12-01\n\nBig.\n\nMore.\n\n## 1.0.0-beta.1\n\nBeta.\n"
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := changelogSection(file, "1.0.0"); err != nil || got != "Big.\n\nMore." {
		t.Fatalf("1.0.0 = %q, %v", got, err)
	}
	if got, err := changelogSection(file, "1.0.0-beta.1"); err != nil || got != "Beta." {
		t.Fatalf("beta = %q, %v", got, err)
	}
	// A version that only starts with the same digits is not the same version.
	if _, err := changelogSection(file, "1.0"); err == nil {
		t.Fatal("1.0 matched 1.0.0")
	}
	if _, err := changelogSection(file, "2.0.0"); err == nil {
		t.Fatal("an absent version has notes")
	}
}
