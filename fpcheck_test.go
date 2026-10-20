package main

import (
	"strings"
	"testing"
)

func TestResolveFpCheck(t *testing.T) {
	ui, api := LaunchOpts{UI: true}, LaunchOpts{}
	on := Settings{FpCheckOn: true, FpCheckSite: "pixelscan"}
	cases := []struct {
		name string
		st   Settings
		p    Profile
		o    LaunchOpts
		want string // "" = none, "builtin", or URL
		once bool
	}{
		{"global off by default", Settings{}, Profile{}, ui, "", false},
		{"global on", on, Profile{}, ui, "https://pixelscan.net/", false},
		{"global on, default site", Settings{FpCheckOn: true}, Profile{}, ui, "https://www.browserscan.net/", false},
		{"API / RPA launch never", on, Profile{}, api, "", false},
		{"API launch even when profile forces on", on, Profile{FpCheck: "on"}, api, "", false},
		{"profile off beats global on", on, Profile{FpCheck: "off"}, ui, "", false},
		{"profile on beats global off", Settings{}, Profile{FpCheck: "on"}, ui, "https://www.browserscan.net/", false},
		{"profile site", on, Profile{FpCheckSite: "creepjs"}, ui, "https://abrahamjuliot.github.io/creepjs/", false},
		{"profile custom", on, Profile{FpCheckSite: "custom", FpCheckURL: "example.com/fp"}, ui, "https://example.com/fp", false},
		{"profile custom invalid", on, Profile{FpCheckSite: "custom", FpCheckURL: "javascript:alert(1)"}, ui, "", false},
		{"global custom", Settings{FpCheckOn: true, FpCheckSite: "custom", FpCheckURL: "http://127.0.0.1:9/x"}, Profile{}, ui, "http://127.0.0.1:9/x", false},
		{"global custom empty", Settings{FpCheckOn: true, FpCheckSite: "custom"}, Profile{}, ui, "", false},
		{"unknown key falls back", Settings{FpCheckOn: true, FpCheckSite: "gone"}, Profile{}, ui, "https://www.browserscan.net/", false},
		{"builtin", Settings{FpCheckOn: true, FpCheckSite: "builtin"}, Profile{}, ui, "builtin", false},
		{"once: first launch", Settings{FpCheckOn: true, FpCheckOnce: true}, Profile{}, ui, "https://www.browserscan.net/", true},
		{"once: already opened", Settings{FpCheckOn: true, FpCheckOnce: true}, Profile{FpCheckAt: 1}, ui, "", false},
		{"profile every beats global once", Settings{FpCheckOn: true, FpCheckOnce: true}, Profile{FpCheckAt: 1, FpCheckFreq: "every"}, ui, "https://www.browserscan.net/", false},
		{"profile once beats global every", on, Profile{FpCheckAt: 1, FpCheckFreq: "once"}, ui, "", false},
		{"profile once, never opened", on, Profile{FpCheckFreq: "once"}, ui, "https://pixelscan.net/", true},
	}
	for _, c := range cases {
		got := resolveFpCheck(c.st, c.p, c.o)
		s := got.URL
		if got.Builtin {
			s = "builtin"
		}
		if s != c.want || (s != "" && got.Once != c.once) {
			t.Errorf("%s: got %+v, want %q once=%v", c.name, got, c.want, c.once)
		}
	}
}

func TestWithCheckURL(t *testing.T) {
	for _, c := range []struct {
		own  []string
		want string
	}{
		{nil, "C"},
		{[]string{""}, "C"},
		{[]string{"about:blank"}, "C"},
		{[]string{"https://a/"}, "https://a/,C"},
		{[]string{"https://a/", "", "https://b/"}, "https://a/,,https://b/,C"},
	} {
		if got := strings.Join(withCheckURL(c.own, "C"), ","); got != c.want {
			t.Errorf("%v: got %q want %q", c.own, got, c.want)
		}
	}
	own := []string{"https://a/"}
	_ = withCheckURL(own, "C")
	if len(own) != 1 {
		t.Fatal("own list modified")
	}
	if got := withCheckURL(own, ""); len(got) != 1 {
		t.Fatal("empty check must not change the list")
	}
}

func TestFpCheckSitesListed(t *testing.T) {
	keys := map[string]bool{}
	for _, s := range fpCheckSites {
		keys[s.Key] = true
		if s.Key != fpCheckBuiltin && !safeSiteURL(s.URL) {
			t.Errorf("%s: bad url %q", s.Key, s.URL)
		}
	}
	for _, k := range []string{"browserscan", "pixelscan", "creepjs", "browserleaks", "iphey", "whoer", "builtin"} {
		if !keys[k] {
			t.Errorf("missing %s", k)
		}
	}
}
