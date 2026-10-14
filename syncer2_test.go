package main

import (
	"math/rand"
	"testing"
	"time"
)

func TestNormalizeURL(t *testing.T) {
	ok := map[string]string{
		"example.com":                    "https://example.com",
		"  example.com/a?b=1#c ":         "https://example.com/a?b=1#c",
		"http://example.com":             "http://example.com",
		"HTTPS://Example.com/x":          "https://Example.com/x",
		"//cdn.example.com/x":            "https://cdn.example.com/x",
		"localhost:3000/p":               "https://localhost:3000/p",
		"127.0.0.1:8080":                 "https://127.0.0.1:8080",
		"www.baidu.com/s?wd=你好":          "https://www.baidu.com/s?wd=你好",
		"about:blank":                    "about:blank",
		"chrome://newtab/":               "chrome://newtab/",
		"sub.domain.co.uk:8443/a/b.html": "https://sub.domain.co.uk:8443/a/b.html",
	}
	for in, want := range ok {
		got, err := normalizeURL(in)
		if err != nil || got != want {
			t.Errorf("normalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "hello", "exa mple.com", "ftp://x.com", "javascript:alert(1)", "https://", "http://.com", "a..b.com"} {
		if got, err := normalizeURL(bad); err == nil {
			t.Errorf("normalizeURL(%q) = %q, want error", bad, got)
		}
	}
}

func TestIsBlankURL(t *testing.T) {
	for _, u := range []string{"", "about:blank", "chrome://newtab/", "chrome://new-tab-page/", "chrome-search://local-ntp/local-ntp.html"} {
		if !isBlankURL(u) {
			t.Errorf("%q should be blank", u)
		}
	}
	for _, u := range []string{"https://example.com", "chrome://settings/", "data:text/html,x"} {
		if isBlankURL(u) {
			t.Errorf("%q should not be blank", u)
		}
	}
}

func TestSampleDelay(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	o := SyncOptions{DelayMin: 50, DelayMax: 120}
	seen := map[time.Duration]bool{}
	for i := 0; i < 5000; i++ {
		d := sampleDelay(o, r)
		if d < 50*time.Millisecond || d > 120*time.Millisecond {
			t.Fatalf("delay %v out of range", d)
		}
		seen[d] = true
	}
	if !seen[50*time.Millisecond] || !seen[120*time.Millisecond] || len(seen) < 60 {
		t.Fatalf("distribution does not cover the range: %d distinct", len(seen))
	}
	if d := sampleDelay(SyncOptions{}, r); d != 0 {
		t.Fatalf("zero range gave %v", d)
	}
	if d := sampleDelay(SyncOptions{DelayMin: 80, DelayMax: 80}, r); d != 80*time.Millisecond {
		t.Fatalf("fixed delay gave %v", d)
	}
	// swapped and out-of-range values are normalised
	n := SyncOptions{DelayMin: 900, DelayMax: 100}.norm()
	if n.DelayMin != 100 || n.DelayMax != 900 {
		t.Fatalf("swap: %+v", n)
	}
	n = SyncOptions{DelayMin: -5, DelayMax: 999999}.norm()
	if n.DelayMin != 0 || n.DelayMax != syncMaxDelay {
		t.Fatalf("clamp: %+v", n)
	}
}

func TestSyncOptionsAllows(t *testing.T) {
	o := defaultSyncOptions()
	for _, ty := range []string{"click", "scroll", "text", "select", "key", "nav", "tabnew", "tabswitch", "tabclose"} {
		if !o.allows(ty) {
			t.Errorf("default should allow %s", ty)
		}
	}
	o = SyncOptions{}
	for _, ty := range []string{"click", "scroll", "text", "select", "key", "tabnew", "tabswitch", "tabclose"} {
		if o.allows(ty) {
			t.Errorf("all-off should block %s", ty)
		}
	}
	if !o.allows("nav") {
		t.Error("navigation is always followed")
	}
}

func TestSyncerPrefsPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	t.Setenv("HOME", dir)
	st := NewStore()
	p := loadSyncerPrefs(st)
	if !p.Click || !p.Input || !p.Key || !p.Scroll || !p.Tabs || p.DelayMax != 0 || p.Followers == nil {
		t.Fatalf("defaults %+v", p)
	}
	saved := saveSyncerPrefs(st, SyncerPrefs{SyncOptions: SyncOptions{Click: true, Key: true, DelayMin: 300, DelayMax: 40}, Master: "m", Followers: []string{"a", "m", "b", "a", ""}, NewTab: true})
	if saved.DelayMin != 40 || saved.DelayMax != 300 || len(saved.Followers) != 2 {
		t.Fatalf("normalised %+v", saved)
	}
	st2 := NewStore() // reload from disk
	got := loadSyncerPrefs(st2)
	if got.Master != "m" || len(got.Followers) != 2 || got.Followers[0] != "a" || got.Followers[1] != "b" || !got.NewTab ||
		!got.Click || got.Input || !got.Key || got.Scroll || got.Tabs || got.DelayMin != 40 || got.DelayMax != 300 {
		t.Fatalf("reloaded %+v", got)
	}
}
