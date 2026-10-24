package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// What each 同步内容 mode uploads. tier: 0 lite, 1 standard, 2 full, 3 never.
var scopeFixture = map[string]int{
	// login state / setup: every mode
	"Local State": 0, "Default/Preferences": 0, "Default/Secure Preferences": 0, "Default/Bookmarks": 0,
	"Default/Cookies": 0, "Default/Cookies-journal": 0, "Default/Network/Cookies": 0, "Default/Network/Cookies-journal": 0,
	"Default/Network/Device Bound Sessions": 0, "Default/Network/TransportSecurity": 0,
	"Default/Local Storage/leveldb/000003.log": 0, "Default/Local Storage/leveldb/CURRENT": 0,
	"Default/IndexedDB/https_x.com_0.indexeddb.leveldb/000005.ldb": 0, "Default/IndexedDB/https_x.com_0.indexeddb.blob/1/00/3": 0,
	"Default/WebStorage/QuotaManager": 0, "Default/WebStorage/3/IndexedDB/indexeddb.leveldb/CURRENT": 0,
	"Default/Login Data": 0, "Default/Login Data For Account": 0, "Default/Web Data": 0, "Default/Account Web Data": 0,
	"Default/Extensions/abc/1.0/manifest.json": 0, "Default/Local Extension Settings/abc/000003.log": 0,
	"Default/Extension Rules/CURRENT": 0, "Default/Storage/ext/abc/def/IndexedDB/x": 0,
	"Default/MediaDeviceSalts": 0, "ghost-session.json": 0, "startInfo": 0, "Profile 1/Cookies": 0, "places.sqlite": 0,
	// 标准
	"Default/History": 1, "Default/History-journal": 1, "Default/Favicons": 1, "Default/Top Sites": 1, "Default/Visited Links": 1,
	"Default/Shortcuts": 1, "Default/Sessions/Session_1": 1, "Default/Session Storage/000003.log": 1,
	"Default/Service Worker/Database/CURRENT": 1, "Default/Service Worker/ScriptCache/index": 1,
	"Default/Service Worker/CacheStorage/abc/def/index": 1, "Default/WebStorage/3/CacheStorage/x/index": 1,
	"Default/GCM Store/CURRENT": 1, "Default/File System/000/p/00/1": 1, "Profile 1/History": 1, "favicons.sqlite": 1,
	// 完整
	"Default/DIPS": 2, "Default/DIPS-wal": 2, "Default/Sync Data/LevelDB/000003.log": 2, "Default/Extension State/CURRENT": 2,
	"Default/Network Action Predictor": 2, "Default/Trust Tokens": 2, "Default/shared_proto_db/CURRENT": 2,
	"Default/segmentation_platform/ukm_db": 2, "segmentation_platform/x": 2,
	// never
	"screen_ai/1/model.tflite": 3, "WidevineCdm/4.10/manifest.json": 3, "OnDeviceHeadSuggestModel/1/m": 3,
	"Safe Browsing/UrlSoceng.store.4_1": 3, "ZxcvbnData/3/en.txt": 3, "hyphen-data/1/hyph-en.hyb": 3,
	"Subresource Filter/Unindexed Rules/9/Filtering Rules": 3, "TrustTokenKeyCommitments/1/k": 3, "MEIPreload/1/p": 3,
	"FileTypePolicies/1/f": 3, "SSLErrorAssistant/1/s": 3, "CertificateRevocation/1/c": 3, "Crowd Deny/1/c": 3,
	"OriginTrials/1/o": 3, "PKIMetadata/1/p": 3, "AutofillStates/1/a": 3, "CookieReadinessList/1/c": 3,
	"first_party_sets.db": 3, "first_party_sets.db-journal": 3, "GPUPersistentCache/GPUCache/x/cache.db-wal": 3,
	"Default/Download Service/EntryDB/CURRENT": 3, "Default/Feature Engagement Tracker/EventDB/CURRENT": 3,
	"Default/optimization_guide_hint_cache_store/CURRENT": 3, "Default/heavy_ad_intervention_opt_out.db": 3,
	"Default/commerce_subscription_db/CURRENT": 3, "Default/Shared Dictionary/db": 3, "Default/Shared Dictionary/cache/index": 3,
	"Default/Network/Reporting and NEL": 3, "Default/Reporting and NEL-journal": 3, "Default/BudgetDatabase/CURRENT": 3,
	"Default/Cache/Cache_Data/index": 3, "Default/Code Cache/js/index": 3, "Default/GPUCache/index": 3,
	"GrShaderCache/index": 3, "Default/blob_storage/x/1": 3, "BrowserMetrics/BrowserMetrics-1.pma": 3,
	"launch.log": 3, "lockfile": 3, "SingletonLock": 3, "cache2/entries/AB": 3, "startupCache/startupCache.8.little": 3,
	"Default/Service Worker/CacheStorage/abc/Cache/index": 3,
}

func writeScopeFixture(t *testing.T, dir string) {
	for rel := range scopeFixture {
		writeRel(t, dir, rel, []byte("data of "+rel))
	}
}

func listScope(t *testing.T, dir string, sc contentScope) map[string]bool {
	got := map[string]bool{}
	if e := walkSyncFiles(dir, sc, func(rel string, _ os.FileInfo) error { got[rel] = true; return nil }); e != nil {
		t.Fatal(e)
	}
	return got
}

// Each mode uploads exactly its tiers, in the chunked walk and the legacy zip.
func TestSyncScopeModes(t *testing.T) {
	dir := t.TempDir()
	writeScopeFixture(t, dir)
	for lvl, mode := range []string{syncScopeLite, syncScopeStandard, syncScopeFull} {
		sc := newSyncScope(mode)
		got := listScope(t, dir, sc)
		zb, e := zipDir(dir, &Profile{ID: "x"}, sc)
		if e != nil {
			t.Fatal(e)
		}
		zr, _ := zip.NewReader(bytes.NewReader(zb), int64(len(zb)))
		inZip := map[string]bool{}
		for _, f := range zr.File {
			if f.Name != profileArchiveMetadata {
				inZip[f.Name] = true
			}
		}
		for rel, tier := range scopeFixture {
			want := tier <= lvl
			if got[rel] != want || inZip[rel] != want {
				t.Errorf("%s: %s synced=%v zip=%v, want %v", mode, rel, got[rel], inZip[rel], want)
			}
		}
		if len(got) != len(inZip) {
			t.Errorf("%s: walk %d files, zip %d", mode, len(got), len(inZip))
		}
	}
	if normSyncScope("") != syncScopeLite || normSyncScope("FULL") != syncScopeFull || normSyncScope("bogus") != syncScopeLite {
		t.Fatal("normSyncScope")
	}
	st := Settings{SyncContent: "standard"}
	if syncScopeFor(st, &Profile{}) != syncScopeStandard || syncScopeFor(st, &Profile{SyncContent: "full"}) != syncScopeFull || syncScopeFor(Settings{}, nil) != syncScopeLite {
		t.Fatal("syncScopeFor")
	}
}

// The size estimate matches what each mode walks.
func TestSyncSizeEstimate(t *testing.T) {
	dir := t.TempDir()
	writeScopeFixture(t, dir)
	est := estimateSyncSizes(dir)
	var want [4]int64
	for rel, tier := range scopeFixture {
		n := int64(len("data of " + rel))
		for l := tier; l <= 2; l++ {
			want[l] += n
		}
		if tier == 3 {
			want[3] += n
		}
	}
	if est.Lite != want[0] || est.Standard != want[1] || est.Full != want[2] || est.Never != want[3] || !est.Exists {
		t.Fatalf("estimate %+v want %v", est, want)
	}
}

// A pull keeps what the cloud copy deliberately left out, never what the
// uploader synced (that was deleted on purpose), never overwrites restored files.
func TestCarryLocalOnly(t *testing.T) {
	old, dst := t.TempDir(), t.TempDir()
	writeScopeFixture(t, old)
	writeRel(t, old, "Default/Gone-in-cloud", []byte("deleted remotely"))
	writeRel(t, dst, "Default/Cookies", []byte("from cloud"))
	writeRel(t, dst, "Default/History", []byte("history from cloud")) // a standard upload could restore it
	carryLocalOnly(old, dst, newSyncScope(syncScopeLite))
	if string(readFile(t, dst, "Default/Cookies")) != "from cloud" || string(readFile(t, dst, "Default/History")) != "history from cloud" {
		t.Fatal("restored files must not be overwritten")
	}
	if _, e := os.Stat(filepath.Join(dst, "Default", "Gone-in-cloud")); e == nil {
		t.Fatal("a synced-tier file missing from the cloud copy must not come back")
	}
	for _, rel := range []string{"Default/Favicons", "Default/Service Worker/Database/CURRENT", "screen_ai/1/model.tflite", "Default/Cache/Cache_Data/index", "Default/DIPS"} {
		if string(readFile(t, dst, rel)) != "data of "+rel {
			t.Errorf("%s should have been kept from the local copy", rel)
		}
	}
	for _, rel := range []string{"SingletonLock", "lockfile"} {
		if _, e := os.Stat(filepath.Join(dst, rel)); e == nil {
			t.Errorf("%s must not be carried", rel)
		}
	}
}

func readFile(t *testing.T, dir, rel string) []byte {
	b, _ := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	return b
}

// PHP round trip: a v0.8.9-style manifest (everything, no scope) restores
// history; a lite push leaves it out; pulling that lite copy back on the
// device that has history keeps it; another device gets login data only.
func TestSyncScopeRoundTripCompat(t *testing.T) {
	env := startRepServer(t)
	a, b, c := env.dev("a", "dev-A"), env.dev("b", "dev-B"), env.dev("c", "dev-C")
	p, _ := repProfile(t, a, "scope", 256<<10)
	writeScopeFixture(t, a.ProfileDir(p.ID))
	st := a.GetSettings()
	st.SyncContent = syncScopeFull
	a.PutSettings(st)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_, m := manifestIDs(t, a, p.ID)
	if m.Scope != syncScopeFull {
		t.Fatalf("manifest scope %q", m.Scope)
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	for rel, tier := range scopeFixture {
		has := len(readRel(b, p.ID, rel)) > 0
		if has != (tier <= tierFull) {
			t.Errorf("full pull on b: %s present=%v", rel, has)
		}
	}
	// b syncs 精简 (the default): its push drops history, SW, caches
	writeRel(t, b.ProfileDir(p.ID), "Default/Local Storage/leveldb/000003.log", []byte("ls updated on b"))
	writeRel(t, b.ProfileDir(p.ID), "Default/History", []byte("history on b"))
	if _, e := b.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_, m = manifestIDs(t, b, p.ID)
	if m.Scope != syncScopeLite {
		t.Fatalf("manifest scope %q", m.Scope)
	}
	for _, f := range m.Files {
		if tier := scopeFixture[f.Path]; tier > tierLite {
			t.Errorf("lite push uploaded %s", f.Path)
		}
	}
	// a pulls the lite copy: login data from b, a's own history kept
	if _, e := a.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	if string(readRel(a, p.ID, "Default/Local Storage/leveldb/000003.log")) != "ls updated on b" {
		t.Fatal("login data from b not restored")
	}
	if string(readRel(a, p.ID, "Default/History")) != "data of Default/History" {
		t.Fatal("a lost its local history on pull")
	}
	if string(readRel(a, p.ID, "screen_ai/1/model.tflite")) != "data of screen_ai/1/model.tflite" {
		t.Fatal("component data dropped on pull")
	}
	// a brand-new device only gets what the lite copy carries
	if _, e := c.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	if len(readRel(c, p.ID, "Default/History")) != 0 || string(readRel(c, p.ID, "Default/Cookies")) != "data of Default/Cookies" {
		t.Fatal("new device restore")
	}
	// the older full version still restores everything
	if _, e := c.pullProfile(p.ID, 1); e != nil {
		t.Fatal(e)
	}
	if string(readRel(c, p.ID, "Default/History")) != "data of Default/History" {
		t.Fatal("old full manifest did not restore history")
	}
}

// Live (GHOST_PHP_TEST + GHOST_CHROME): log in on a local site in a real
// Chromium, close, push 精简, pull on another device, launch: the cookie and
// localStorage survive.
func TestSyncScopeLiveLogin(t *testing.T) {
	chrome := os.Getenv("GHOST_CHROME")
	if chrome == "" {
		t.Skip("set GHOST_CHROME")
	}
	env := startRepServer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "logged-in-42", Path: "/", Expires: time.Now().Add(48 * time.Hour)})
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>site</body></html>")
	}))
	defer srv.Close()
	a, b := env.dev("a", "dev-A"), env.dev("b", "dev-B")
	for _, s := range []*Store{a, b} {
		st := s.GetSettings()
		st.ChromiumPath = chrome
		s.PutSettings(st)
	}
	fp := GenerateFingerprint("windows", "141")
	fp.StartupParameters = "--no-sandbox --headless=new --password-store=basic"
	p := a.PutProfile(Profile{Name: "login", Kernel: "chromium", Fingerprint: fp, StartURLs: []string{"about:blank"}})
	eval := func(s *Store, js string) string {
		rs, ok := runReg.get(p.ID)
		if !ok {
			t.Fatal("not running")
		}
		c, e := dialCDP(rs.WSURL, nil)
		if e != nil {
			t.Fatal(e)
		}
		defer c.c.Close()
		raw, e := c.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/app"})
		if e != nil {
			t.Fatal(e)
		}
		var tt struct{ TargetID string }
		_ = json.Unmarshal(raw, &tt)
		sid, e := attachTarget(c, tt.TargetID)
		if e != nil {
			t.Fatal(e)
		}
		var out string
		for end := time.Now().Add(15 * time.Second); time.Now().Before(end); time.Sleep(150 * time.Millisecond) {
			if evalValue(c, sid, 0, `document.readyState==="complete"?"ok":""`, &out) == nil && out == "ok" {
				break
			}
		}
		if e := evalValue(c, sid, 0, js, &out); e != nil {
			t.Fatal(e)
		}
		return out
	}
	stop := func(s *Store) {
		if rs, ok := runReg.get(p.ID); ok {
			_ = closeBrowser(rs.WSURL)
		}
		for i := 0; i < 100; i++ {
			if _, ok := runReg.get(p.ID); !ok {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		runReg.stop(p.ID)
		time.Sleep(2 * time.Second)
	}
	if _, e := a.Launch(p.ID); e != nil {
		t.Fatal(e)
	}
	eval(a, `(async()=>{await fetch('/login');localStorage.setItem('auth','tok-lite');return 'ok'})()`)
	stop(a)
	if _, e := a.pushProfile(p.ID); e != nil {
		t.Fatal(e)
	}
	_, m := manifestIDs(t, a, p.ID)
	for _, f := range m.Files {
		if strings.HasPrefix(f.Path, "Default/History") || strings.HasPrefix(f.Path, "Default/Cache/") {
			t.Fatalf("lite push uploaded %s", f.Path)
		}
	}
	if _, e := b.pullProfile(p.ID, 0); e != nil {
		t.Fatal(e)
	}
	bp, _ := b.GetProfile(p.ID)
	bp.Fingerprint.StartupParameters = fp.StartupParameters
	b.PutProfile(*bp)
	if _, e := b.Launch(p.ID); e != nil {
		t.Fatal(e)
	}
	defer stop(b)
	got := eval(b, `document.cookie+"|"+localStorage.getItem('auth')`)
	if !strings.Contains(got, "sid=logged-in-42") || !strings.HasSuffix(got, "|tok-lite") {
		t.Fatalf("login state after lite round trip: %q", got)
	}
}
