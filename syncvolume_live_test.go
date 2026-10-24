package main

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Sync volume measurement (not a regular test):
//
//	GHOST_PERF_SYNC=1 GHOST_PHP_TEST=php GHOST_CHROME=/usr/bin/chromium \
//	  go test -run TestSyncVolumeLive -v -timeout 30m .
//
// Launches a real Chromium through the launcher, browses, closes, pushes,
// then repeats a short browse twice. Logs the bytes each push sent and the
// files / directories behind them. Knobs: GHOST_PERF_PROXY (http://user:pass@
// host:port for the browser), GHOST_PERF_SNAP (copy the profile directory
// after every close into this folder for offline analysis),
// GHOST_PERF_URLS1 / GHOST_PERF_URLS2 (comma separated), GHOST_PERF_SYNCMODE
// (lite / standard / full). The browser runs headed: on Linux start Xvfb and
// set DISPLAY. v0.8.10 numbers: /workspace/work/perflogs/syncvol/SUMMARY.md.
func TestSyncVolumeLive(t *testing.T) {
	chrome := os.Getenv("GHOST_CHROME")
	if os.Getenv("GHOST_PERF_SYNC") == "" || chrome == "" {
		t.Skip("set GHOST_PERF_SYNC=1 and GHOST_CHROME")
	}
	env := startRepServer(t)
	s := env.dev("a", "dev-A")
	st := s.GetSettings()
	st.ChromiumPath = chrome
	if m := os.Getenv("GHOST_PERF_SYNCMODE"); m != "" {
		st.SyncContent = m
	}
	s.PutSettings(st)
	fp := GenerateFingerprint("windows", "141")
	fp.StartupParameters = "--no-sandbox"
	p := Profile{Name: "volume", Kernel: "chromium", Fingerprint: fp, StartURLs: []string{"about:blank"}}
	if pu := os.Getenv("GHOST_PERF_PROXY"); pu != "" {
		u, e := url.Parse(pu)
		if e != nil {
			t.Fatal(e)
		}
		port, _ := strconv.Atoi(u.Port())
		pw, _ := u.User.Password()
		p.InlineProxy = &Proxy{Type: "http", Host: u.Hostname(), Port: port, User: u.User.Username(), Pass: pw}
	}
	p = s.PutProfile(p)
	dir := s.ProfileDir(p.ID)
	urls := func(k, def string) []string {
		v := os.Getenv(k)
		if v == "" {
			v = def
		}
		return strings.Split(v, ",")
	}
	heavy := urls("GHOST_PERF_URLS1", "https://www.youtube.com/,https://www.google.com/search?q=antidetect+browser,https://en.wikipedia.org/wiki/Web_browser,https://github.com/,https://excalidraw.com/,https://squoosh.app/,https://www.bilibili.com/")
	light := urls("GHOST_PERF_URLS2", "https://www.google.com/search?q=weather,https://en.wikipedia.org/wiki/Chromium_(web_browser)")
	browse := func(list []string, dwell, idle time.Duration) {
		rs, e := s.Launch(p.ID)
		if e != nil {
			t.Fatal(e)
		}
		for _, u := range list {
			_, _ = cdpCall(rs.WSURL, "Target.createTarget", map[string]any{"url": u})
			time.Sleep(dwell)
		}
		time.Sleep(idle)
		_ = closeBrowser(rs.WSURL)
		for i := 0; i < 100; i++ {
			if _, ok := runReg.get(p.ID); !ok {
				break
			}
			time.Sleep(200 * time.Millisecond)
		}
		runReg.stop(p.ID)
		time.Sleep(3 * time.Second) // helpers
	}
	snap := os.Getenv("GHOST_PERF_SNAP")
	var prev map[string]bool
	push := func(round int) {
		if snap != "" {
			dst := filepath.Join(snap, fmt.Sprintf("r%d", round))
			_ = os.RemoveAll(dst)
			_ = os.MkdirAll(snap, 0o755)
			if out, e := exec.Command("cp", "-a", dir, dst).CombinedOutput(); e != nil {
				t.Logf("snapshot: %v %s", e, out)
			}
		}
		var last TransferEvent
		s.onTransfer = func(e TransferEvent) { last = e }
		t0 := time.Now()
		sent, e := s.pushProfile(p.ID)
		if e != nil {
			t.Fatalf("push %d: %v", round, e)
		}
		_, m := manifestIDs(t, s, p.ID)
		var total int64
		per := map[string]int64{}
		cur := map[string]bool{}
		var newPlain int64
		for _, f := range m.Files {
			total += f.Size
			for i, id := range f.Chunks {
				cur[id] = true
				if prev[id] {
					continue
				}
				n := f.Size - int64(i)*chunkSize
				if n > chunkSize {
					n = chunkSize
				}
				newPlain += n
				per[volGroup(f.Path)] += n
			}
		}
		for id := range prev {
			cur[id] = true // the server keeps them (dedupe across versions)
		}
		prev = cur
		t.Logf("PUSH %d: %d files, profile %s, new chunk data %s, wire %s, dock total %s saved %s, %.1fs", round, len(m.Files), mb(total), mb(newPlain), mb(sent), mb(last.Total), mb(last.Saved), time.Since(t0).Seconds())
		type kv struct {
			k string
			v int64
		}
		var l []kv
		for k, v := range per {
			l = append(l, kv{k, v})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].v > l[j].v })
		for i, x := range l {
			if i >= 40 || x.v < 64<<10 {
				break
			}
			t.Logf("   %10s  %s", mb(x.v), x.k)
		}
	}
	browse(heavy, 12*time.Second, 60*time.Second)
	push(1)
	browse(light, 10*time.Second, 15*time.Second)
	push(2)
	browse([]string{"https://www.youtube.com/"}, 10*time.Second, 10*time.Second)
	push(3)
}

func mb(n int64) string { return fmt.Sprintf("%.2fMB", float64(n)/(1<<20)) }

// volGroup folds a path to the directory that explains it (two levels below
// Default/, one at the top).
func volGroup(rel string) string {
	parts := strings.Split(rel, "/")
	n := 1
	if len(parts) > 1 && (parts[0] == "Default" || strings.HasPrefix(parts[0], "Profile ")) {
		n = 2
		if len(parts) > 2 && (parts[1] == "IndexedDB" || parts[1] == "Service Worker" || parts[1] == "Storage" || parts[1] == "Network" || parts[1] == "Extensions" || parts[1] == "Local Extension Settings") {
			n = 3
		}
	}
	if n >= len(parts) {
		return rel
	}
	return strings.Join(parts[:n], "/") + "/"
}
