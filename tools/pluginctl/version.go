package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// version is a SemVer version. It follows Regulus' internal/update.ParseVersion
// (build metadata is accepted and ignored when comparing).
type version struct {
	major, minor, patch uint64
	pre                 []string
}

var versionRE = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z.-]+)?$`)

func parseVersion(s string) (version, error) {
	m := versionRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return version{}, fmt.Errorf("%q is not a version like 1.4.0 or 1.5.0-beta.1", s)
	}
	var v version
	for i, dst := range []*uint64{&v.major, &v.minor, &v.patch} {
		n, err := strconv.ParseUint(m[i+1], 10, 64)
		if err != nil {
			return version{}, fmt.Errorf("%q: number too large", s)
		}
		*dst = n
	}
	if m[4] != "" {
		v.pre = strings.Split(m[4], ".")
		for _, id := range v.pre {
			if isNumeric(id) && len(id) > 1 && id[0] == '0' {
				return version{}, fmt.Errorf("%q: numeric pre-release identifier %q has a leading zero", s, id)
			}
		}
	}
	return v, nil
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// compare orders versions as SemVer does: a pre-release comes before its
// release, numeric identifiers compare as numbers and sort before the others.
func (v version) compare(o version) int {
	for _, p := range [][2]uint64{{v.major, o.major}, {v.minor, o.minor}, {v.patch, o.patch}} {
		if p[0] != p[1] {
			if p[0] < p[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(v.pre) == 0 && len(o.pre) == 0:
		return 0
	case len(v.pre) == 0:
		return 1
	case len(o.pre) == 0:
		return -1
	}
	for i := 0; i < len(v.pre) && i < len(o.pre); i++ {
		a, b := v.pre[i], o.pre[i]
		if a == b {
			continue
		}
		an, bn := isNumeric(a), isNumeric(b)
		switch {
		case an && bn:
			x, _ := strconv.ParseUint(a, 10, 64)
			y, _ := strconv.ParseUint(b, 10, 64)
			if x < y {
				return -1
			}
			return 1
		case an:
			return -1
		case bn:
			return 1
		case a < b:
			return -1
		default:
			return 1
		}
	}
	switch {
	case len(v.pre) < len(o.pre):
		return -1
	case len(v.pre) > len(o.pre):
		return 1
	}
	return 0
}

// channel is the channel a version belongs to, with the rule of Regulus
// (docs/decisions.md, "Releases and channels"): no pre-release is stable;
// beta and rc are beta; anything else is alpha, so an unknown kind never
// reaches a more cautious channel.
func (v version) channel() string {
	if len(v.pre) == 0 {
		return "stable"
	}
	switch v.pre[0] {
	case "beta", "rc":
		return "beta"
	}
	return "alpha"
}

// rangeOK checks a `requires` range with the rules of Regulus'
// plugin.ParseRange: constraints separated by spaces or commas, each an
// operator (>=, <=, >, <, =) followed by a version. Empty means any.
func rangeOK(s string) error {
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == ',' }) {
		op := ""
		for _, o := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(f, o) {
				op = o
				break
			}
		}
		if op == "" {
			return fmt.Errorf("%q needs an operator: >=, >, <=, < or =", f)
		}
		if _, err := parseVersion(strings.TrimPrefix(f, op)); err != nil {
			return fmt.Errorf("%q: %w", f, err)
		}
	}
	return nil
}
