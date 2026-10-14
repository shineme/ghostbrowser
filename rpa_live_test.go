package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const rpaLivePage = `<!doctype html><html><head><title>rpa</title></head><body>
<input id="name" placeholder="name">
<button id="go" onclick="document.getElementById('out').textContent = 'Hello, ' + document.getElementById('name').value + '!'">Go</button>
<div id="out"></div>
</body></html>`

// GHOST_CHROME=/usr/bin/chromium go test -run TestRpaLive
func TestRpaLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	dir := t.TempDir()
	page := filepath.Join(dir, "page.html")
	if err := os.WriteFile(page, []byte(rpaLivePage), 0o644); err != nil {
		t.Fatal(err)
	}
	pageURL := "file://" + page
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+filepath.Join(dir, "ud"), fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	// ---- goto → fill → click → extract_text → execute_js ----
	wf := Workflow{Name: "live", Vars: map[string]string{"who": "Ghost"}, Steps: []RpaStep{
		st(stGoto, "url", pageURL),
		st(stFill, "selector", "#name", "text", "{{who}} 测试"),
		st(stClick, "selector", "#go"),
		st(stWaitFor, "selector", "text:Hello", "timeout", "5000"),
		st(stExtractText, "selector", "#out", "var", "greeting"),
		st(stExecuteJS, "code", "return vars.greeting.length + ':' + document.title", "saveAs", "js"),
		st(stExecuteJS, "code", "document.querySelector('#name').value", "saveAs", "val"),
		withKids(st(stCondition, "kind", "exists", "selector", "#out"), st(stScreenshot, "saveAs", "shot")),
	}}
	var failed []string
	shots := 0
	vars, err := runRpa(context.Background(), wf, d, func(e RpaStepEvent) {
		if e.Status == "failed" {
			failed = append(failed, e.Type+": "+e.Log)
		}
	}, func(_ int, png []byte) (string, error) {
		shots++
		if len(png) < 100 || string(png[1:4]) != "PNG" {
			return "", fmt.Errorf("bad png")
		}
		return "shot.png", nil
	})
	if err != nil {
		t.Fatalf("run: %v (failed steps %v)", err, failed)
	}
	if vars["greeting"] != "Hello, Ghost 测试!" {
		t.Fatalf("greeting = %q", vars["greeting"])
	}
	if want := fmt.Sprintf("%d:rpa", len([]rune("Hello, Ghost 测试!"))); vars["js"] != want {
		t.Fatalf("js = %q want %q", vars["js"], want)
	}
	if vars["val"] != "Ghost 测试" || shots != 1 || vars["shot"] != "shot.png" {
		t.Fatalf("val=%q shots=%d vars=%v", vars["val"], shots, vars)
	}
	t.Logf("vars: greeting=%q js=%q", vars["greeting"], vars["js"])

	// ---- tab ordering ----
	raw, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.c.Close()
	urlOf := func(id string) string {
		r, _ := raw.call("", "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ TargetID, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(r, &tl)
		for _, ti := range tl.TargetInfos {
			if ti.TargetID == id {
				return ti.URL
			}
		}
		return ""
	}
	for _, q := range []string{"t=2", "t=3"} {
		if _, err := raw.call("", "Target.createTarget", map[string]any{"url": pageURL + "?" + q}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(400 * time.Millisecond)
	}
	// Log the raw getTargets order so the assumption in newCDPDriver is checked.
	pages, _ := d.(*cdpDriver).pages()
	order := []string{}
	for _, id := range pages {
		order = append(order, tail(urlOf(id)))
	}
	// Informational: Chromium returns pages in no stable order, which is why
	// the driver tracks creation order itself.
	t.Logf("Target.getTargets page order (unordered): %v", order)

	cd := d.(*cdpDriver)
	tabURLs := func(dr *cdpDriver) []string {
		out := []string{}
		for _, id := range dr.tabs() {
			out = append(out, tail(urlOf(id)))
		}
		return out
	}
	if got := tabURLs(cd); strings.Join(got, ",") != "page.html,t=2,t=3" {
		t.Fatalf("driver tab order = %v", got)
	}
	here := func(dr RpaDriver) string {
		v, err := dr.Eval("location.search || 'none'", nil)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	if err := d.SelectTab(3); err != nil || here(d) != "?t=3" {
		t.Fatalf("select_tab 3: %v %q", err, here(d))
	}
	if err := d.SelectTab(1); err != nil || here(d) != "none" {
		t.Fatalf("select_tab 1: %v", err)
	}
	if err := d.SelectTab(4); err == nil {
		t.Fatal("select_tab 4 should fail with 3 tabs")
	}
	if err := d.CloseTab(2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if got := tabURLs(cd); strings.Join(got, ",") != "page.html,t=3" {
		t.Fatalf("after close_tab 2 = %v", got)
	}
	if here(d) != "none" {
		t.Fatal("closing another tab moved the current tab")
	}
	// close the current tab: driver falls back to the last remaining one
	if err := d.SelectTab(2); err != nil {
		t.Fatal(err)
	}
	if err := d.CloseTab(0); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if got := tabURLs(cd); len(got) != 1 || here(d) != "none" {
		t.Fatalf("after close current: %v %q", got, here(d))
	}
	if err := d.CloseTab(0); err == nil {
		t.Fatal("closing the last tab should be refused")
	}

	// A fresh driver on a browser that already has several tabs must number
	// them oldest first and attach to the newest one.
	if _, err := raw.call("", "Target.createTarget", map[string]any{"url": pageURL + "?t=5"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	d2, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	if got := tabURLs(d2.(*cdpDriver)); strings.Join(got, ",") != "page.html,t=5" {
		t.Fatalf("fresh driver tab order = %v", got)
	}
	if here(d2) != "?t=5" {
		t.Fatalf("fresh driver attached to %q, want newest tab", here(d2))
	}
}

func tail(u string) string {
	if i := strings.LastIndex(u, "?"); i >= 0 {
		return u[i+1:]
	}
	return filepath.Base(u)
}
