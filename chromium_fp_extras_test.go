package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestFpExtrasSeed(t *testing.T) {
	seen := map[int64]bool{}
	for i := 0; i < 200; i++ {
		s := newNoiseSeed()
		if s < 1 || s >= 1<<40 {
			t.Fatalf("seed out of range: %d", s)
		}
		seen[s] = true
	}
	if len(seen) < 199 {
		t.Fatal("seeds repeat")
	}
	if seedShortID(0x1a2b3c4d5e) != "1a2b3c4d" || seedShortID(255) != "000000ff" {
		t.Fatalf("short id %s %s", seedShortID(0x1a2b3c4d5e), seedShortID(255))
	}
	st := testStore(t)
	fp := GenerateFingerprint("windows", "141")
	st.PutProfile(Profile{ID: "p1", Name: "x", Cookies: `[{"name":"a"}]`, Fingerprint: fp})
	before, _ := st.GetProfile("p1")
	seed, err := st.regenerateSeed("p1")
	if err != nil {
		t.Fatal(err)
	}
	after, _ := st.GetProfile("p1")
	if seed == before.Fingerprint.NoiseSeed || after.Fingerprint.NoiseSeed != seed {
		t.Fatal("seed not replaced")
	}
	if after.Cookies != before.Cookies || after.Fingerprint.UserAgent != before.Fingerprint.UserAgent || after.Fingerprint.WebGLRenderer != before.Fingerprint.WebGLRenderer {
		t.Fatal("regenerating the seed changed something else")
	}
	if effectiveSeed("p1", seed, false) == effectiveSeed("p1", before.Fingerprint.NoiseSeed, false) {
		t.Fatal("effective seed unchanged")
	}
	if _, err := st.regenerateSeed("missing"); err == nil {
		t.Fatal("missing profile")
	}
}

func TestFpExtrasScriptConfig(t *testing.T) {
	fp := GenerateFingerprint("windows", "141")
	fp.MediaIDAlias = true
	js := pageScript(fp, nil)
	if !strings.Contains(js, `"mida":true`) || !strings.Contains(js, "C.mida") {
		t.Fatal("alias flag not passed to the page script")
	}
	fp.MediaIDAlias = false
	if !strings.Contains(pageScript(fp, nil), `"mida":false`) {
		t.Fatal("alias flag default")
	}
	fp.WebGPU = "off"
	if !strings.Contains(fmt.Sprint(launchFlags(fp)), "--disable-blink-features=WebGPU") {
		t.Fatal("webgpu flag")
	}
}
