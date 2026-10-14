package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// End-to-end: real chromium-kernel profiles in a temp store, run through the
// same App binding the 运行 button calls (RunWorkflowWith), auto-launching
// the browsers, with variables / loop / condition / screenshot, run history
// and closing the browsers afterwards. Then the "already open" and "two runs
// on one browser" paths.
//
// GHOST_CHROME=/usr/bin/chromium go test -run TestRpaE2ELive

const rpaShopPage = `<!doctype html><html><head><meta charset="utf-8"><title>shop</title></head><body>
<input id="q"><button id="go">搜索</button>
<div id="res"></div><div id="page">0</div><button id="next" disabled>下一页</button>
<script>
let page = 0;
const render = () => {
  page++; document.getElementById('page').textContent = page;
  const res = document.getElementById('res');
  for (let i = 0; i < 2; i++) { const d = document.createElement('div'); d.className = 'item'; d.textContent = document.getElementById('q').value + '-' + page + '-' + i; res.appendChild(d); }
  document.getElementById('next').disabled = page >= 3;
};
document.getElementById('go').onclick = () => setTimeout(render, 200);
document.getElementById('next').onclick = render;
</script></body></html>`

func newRpaE2EApp(t *testing.T, bin string) *App {
	t.Helper()
	// Not t.TempDir: Chromium helpers can still be writing as the test ends.
	root, err := os.MkdirTemp("", "rpa-e2e-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { time.Sleep(500 * time.Millisecond); _ = os.RemoveAll(root) })
	s := &Store{dir: root, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ChromiumPath: bin, ProfilesDir: filepath.Join(root, "profiles"), DefaultOS: "windows", DeviceID: "e2e"}}
	a := &App{store: s, auto: &autoSyncer{s: s, busy: map[string]bool{}}}
	// rpa() is a process-wide singleton bound to the first App; give this test its own.
	rpaOnce, rpaSvc = sync.Once{}, nil
	t.Cleanup(func() { rpaOnce, rpaSvc = sync.Once{}, nil })
	return a
}

func waitRun(t *testing.T, a *App, runID string, within time.Duration) RpaRun {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		r, err := a.WorkflowRun(runID)
		if err == nil && r.Status != "running" {
			return r
		}
		time.Sleep(200 * time.Millisecond)
	}
	r, _ := a.WorkflowRun(runID)
	t.Fatalf("run %s still %q after %v: %+v", runID, r.Status, within, r.Targets)
	return r
}

func TestRpaE2ELive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	var mu sync.Mutex
	hits := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits[r.URL.Query().Get("u")]++
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, rpaShopPage)
	}))
	defer srv.Close()

	a := newRpaE2EApp(t, bin)
	var pids []string
	for i := 1; i <= 2; i++ {
		fp := GenerateTyped("desktop", "windows", "")
		fp.StartupParameters = "--headless=new --no-sandbox"
		fp.Timezone, fp.TimezoneFromIP = []string{"", "Asia/Tokyo", "Europe/Berlin"}[i], false
		p := a.store.PutProfile(Profile{Name: fmt.Sprintf("E2E 环境 %d", i), Kernel: "chromium", Fingerprint: fp})
		pids = append(pids, p.ID)
	}
	defer func() {
		for _, id := range pids {
			runReg.stop(id)
		}
	}()

	wf, err := a.SaveWorkflow(Workflow{Name: "搜索翻页", Vars: map[string]string{"kw": "幽灵", "base": srv.URL}, Steps: []RpaStep{
		st(stGoto, "url", "{{base}}/shop?u={{kw}}"),
		st(stFill, "selector", "#q", "text", "{{kw}}"),
		st(stClick, "selector", "#go"),
		st(stWaitFor, "selector", ".item", "timeout", "5000"),
		withKids(st(stLoop, "mode", "while", "kind", "exists", "selector", "#next:not([disabled])", "max", "10", "indexVar", "p"),
			st(stClick, "selector", "#next"), st(stDelay, "ms", "80")),
		st(stExtractText, "selector", "#page", "var", "page"),
		withElse(withKids(st(stCondition, "kind", "equals", "left", "{{page}}", "right", "3"),
			st(stExecuteJS, "code", "return document.querySelectorAll('.item').length + '|' + document.querySelector('.item').textContent", "saveAs", "total")),
			st(stExecuteJS, "code", "throw new Error('翻页数不对：' + vars.page)")),
		st(stExecuteJS, "code", "Intl.DateTimeFormat().resolvedOptions().timeZone", "saveAs", "tz"),
		st(stScreenshot, "fullPage", "true", "saveAs", "shot"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if issues := a.ValidateWorkflow(wf); len(issues) != 0 {
		t.Fatalf("issues: %v", issues)
	}

	// ---- 2 profiles in parallel, auto-started and closed afterwards ----
	t0 := time.Now()
	runID, err := a.RunWorkflowWith(wf.ID, pids, RpaRunOptions{Concurrency: 2, AutoStart: true, CloseAfter: true})
	if err != nil {
		t.Fatal(err)
	}
	run := waitRun(t, a, runID, 90*time.Second)
	t.Logf("parallel run %s in %v", run.Status, time.Since(t0).Round(time.Millisecond))
	if run.Status != "ok" {
		for _, tg := range run.Targets {
			t.Logf("target %s: %s %s", tg.ProfileName, tg.Status, tg.Error)
		}
		t.Fatalf("run status %q", run.Status)
	}
	tzs := map[string]bool{}
	for _, tg := range run.Targets {
		if tg.Status != "ok" || !tg.AutoStarted || tg.Vars["page"] != "3" || tg.Vars["total"] != "6|幽灵-1-0" || tg.Vars["p"] != "2" {
			t.Fatalf("target %s: %+v vars=%v", tg.ProfileName, tg.Status, tg.Vars)
		}
		if fi, err := os.Stat(tg.Vars["shot"]); err != nil || fi.Size() < 1000 {
			t.Fatalf("screenshot %q: %v", tg.Vars["shot"], err)
		}
		// goto, fill, click, wait, loop(+2×(click, delay)), extract, condition(+js), js, shot
		if len(tg.Steps) != 14 {
			t.Fatalf("%s: %d step records", tg.ProfileName, len(tg.Steps))
		}
		tzs[tg.Vars["tz"]] = true
	}
	if !tzs["Asia/Tokyo"] || !tzs["Europe/Berlin"] {
		t.Fatalf("each profile should run in its own browser/fingerprint: %v", tzs)
	}
	if hits["幽灵"] != 2 {
		t.Fatalf("server hits %v", hits)
	}
	// browsers the run started are closed again
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		_, r1 := runReg.get(pids[0])
		_, r2 := runReg.get(pids[1])
		if !r1 && !r2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("auto-started browsers were not closed after the run")
		}
	}
	// history
	hist := a.WorkflowRuns(wf.ID)
	if len(hist) != 1 || hist[0].ID != runID || len(hist[0].Targets[0].Steps) != 0 {
		t.Fatalf("history: %+v", hist)
	}
	if b, err := os.ReadFile(filepath.Join(a.store.dir, "rpa_runs.json")); err != nil || !strings.Contains(string(b), runID) {
		t.Fatalf("rpa_runs.json: %v", err)
	}
	if w, _ := a.rpa().db.Get(wf.ID); w.LastStatus != "ok" || w.RunCount != 1 || w.LastRunID != runID {
		t.Fatalf("workflow stats: %+v", w)
	}

	// ---- already-open profile: used as is and left open ----
	if _, err := a.StartProfile(pids[0]); err != nil {
		t.Fatal(err)
	}
	slow := Workflow{Name: "慢流程", Vars: map[string]string{"base": srv.URL}, Steps: []RpaStep{
		st(stGoto, "url", "{{base}}/?u=slow"), st(stDelay, "ms", "1500"), st(stExecuteJS, "code", "document.title", "saveAs", "t"),
	}}
	slow, _ = a.SaveWorkflow(slow)
	r1, err := a.RunWorkflowWith(slow.ID, []string{pids[0]}, RpaRunOptions{Concurrency: 1, AutoStart: true, CloseAfter: true})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	// A second run on the same browser while the first is still driving it.
	r2, err := a.RunWorkflow(slow.ID, []string{pids[0]})
	if err != nil {
		t.Fatal(err)
	}
	run1, run2 := waitRun(t, a, r1, 30*time.Second), waitRun(t, a, r2, 30*time.Second)
	if run1.Status != "ok" || run1.Targets[0].AutoStarted || run1.Targets[0].Vars["t"] != "shop" {
		t.Fatalf("run on open profile: %s %+v", run1.Status, run1.Targets[0])
	}
	if run2.Status != "failed" || !strings.Contains(run2.Targets[0].Error, "正在运行另一个") {
		t.Fatalf("concurrent run on same profile: %s %q", run2.Status, run2.Targets[0].Error)
	}
	if _, open := runReg.get(pids[0]); !open {
		t.Fatal("a browser the user opened was closed by the run")
	}

	// ---- browser closed by the user mid-run ----
	r3, _ := a.RunWorkflowWith(slow.ID, []string{pids[0]}, RpaRunOptions{Concurrency: 1})
	time.Sleep(700 * time.Millisecond)
	a.StopProfile(pids[0])
	run3 := waitRun(t, a, r3, 20*time.Second)
	if run3.Status != "failed" || !strings.Contains(run3.Targets[0].Error, "浏览器") {
		t.Fatalf("closed mid-run: %s %q", run3.Status, run3.Targets[0].Error)
	}
	t.Logf("closed mid-run error: %s", run3.Targets[0].Error)

	// ---- cancel while auto-starting / running ----
	r4, _ := a.RunWorkflowWith(slow.ID, pids, RpaRunOptions{Concurrency: 2, AutoStart: true, CloseAfter: true})
	time.Sleep(2500 * time.Millisecond)
	a.CancelWorkflowRun(r4)
	run4 := waitRun(t, a, r4, 20*time.Second)
	if run4.Status != "cancelled" {
		t.Fatalf("cancelled run status %q %+v", run4.Status, run4.Targets)
	}
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(200 * time.Millisecond) {
		_, o1 := runReg.get(pids[0])
		_, o2 := runReg.get(pids[1])
		if !o1 && !o2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("browsers auto-started by a cancelled run were left open")
		}
	}
}
