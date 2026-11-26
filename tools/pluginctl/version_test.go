package main

import "testing"

func TestVersionOrderAndChannel(t *testing.T) {
	order := []string{"0.1.0-alpha.1", "0.1.0-alpha.2", "0.1.0-alpha.10", "0.1.0-beta.1", "0.1.0-rc.1", "0.1.0", "0.1.1", "0.2.0-alpha.1", "1.0.0"}
	for i := 0; i < len(order)-1; i++ {
		a, err := parseVersion(order[i])
		if err != nil {
			t.Fatal(err)
		}
		b, _ := parseVersion(order[i+1])
		if a.compare(b) >= 0 || b.compare(a) <= 0 {
			t.Errorf("%s should come before %s", order[i], order[i+1])
		}
	}
	for ver, want := range map[string]string{
		"1.0.0": "stable", "1.0.0-beta.2": "beta", "1.0.0-rc.1": "beta",
		"1.0.0-alpha.1": "alpha", "1.0.0-dev.1": "alpha", "1.0.0-whatever": "alpha",
	} {
		v, err := parseVersion(ver)
		if err != nil {
			t.Fatal(err)
		}
		if got := v.channel(); got != want {
			t.Errorf("channel(%s) = %s, want %s", ver, got, want)
		}
	}
	for _, bad := range []string{"1.0", "01.0.0", "1.0.0-alpha.01", "x", ""} {
		if _, err := parseVersion(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
