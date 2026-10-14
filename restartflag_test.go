package main

import (
	"testing"
)

func rfStore(t *testing.T) *Store {
	d := t.TempDir()
	return &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: d + "/profiles"}}
}

func TestRestartFlagLaunchRelevantOnly(t *testing.T) {
	s := rfStore(t)
	px := s.PutProxy(Proxy{Name: "px", Type: "socks5", Host: "1.2.3.4", Port: 1080})
	p := s.PutProfile(Profile{Name: "A", ProxyID: px.ID, Fingerprint: Fingerprint{OS: "windows", Timezone: "Asia/Tokyo", Languages: []string{"ja-JP"}}})
	runReg.add(p.ID, &RunState{ProfileID: p.ID}, nil)
	t.Cleanup(func() { runReg.mu.Lock(); delete(runReg.m, p.ID); runReg.mu.Unlock(); launchCfg.forget(p.ID) })
	launchCfg.launched(s, p.ID)
	pending := func() bool {
		for _, id := range s.pendingRestart() {
			if id == p.ID {
				return true
			}
		}
		return false
	}
	if pending() {
		t.Fatal("fresh launch must not be pending")
	}
	// name / group / tags / notes / start pages: no flag
	q, _ := s.GetProfile(p.ID)
	q.Name, q.Group, q.Tags, q.Notes, q.StartURLs = "B", "g", []string{"t"}, "n", []string{"https://x.test/"}
	s.PutProfile(*q)
	if pending() {
		t.Fatal("cosmetic edit flagged 待重启")
	}
	// proxy check results must not flag
	cur, _ := s.GetProxy(px.ID)
	cur.IP, cur.Latency, cur.Country, cur.Status = "9.9.9.9", 120, "JP", "ok"
	s.PutProxy(*cur)
	if pending() {
		t.Fatal("proxy check result flagged 待重启")
	}
	cases := []func(p *Profile){
		func(p *Profile) { p.Fingerprint.Timezone = "Europe/Berlin" },
		func(p *Profile) { p.Fingerprint.Languages = []string{"de-DE"} },
		func(p *Profile) { p.Kernel = "chromium" },
		func(p *Profile) { p.ProxyID = "" },
		func(p *Profile) { p.Native = &NativeProfile{ExtraArgs: []string{"--foo"}} },
		func(p *Profile) { p.ExtensionIDs = []string{"ext"} },
	}
	for i, mut := range cases {
		base, _ := s.GetProfile(p.ID)
		saved := *base
		launchCfg.launched(s, p.ID)
		edited := saved
		mut(&edited)
		s.PutProfile(edited)
		if !pending() {
			t.Fatalf("case %d: launch-relevant edit not flagged", i)
		}
		s.PutProfile(saved)
		if pending() {
			t.Fatalf("case %d: reverting the edit should clear the flag", i)
		}
	}
	// editing the assigned proxy endpoint itself counts
	launchCfg.launched(s, p.ID)
	cur, _ = s.GetProxy(px.ID)
	cur.Port = 1081
	s.PutProxy(*cur)
	if !pending() {
		t.Fatal("proxy endpoint edit not flagged")
	}
	// not running -> never pending
	runReg.mu.Lock()
	delete(runReg.m, p.ID)
	runReg.mu.Unlock()
	if pending() {
		t.Fatal("stopped profile reported as pending")
	}
}

func TestLaunchURLOneShot(t *testing.T) {
	launchCfg.setURL("x", "https://a.test/")
	if launchCfg.takeURL("x") != "https://a.test/" || launchCfg.takeURL("x") != "" {
		t.Fatal("one-shot URL must be consumed once")
	}
}

func TestShouldHideOnClose(t *testing.T) {
	if !shouldHideOnClose(Settings{}, true, false) {
		t.Fatal("default must hide to tray")
	}
	if shouldHideOnClose(Settings{CloseToTrayOff: true}, true, false) || shouldHideOnClose(Settings{}, false, false) || shouldHideOnClose(Settings{}, true, true) {
		t.Fatal("off / no tray / quitting must close")
	}
}
