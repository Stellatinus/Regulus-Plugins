package main

// The plugin index: index.json in the format of "The index" in
// docs/plugins/store.md of Regulus. It is built from the plugin releases of
// this repository and signed by publish-index.yml.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const indexFormat = 1

type index struct {
	Format   int           `json:"format"`
	Source   string        `json:"source"`
	IssuedAt string        `json:"issued_at"`
	Expires  string        `json:"expires"`
	Plugins  []indexPlugin `json:"plugins"`
}

type indexPlugin struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Kind        string         `json:"kind,omitempty"`
	Audience    string         `json:"audience"`
	Homepage    string         `json:"homepage"`
	Versions    []indexVersion `json:"versions"`
}

type indexVersion struct {
	Version     string       `json:"version"`
	Channel     string       `json:"channel"`
	ReleasedAt  string       `json:"released_at"`
	API         uint32       `json:"api"`
	Requires    string       `json:"requires"`
	Components  []string     `json:"components"`
	Provides    []string     `json:"provides"`
	Needs       []need       `json:"needs"`
	Permissions []permission `json:"permissions"`
	Artifacts   []artifact   `json:"artifacts"`
	Yanked      bool         `json:"yanked"`
	YankReason  string       `json:"yank_reason,omitempty"`
}

type need struct {
	Capability string `yaml:"capability" json:"capability"`
	Optional   bool   `yaml:"optional" json:"optional,omitempty"`
}

type artifact struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// release is what `gh api repos/<repo>/releases` returns, as far as the index
// needs it.
type release struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Published  string  `json:"published_at"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	URL  string `json:"browser_download_url"`
}

// futureManifest reads the keys the store design adds (kind, audience,
// needs) without refusing the ones it does not know: the index must not need a
// change in this tool the day Regulus starts accepting them. plugin.yaml was
// already checked strictly by `check` before a release is made.
type futureManifest struct {
	manifest `yaml:",inline"`
	Kind     string `yaml:"kind"`
	Audience string `yaml:"audience"`
	Needs    []need `yaml:"needs"`
}

var tagRE = regexp.MustCompile(`^plugin/([a-z][a-z0-9]*(-[a-z0-9]+)*)/v(.+)$`)

// assetDir is where the workflow downloaded the small files of a release: the
// tag with its slashes replaced.
func assetDir(root, tag string) string {
	return filepath.Join(root, strings.ReplaceAll(tag, "/", "__"))
}

type buildOptions struct {
	releases []release
	assets   string // directory of downloaded manifests and checksums
	yanked   map[string]string
	repo     string
	now      time.Time
	validFor time.Duration
}

func buildIndex(o buildOptions) (index, error) {
	type key struct{ id, ver string }
	byPlugin := map[string][]indexVersion{}
	info := map[string]indexPlugin{}
	best := map[string]version{}
	seen := map[key]bool{}
	var rels []release
	for _, r := range o.releases {
		if r.Draft || !tagRE.MatchString(r.Tag) {
			continue
		}
		rels = append(rels, r)
	}
	sort.Slice(rels, func(i, j int) bool { return rels[i].Tag < rels[j].Tag })

	for _, r := range rels {
		m := tagRE.FindStringSubmatch(r.Tag)
		id, verText := m[1], m[3]
		v, err := parseVersion(verText)
		if err != nil || strings.HasPrefix(verText, "v") || strings.Contains(verText, "+") {
			return index{}, fmt.Errorf("%s: %q is not a version", r.Tag, verText)
		}
		if (len(v.pre) > 0) != r.Prerelease {
			return index{}, fmt.Errorf("%s: the release is marked prerelease=%v but the version is %s", r.Tag, r.Prerelease, verText)
		}
		if seen[key{id, verText}] {
			return index{}, fmt.Errorf("%s: two releases for one version", r.Tag)
		}
		seen[key{id, verText}] = true

		dir := assetDir(o.assets, r.Tag)
		raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%s_%s_plugin.yaml", id, verText)))
		if err != nil {
			return index{}, fmt.Errorf("%s: the release has no %s_%s_plugin.yaml: %v", r.Tag, id, verText, err)
		}
		var fm futureManifest
		if err := yaml.Unmarshal(raw, &fm); err != nil {
			return index{}, fmt.Errorf("%s: plugin.yaml: %v", r.Tag, err)
		}
		if fm.ID != id || fm.Version != verText {
			return index{}, fmt.Errorf("%s: plugin.yaml says %s %s", r.Tag, fm.ID, fm.Version)
		}

		var arts []artifact
		prefix := fmt.Sprintf("%s_%s_", id, verText)
		for _, a := range r.Assets {
			if !strings.HasPrefix(a.Name, prefix) || !strings.HasSuffix(a.Name, ".tar.gz") {
				continue
			}
			plat := strings.TrimSuffix(strings.TrimPrefix(a.Name, prefix), ".tar.gz")
			osName, arch, ok := strings.Cut(plat, "_")
			if !ok || osName == "" || arch == "" || strings.Contains(arch, "_") {
				return index{}, fmt.Errorf("%s: %s is not named <id>_<version>_<os>_<arch>.tar.gz", r.Tag, a.Name)
			}
			sum, err := readChecksum(filepath.Join(dir, a.Name+".sha256"), a.Name)
			if err != nil {
				return index{}, fmt.Errorf("%s: %v", r.Tag, err)
			}
			arts = append(arts, artifact{OS: osName, Arch: arch, URL: a.URL, SHA256: sum, Size: a.Size})
		}
		if len(arts) == 0 {
			return index{}, fmt.Errorf("%s: the release has no package", r.Tag)
		}
		sort.Slice(arts, func(i, j int) bool {
			return arts[i].OS+"/"+arts[i].Arch < arts[j].OS+"/"+arts[j].Arch
		})

		iv := indexVersion{
			Version: verText, Channel: v.channel(), ReleasedAt: r.Published,
			API: fm.API, Requires: fm.Requires,
			Components: componentNames(fm.Components),
			Provides:   nonNil(fm.Provides),
			Needs:      fm.Needs, Permissions: fm.Permissions, Artifacts: arts,
		}
		if iv.Needs == nil {
			iv.Needs = []need{}
		}
		if iv.Permissions == nil {
			iv.Permissions = []permission{}
		}
		if reason, ok := o.yanked[id+" "+verText]; ok {
			iv.Yanked, iv.YankReason = true, reason
			delete(o.yanked, id+" "+verText)
		}
		byPlugin[id] = append(byPlugin[id], iv)

		// The plugin's own fields come from its newest version, any channel.
		if cur, ok := best[id]; !ok || v.compare(cur) > 0 {
			best[id] = v
			aud := fm.Audience
			if aud == "" {
				aud = "admin" // the default: never shown to ordinary users by mistake
			}
			info[id] = indexPlugin{
				ID: id, Name: fm.Name, Description: fm.Description, Kind: fm.Kind, Audience: aud,
				Homepage: fmt.Sprintf("https://github.com/%s/tree/main/plugins/%s", o.repo, id),
			}
		}
	}
	if len(o.yanked) > 0 {
		var left []string
		for k := range o.yanked {
			left = append(left, k)
		}
		sort.Strings(left)
		return index{}, fmt.Errorf("yanked.txt names a release that does not exist: %s", strings.Join(left, ", "))
	}

	out := index{
		Format: indexFormat, Source: "regulus-plugins",
		IssuedAt: o.now.UTC().Format(time.RFC3339),
		Expires:  o.now.Add(o.validFor).UTC().Format(time.RFC3339),
		Plugins:  []indexPlugin{},
	}
	ids := make([]string, 0, len(byPlugin))
	for id := range byPlugin {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		vs := byPlugin[id]
		sort.Slice(vs, func(i, j int) bool {
			a, _ := parseVersion(vs[i].Version)
			b, _ := parseVersion(vs[j].Version)
			return a.compare(b) > 0 // newest first
		})
		p := info[id]
		p.Versions = vs
		out.Plugins = append(out.Plugins, p)
	}
	return out, nil
}

func componentNames(c components) []string {
	out := []string{}
	if c.Agent != nil {
		out = append(out, "agent")
	}
	if c.Hub != nil {
		out = append(out, "hub")
	}
	if c.UI != nil {
		out = append(out, "ui")
	}
	return out
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

var sumRE = regexp.MustCompile(`^[0-9a-f]{64}$`)

// readChecksum reads "<sha256>  <file name>" and checks the name.
func readChecksum(file, name string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("no checksum for %s: %v", name, err)
	}
	f := strings.Fields(string(data))
	if len(f) != 2 || !sumRE.MatchString(f[0]) || strings.TrimPrefix(f[1], "*") != name {
		return "", fmt.Errorf("%s.sha256 is not \"<sha256>  %s\"", name, name)
	}
	return f[0], nil
}

// readYanked reads yanked.txt: "<id> <version> [reason]" per line, "#" starts
// a comment. The key is "<id> <version>".
func readYanked(file string) (map[string]string, error) {
	out := map[string]string{}
	f, err := os.Open(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || !idRE.MatchString(fields[0]) {
			return nil, fmt.Errorf("%s:%d: want \"<id> <version> [reason]\"", file, n)
		}
		if _, err := parseVersion(fields[1]); err != nil || strings.HasPrefix(fields[1], "v") {
			return nil, fmt.Errorf("%s:%d: %q is not a version without a leading v", file, n, fields[1])
		}
		k := fields[0] + " " + fields[1]
		if _, dup := out[k]; dup {
			return nil, fmt.Errorf("%s:%d: %s is listed twice", file, n, k)
		}
		out[k] = strings.Join(fields[2:], " ")
	}
	return out, sc.Err()
}

// verifyIndex checks a built index before it is signed: it is what the
// workflow's signing job refuses to sign if it is wrong.
func verifyIndex(data []byte, now time.Time, maxAge time.Duration) error {
	var ix index
	if err := json.Unmarshal(data, &ix); err != nil {
		return fmt.Errorf("index.json: %v", err)
	}
	if ix.Format != indexFormat || ix.Source == "" {
		return fmt.Errorf("index.json: format %d, source %q", ix.Format, ix.Source)
	}
	issued, err1 := time.Parse(time.RFC3339, ix.IssuedAt)
	expires, err2 := time.Parse(time.RFC3339, ix.Expires)
	if err1 != nil || err2 != nil {
		return errors.New("index.json: issued_at or expires is not a time")
	}
	if issued.After(now.Add(5 * time.Minute)) {
		return errors.New("index.json: issued in the future")
	}
	if now.Sub(issued) > maxAge {
		return fmt.Errorf("index.json: issued %s ago, more than %s", now.Sub(issued).Round(time.Second), maxAge)
	}
	if !expires.After(now) || expires.Sub(issued) > 90*24*time.Hour {
		return errors.New("index.json: it has expired, or it is valid for more than 90 days")
	}
	for _, p := range ix.Plugins {
		if len(p.Versions) == 0 {
			return fmt.Errorf("index.json: %s has no version", p.ID)
		}
		for _, v := range p.Versions {
			if len(v.Artifacts) == 0 {
				return fmt.Errorf("index.json: %s %s has no package", p.ID, v.Version)
			}
			for _, a := range v.Artifacts {
				if !sumRE.MatchString(a.SHA256) || !strings.HasPrefix(a.URL, "https://") {
					return fmt.Errorf("index.json: %s %s has a bad artifact (%s)", p.ID, v.Version, a.URL)
				}
			}
		}
	}
	return nil
}
