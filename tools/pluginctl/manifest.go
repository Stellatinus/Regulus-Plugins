package main

// The rules of plugin.yaml, written here ONLY because Regulus has no command
// that checks a manifest yet. They copy internal/plugin/manifest.go of Regulus
// (ParseManifest, Validate, ParseRange) as it is today: the real fix is a
// `regulus plugin check` command in Regulus that CI runs, so the rules live in
// one place. Until then, change them here only when Regulus changes them.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxManifest = 64 << 10

type manifest struct {
	ID          string       `yaml:"id"`
	Version     string       `yaml:"version"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	API         uint32       `yaml:"api"`
	Requires    string       `yaml:"requires"`
	Components  components   `yaml:"components"`
	Permissions []permission `yaml:"permissions"`
	Provides    []string     `yaml:"provides"`
}

type components struct {
	Agent *program  `yaml:"agent"`
	Hub   *program  `yaml:"hub"`
	UI    *uiModule `yaml:"ui"`
}

type program struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type uiModule struct {
	Module string `yaml:"module"`
}

type permission struct {
	ID          string `yaml:"id" json:"id"`
	Description string `yaml:"description" json:"description"`
}

var (
	idRE   = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
	permRE = regexp.MustCompile(`^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+$`)
)

// proposedKeys are keys of the store design that Regulus does not accept yet.
var proposedKeys = []string{"kind", "audience", "needs"}

func parseManifest(r io.Reader) (manifest, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxManifest+1))
	if err != nil {
		return manifest{}, err
	}
	if len(data) > maxManifest {
		return manifest{}, fmt.Errorf("plugin.yaml is larger than %d KiB", maxManifest>>10)
	}
	var m manifest
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		if errors.Is(err, io.EOF) {
			return manifest{}, errors.New("plugin.yaml is empty")
		}
		msg := err.Error()
		for _, k := range proposedKeys {
			if strings.Contains(msg, "field "+k+" not found") {
				msg += fmt.Sprintf(" (%q is in the design of the store but Regulus does not accept it yet: keep it as a comment)", k)
			}
		}
		return manifest{}, fmt.Errorf("plugin.yaml: %s", msg)
	}
	if err := m.validate(); err != nil {
		return manifest{}, err
	}
	return m, nil
}

func (m manifest) validate() error {
	if len(m.ID) < 2 || len(m.ID) > 40 || !idRE.MatchString(m.ID) {
		return fmt.Errorf("id %q must be 2 to 40 lower-case letters, digits and single hyphens, starting with a letter", m.ID)
	}
	if _, err := parseVersion(m.Version); err != nil || strings.HasPrefix(m.Version, "v") {
		return fmt.Errorf("version %q must be a SemVer version without a leading v, such as 0.1.0", m.Version)
	}
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("name must not be empty")
	}
	if m.API == 0 {
		return errors.New("api must name the plugin API version the plugin speaks (1)")
	}
	if err := rangeOK(m.Requires); err != nil {
		return fmt.Errorf("requires: %w", err)
	}
	c := m.Components
	if c.Agent == nil && c.Hub == nil && c.UI == nil {
		return errors.New("components must name at least one of agent, hub and ui")
	}
	if c.Agent != nil {
		if err := checkRelative(c.Agent.Command); err != nil {
			return fmt.Errorf("components.agent.command: %w", err)
		}
	}
	if c.Hub != nil {
		if err := checkRelative(c.Hub.Command); err != nil {
			return fmt.Errorf("components.hub.command: %w", err)
		}
	}
	if c.UI != nil {
		if err := checkRelative(c.UI.Module); err != nil {
			return fmt.Errorf("components.ui.module: %w", err)
		}
		if !strings.HasSuffix(c.UI.Module, ".js") {
			return errors.New("components.ui.module must be a .js module")
		}
	}
	seen := map[string]bool{}
	for _, p := range m.Permissions {
		if !permRE.MatchString(p.ID) || !strings.HasPrefix(p.ID, m.ID+".") {
			return fmt.Errorf("permission %q must start with the plugin id and a dot (%s.something)", p.ID, m.ID)
		}
		if seen[p.ID] {
			return fmt.Errorf("permission %q is listed twice", p.ID)
		}
		seen[p.ID] = true
		if strings.TrimSpace(p.Description) == "" {
			return fmt.Errorf("permission %q needs a description: it is what the administrator reads before allowing it", p.ID)
		}
	}
	return nil
}

func checkRelative(p string) error {
	switch {
	case p == "":
		return errors.New("must not be empty")
	case strings.HasPrefix(p, "/"):
		return fmt.Errorf("%q must be relative to the plugin directory", p)
	case strings.ContainsRune(p, '\\') || strings.ContainsRune(p, 0):
		return fmt.Errorf("%q is not a plain path", p)
	}
	clean := path.Clean(p)
	if clean != p || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%q must be a clean path inside the plugin directory", p)
	}
	return nil
}

// checkPlugin checks one plugin directory and returns every problem found.
// release is the version a release is being made for ("" for a plain CI run).
func checkPlugin(dir, release string) []string {
	var problems []string
	add := func(format string, a ...any) {
		problems = append(problems, fmt.Sprintf("%s: %s", dir, fmt.Sprintf(format, a...)))
	}

	f, err := os.Open(filepath.Join(dir, "plugin.yaml"))
	if err != nil {
		add("no plugin.yaml")
		return problems
	}
	m, err := parseManifest(f)
	f.Close()
	if err != nil {
		add("%v", err)
		return problems
	}
	if m.ID != filepath.Base(filepath.Clean(dir)) {
		add("the directory is not named after the id %q", m.ID)
	}
	for _, name := range []string{"README.md", "CHANGELOG.md"} {
		if st, err := os.Stat(filepath.Join(dir, name)); err != nil || !st.Mode().IsRegular() {
			add("%s is missing", name)
		}
	}
	if release != "" {
		if m.Version != release {
			add("plugin.yaml says version %s, the release is %s", m.Version, release)
		} else if _, err := changelogSection(filepath.Join(dir, "CHANGELOG.md"), release); err != nil {
			add("%v", err)
		}
	}
	return problems
}

// changelogSection returns the text under the "## <version>" heading of a
// CHANGELOG.md, up to the next "## " heading. It fails when there is none or
// it is empty: a release without notes is refused.
func changelogSection(file, ver string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("CHANGELOG.md is missing")
	}
	head := regexp.MustCompile(`^## \[?` + regexp.QuoteMeta(ver) + `\]?(\s.*)?$`)
	var out []string
	in := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case head.MatchString(line):
			in = true
		case in && strings.HasPrefix(line, "## "):
			in = false
		case in:
			out = append(out, line)
		}
	}
	text := strings.TrimSpace(strings.Join(out, "\n"))
	if text == "" {
		return "", fmt.Errorf("CHANGELOG.md has no notes under a \"## %s\" heading", ver)
	}
	return text, nil
}
