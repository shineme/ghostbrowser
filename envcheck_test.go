package main

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func envFindItem(t *testing.T, r EnvReport, key string) EnvItem {
	t.Helper()
	for _, it := range r.Items {
		if it.Key == key {
			return it
		}
	}
	t.Fatalf("item %s missing", key)
	return EnvItem{}
}

func badProfile() Profile {
	fp := GenerateFingerprint("windows", "130")
	fp.Languages = []string{"zh-CN", "zh"}
	fp.LanguageFromIP, fp.TimezoneFromIP, fp.GeoFromIP = false, false, false
	fp.Timezone = "Asia/Shanghai"
	fp.WebRTC = "real"
	fp.Latitude, fp.Longitude = 31.23, 121.47
	fp.CanvasMode, fp.NoiseSeed = "noise", 0
	fp.RandomFingerprintOnLaunch = false
	fp.Fonts = append(fp.Fonts, "PingFang SC")
	fp.StartupParameters = "--lang=zh-CN --enable-automation"
	return Profile{ID: "p1", Name: "测试环境", Fingerprint: fp}
}

var usExit = GeoMatch{OK: true, IP: "203.0.113.7", Country: "US", Region: "California", City: "Los Angeles", Timezone: "America/Los_Angeles", Languages: []string{"en-US", "en"}, Lat: 34.05, Lon: -118.24}

func TestEnvReportAssembly(t *testing.T) {
	px := &Proxy{ID: "x", Type: "socks5", Host: "1.2.3.4", Port: 1080, Country: "US"}
	r := buildEnvReport(envInput{Profile: badProfile(), Proxy: px, Exit: usExit, Now: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Clock: ClockReport{Status: "fail", SkewSecs: 300, Detail: "x"}})
	want := map[string]string{"exit": "pass", "language": "fail", "timezone": "fail", "geo": "fail", "webrtc": "fail", "webdriver": "fail", "fonts": "fail", "seed": "warn", "ua": "pass", "clock": "fail"}
	for k, st := range want {
		if it := envFindItem(t, r, k); it.Status != st {
			t.Errorf("%s: status %s, want %s (%s)", k, it.Status, st, it.Detail)
		}
	}
	if envFindItem(t, r, "clock").Fixable || envFindItem(t, r, "ua").Fixable || envFindItem(t, r, "exit").Fixable {
		t.Error("clock/ua/exit must be report-only")
	}
	if got := strings.Join(r.Fixable, ","); got != "language,timezone,geo,webrtc,webdriver,fonts,seed" {
		t.Errorf("fixable = %s", got)
	}
	if r.Pass+r.Warn+r.Fail != len(r.Items) {
		t.Errorf("counts %d/%d/%d of %d", r.Pass, r.Warn, r.Fail, len(r.Items))
	}

	// Same offset, different zone name -> warn; follow-IP -> pass; unknown exit -> unknown.
	p := badProfile()
	p.Fingerprint.Timezone = "America/Vancouver"
	if it := checkTimezone(p.Fingerprint, usExit, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)); it.Status != "warn" {
		t.Errorf("same-offset tz: %s", it.Status)
	}
	p.Fingerprint.TimezoneFromIP = true
	if it := checkTimezone(p.Fingerprint, usExit, time.Now()); it.Status != "pass" {
		t.Errorf("follow-IP tz: %s", it.Status)
	}
	r2 := buildEnvReport(envInput{Profile: badProfile(), Exit: GeoMatch{Error: "timeout"}, Now: time.Now(), Clock: ClockReport{Status: "unknown"}})
	if envFindItem(t, r2, "exit").Status != "fail" || envFindItem(t, r2, "language").Status != "unknown" || envFindItem(t, r2, "clock").Actual != "尚未确认" {
		t.Error("unknown-exit report wrong")
	}
	// Direct connection: real WebRTC is fine.
	if envFindItem(t, r2, "webrtc").Status != "pass" {
		t.Error("webrtc real without proxy should pass")
	}
	// UA / platform mismatch.
	fp := GenerateFingerprint("windows", "130")
	fp.Platform = "MacIntel"
	if it := checkUA(fp); it.Status != "fail" {
		t.Errorf("ua mismatch: %s", it.Status)
	}
}

func TestEnvRepairAppliesLogsRestores(t *testing.T) {
	s := testStore(t)
	p := s.PutProfile(badProfile())
	orig := p.Fingerprint
	items := []string{"language", "timezone", "geo", "webrtc", "seed", "fonts", "webdriver", "clock", "ua"}
	res, err := s.applyRepair(&p, items, usExit, true)
	if err != nil || res.Entry == nil {
		t.Fatalf("repair: %v %+v", err, res)
	}
	got, _ := s.GetProfile(p.ID)
	fp := got.Fingerprint
	if !fp.LanguageFromIP || fp.Languages[0] != "en-US" || !fp.TimezoneFromIP || fp.Timezone != "America/Los_Angeles" || !fp.GeoFromIP || fp.Latitude != 34.05 {
		t.Errorf("geo fields not repaired: %+v", fp)
	}
	if fp.WebRTC != "proxy" || fp.NoiseSeed == 0 || strings.Contains(fp.StartupParameters, "automation") || fp.StartupParameters != "--lang=zh-CN" {
		t.Errorf("webrtc/seed/webdriver not repaired: %q %d %q", fp.WebRTC, fp.NoiseSeed, fp.StartupParameters)
	}
	for _, f := range fp.Fonts {
		if f == "PingFang SC" {
			t.Error("foreign font kept")
		}
	}
	if len(res.Skipped) != 2 {
		t.Errorf("skipped = %v (clock and ua must be report-only)", res.Skipped)
	}
	// Re-check passes for the repaired items.
	r := buildEnvReport(envInput{Profile: *got, Proxy: &Proxy{Type: "http", Host: "h"}, Exit: usExit, Now: time.Now(), Clock: ClockReport{Status: "pass"}})
	for _, k := range []string{"language", "timezone", "geo", "webrtc", "webdriver", "fonts", "seed"} {
		if it := envFindItem(t, r, k); it.Status != "pass" {
			t.Errorf("after repair %s = %s (%s)", k, it.Status, it.Detail)
		}
	}
	// A later unrelated edit survives the restore.
	got.Fingerprint.ScreenWidth = 1234
	s.PutProfile(*got)

	h := s.RepairHistory(p.ID)
	if len(h) != 1 || h[0].Kind != "repair" || len(h[0].Changes) < 8 {
		t.Fatalf("history: %+v", h)
	}
	for _, c := range h[0].Changes {
		if c.Field == "timezone" && (c.Old != "Asia/Shanghai" || c.New != "America/Los_Angeles") {
			t.Errorf("timezone change logged as %s -> %s", c.Old, c.New)
		}
	}
	if _, err := s.RestoreRepair(p.ID, h[0].ID); err != nil {
		t.Fatal(err)
	}
	back, _ := s.GetProfile(p.ID)
	b := back.Fingerprint
	if b.Timezone != orig.Timezone || b.TimezoneFromIP || b.WebRTC != "real" || b.NoiseSeed != 0 || b.StartupParameters != orig.StartupParameters || len(b.Fonts) != len(orig.Fonts) || b.Languages[0] != "zh-CN" || b.Latitude != orig.Latitude {
		t.Errorf("restore incomplete: %+v", b)
	}
	if b.ScreenWidth != 1234 {
		t.Error("restore clobbered a later edit")
	}
	h = s.RepairHistory(p.ID)
	if len(h) != 2 || h[0].Kind != "restore" || !h[1].Restored || h[0].RestoreOf != h[1].ID {
		t.Fatalf("history after restore: %+v", h)
	}
	if _, err := s.RestoreRepair(p.ID, h[1].ID); err == nil {
		t.Error("double restore allowed")
	}
	// Nothing to change -> no entry.
	res, _ = s.applyRepair(back, []string{"clock"}, usExit, true)
	if res.Entry != nil || len(s.RepairHistory(p.ID)) != 2 {
		t.Error("no-op repair logged")
	}
}

func TestEnvClockSkewClassification(t *testing.T) {
	srv := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	probe := func(d time.Duration, err error) clockProbe {
		return func() (time.Time, time.Time, string, error) {
			if err != nil {
				return time.Time{}, time.Time{}, "", err
			}
			return srv, srv.Add(d), "https://fake/", nil
		}
	}
	cases := []struct {
		d    time.Duration
		st   string
		skew int64
	}{{3 * time.Second, "pass", 3}, {-90 * time.Second, "pass", -90}, {120 * time.Second, "pass", 120}, {121 * time.Second, "fail", 121}, {-10 * time.Minute, "fail", -600}}
	for _, c := range cases {
		r := CheckClock(probe(c.d, nil))
		if r.Status != c.st || r.SkewSecs != c.skew {
			t.Errorf("%v: %s %d", c.d, r.Status, r.SkewSecs)
		}
	}
	if r := CheckClock(probe(0, errors.New("offline"))); r.Status != "unknown" || !strings.Contains(r.Detail, "尚未确认") {
		t.Errorf("no reference: %+v", r)
	}
	if r := CheckClock(probe(-10*time.Minute, nil)); !strings.Contains(r.Detail, "慢") || !strings.Contains(r.Detail, "自动") {
		t.Errorf("guidance: %s", r.Detail)
	}
	if d, err := parseHTTPDate("Wed, 14 Oct 2026 12:00:00 GMT"); err != nil || !d.Equal(srv) {
		t.Errorf("Date header parse: %v %v", d, err)
	}
}

// TestEnvClockLive hits the real HTTPS reference servers (GHOST_CHROME set = live tests on).
func TestEnvClockLive(t *testing.T) {
	if os.Getenv("GHOST_CHROME") == "" {
		t.Skip("live test: set GHOST_CHROME")
	}
	r := CheckClock(nil)
	t.Logf("clock: %+v", r)
	if r.Status == "unknown" {
		t.Skipf("no reference reachable: %s", r.Error)
	}
	if r.Source == "" || r.Server == 0 {
		t.Errorf("bad live report %+v", r)
	}
}

// TestEnvCheckLive runs the full check on a direct-connection profile (real
// exit-IP lookup + real clock).
func TestEnvCheckLive(t *testing.T) {
	if os.Getenv("GHOST_CHROME") == "" {
		t.Skip("live test: set GHOST_CHROME")
	}
	s := testStore(t)
	p := s.PutProfile(badProfile())
	r, err := s.CheckEnvironment(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range r.Items {
		t.Logf("%-9s %-7s %s | %s | %s", it.Key, it.Status, it.Actual, it.Expected, it.Detail)
	}
	if len(r.Items) != 10 {
		t.Errorf("items = %d", len(r.Items))
	}
}
