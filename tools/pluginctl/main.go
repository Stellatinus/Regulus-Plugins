// Command pluginctl is the tooling of the regulus-plugins repository: it
// checks plugin directories, prints the release notes of a version, writes the
// plugin index and checks it before it is signed. It is not part of any
// plugin, and it does not run on a server.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `usage:
  pluginctl check [-release VERSION] DIR...      check plugin directories (manifest, README.md, CHANGELOG.md)
  pluginctl notes DIR VERSION                    print the CHANGELOG.md notes of a version
  pluginctl index -releases FILE -assets DIR -repo OWNER/NAME [-yanked FILE] [-now TIME] [-valid-days N] -o FILE
  pluginctl verify-index [-max-age DURATION] FILE
  pluginctl check-yanked FILE`

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	switch args[0] {
	case "check":
		fs := flag.NewFlagSet("check", flag.ContinueOnError)
		fs.SetOutput(stderr)
		rel := fs.String("release", "", "the version a release is being made for")
		if fs.Parse(args[1:]) != nil || fs.NArg() == 0 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		bad := 0
		for _, dir := range fs.Args() {
			if problems := checkPlugin(dir, *rel); len(problems) > 0 {
				for _, p := range problems {
					fmt.Fprintln(stderr, "error:", p)
				}
				bad++
			} else {
				fmt.Fprintln(stdout, "ok:", dir)
			}
		}
		if bad > 0 {
			return 1
		}
		return 0
	case "notes":
		if len(args) != 3 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		text, err := changelogSection(args[1]+"/CHANGELOG.md", args[2])
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, text)
		return 0
	case "index":
		return runIndex(args[1:], stdout, stderr)
	case "verify-index":
		fs := flag.NewFlagSet("verify-index", flag.ContinueOnError)
		fs.SetOutput(stderr)
		maxAge := fs.Duration("max-age", time.Hour, "the oldest issued_at accepted")
		if fs.Parse(args[1:]) != nil || fs.NArg() != 1 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		data, err := os.ReadFile(fs.Arg(0))
		if err == nil {
			err = verifyIndex(data, time.Now(), *maxAge)
		}
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, "ok: the index can be signed")
		return 0
	case "check-yanked":
		if len(args) != 2 {
			fmt.Fprintln(stderr, usage)
			return 2
		}
		if _, err := readYanked(args[1]); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, "ok:", args[1])
		return 0
	}
	fmt.Fprintln(stderr, usage)
	return 2
}

func runIndex(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(stderr)
	releases := fs.String("releases", "", "JSON of the releases (gh api repos/OWNER/NAME/releases --paginate --slurp, flattened)")
	assets := fs.String("assets", "", "directory of the downloaded manifests and checksums")
	repo := fs.String("repo", "", "OWNER/NAME")
	yanked := fs.String("yanked", "", "yanked.txt")
	nowText := fs.String("now", "", "the time to issue the index at (RFC 3339; default: now)")
	days := fs.Int("valid-days", 21, "days the index is valid")
	outFile := fs.String("o", "", "file to write")
	if fs.Parse(args) != nil || *releases == "" || *assets == "" || *repo == "" || *outFile == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	fail := func(err error) int { fmt.Fprintln(stderr, "error:", err); return 1 }

	raw, err := os.ReadFile(*releases)
	if err != nil {
		return fail(err)
	}
	var rels []release
	if err := json.Unmarshal(raw, &rels); err != nil {
		return fail(fmt.Errorf("%s: %v", *releases, err))
	}
	y := map[string]string{}
	if *yanked != "" {
		if y, err = readYanked(*yanked); err != nil {
			return fail(err)
		}
	}
	now := time.Now()
	if *nowText != "" {
		if now, err = time.Parse(time.RFC3339, *nowText); err != nil {
			return fail(err)
		}
	}
	ix, err := buildIndex(buildOptions{releases: rels, assets: *assets, yanked: y, repo: *repo, now: now, validFor: time.Duration(*days) * 24 * time.Hour})
	if err != nil {
		return fail(err)
	}
	data, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return fail(err)
	}
	if err := os.WriteFile(*outFile, append(data, '\n'), 0o644); err != nil {
		return fail(err)
	}
	fmt.Fprintf(stdout, "wrote %s: %d plugins\n", *outFile, len(ix.Plugins))
	return 0
}
