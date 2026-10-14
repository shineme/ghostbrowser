package main

import (
	"strings"
	"testing"
)

func recTypes(steps []RpaStep) string {
	out := []string{}
	for _, s := range steps {
		t := s.Type
		switch s.Type {
		case stClick:
			if s.Params["double"] == "true" {
				t += "×2"
			}
		case stDelay:
			t += ":" + s.Params["ms"]
		case stSelectTab, stCloseTab:
			t += ":" + s.Params["index"]
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}

func TestRecordCoalesceTyping(t *testing.T) {
	c := newRecCore(RecordOptions{})
	c.addTab("A", "about:blank", false)
	c.navigated("A", "https://x.test/login", false, "", 1000)
	for i, v := range []string{"a", "al", "ali", "alic", "alice"} {
		c.event("A", recEvent{T: "input", Sel: "#user", V: v, TS: int64(1100 + i*80)})
	}
	// a pause mid-typing does not split the fill
	c.event("A", recEvent{T: "input", Sel: "#user", V: "alice1", TS: 4000})
	for i, v := range []string{"s", "se", "sec"} {
		c.event("A", recEvent{T: "input", Sel: `input[name="pw"]`, V: v, PW: true, TS: int64(4100 + i*50)})
	}
	c.event("A", recEvent{T: "select", Sel: "#city", V: "bj", TS: 4300})
	c.event("A", recEvent{T: "select", Sel: "#city", V: "sh", TS: 4400})
	if got := recTypes(c.steps); got != "goto fill fill fill" {
		t.Fatalf("steps: %s", got)
	}
	if c.steps[1].Params["text"] != "alice1" || c.steps[2].Params["text"] != "{{password}}" || c.steps[3].Params["text"] != "sh" {
		t.Fatalf("values: %+v", c.steps)
	}
	if c.secrets[c.steps[2].ID] != "sec" || c.steps[2].Params[recRecTag] != "password" {
		t.Fatalf("password secret not kept: %v", c.secrets)
	}
	// finalize: variable by default, real value when asked
	f := finalizeRecording(c.steps, RecordFinalizeOptions{PasswordVar: true}, c.secrets)
	if f.Steps[2].Params["text"] != "{{password}}" || f.Vars["password"] != "" || len(f.Vars) != 1 || f.Steps[2].Params[recRecTag] != "" {
		t.Fatalf("finalize var: %+v %v", f.Steps[2], f.Vars)
	}
	f = finalizeRecording(c.steps, RecordFinalizeOptions{PasswordVar: false}, c.secrets)
	if f.Steps[2].Params["text"] != "sec" || len(f.Vars) != 0 {
		t.Fatalf("finalize plain: %+v %v", f.Steps[2], f.Vars)
	}
}

func TestRecordClicksAndThinkTime(t *testing.T) {
	c := newRecCore(RecordOptions{MaxDelayMs: 3000})
	c.addTab("A", "https://x.test/", false)
	c.event("A", recEvent{T: "down", Sel: "#a", TS: 100})
	c.event("A", recEvent{T: "click", Sel: "#a", TS: 180})
	c.event("A", recEvent{T: "click", Sel: "#b", TS: 2000}) // 1.8 s think
	c.event("A", recEvent{T: "dbl", Sel: "#b", TS: 2100})
	c.event("A", recEvent{T: "click", Sel: ".late", Dyn: true, TS: 12000}) // capped, wait inserted
	c.event("A", recEvent{T: "enter", Sel: "#q", TS: 12100})
	c.event("A", recEvent{T: "upload", Sel: "#f", Files: []string{"a.png"}, TS: 12200})
	c.event("A", recEvent{T: "scroll", X: 0, Y: 300, TS: 12300}) // scroll off by default
	c.event("A", recEvent{T: "extract", Sel: "#price", TS: 12400})
	c.event("A", recEvent{T: "assert", Sel: "#done", TS: 12500})
	if got := recTypes(c.steps); got != "click delay:1800 click×2 delay:3000 wait_for click execute_js upload extract_text wait_for" {
		t.Fatalf("steps: %s", got)
	}
	if c.steps[4].Params["selector"] != ".late" || c.steps[7].Params["files"] != "{{upload_file}}" || c.steps[8].Params["var"] != "text1" {
		t.Fatalf("params: %+v", c.steps)
	}
	f := finalizeRecording(c.steps, RecordFinalizeOptions{}, nil)
	if got := recTypes(f.Steps); got != "click click×2 wait_for click execute_js upload extract_text wait_for" {
		t.Fatalf("finalized: %s", got)
	}
	if _, ok := f.Vars["upload_file"]; !ok || f.Steps[4].Params["selector"] != "" || !strings.Contains(f.Steps[4].Params["code"], "requestSubmit") {
		t.Fatalf("finalize: %v %+v", f.Vars, f.Steps[4])
	}
	if got := recTypes(finalizeRecording(c.steps, RecordFinalizeOptions{KeepDelays: true}, nil).Steps); !strings.Contains(got, "delay:1800") {
		t.Fatalf("keep delays: %s", got)
	}
}

func TestRecordScrollCoalesce(t *testing.T) {
	c := newRecCore(RecordOptions{Scroll: true})
	c.addTab("A", "", false)
	for i := 1; i <= 4; i++ {
		c.event("A", recEvent{T: "scroll", Y: i * 200, TS: int64(i * 100)})
	}
	if len(c.steps) != 1 || c.steps[0].Params["y"] != "800" || c.steps[0].Params["mode"] != "to" {
		t.Fatalf("scroll: %+v", c.steps)
	}
}

func TestRecordNavigationAndTabs(t *testing.T) {
	c := newRecCore(RecordOptions{})
	c.addTab("A", "about:blank", false)
	c.navigated("A", "chrome://newtab/", false, "", 10) // ignored
	c.navigated("A", "https://x.test/", false, "", 20)  // typed → goto
	c.navigated("A", "https://x.test/", false, "", 30)  // reload → nothing
	c.event("A", recEvent{T: "click", Sel: "text:下一页", TS: 40})
	c.navigated("A", "https://x.test/2", true, "anchorClick", 60) // caused by the click
	// The click died with the old document; only its pointerdown made it out.
	c.event("A", recEvent{T: "down", Sel: "#go", TS: 100})
	c.navigated("A", "https://x.test/3", true, "anchorClick", 150)
	// a script redirect without any user action is not recorded either
	c.navigated("A", "https://x.test/4", true, "scriptInitiated", 160)
	// link opens a new tab; the user works there, closes it, goes back
	c.event("A", recEvent{T: "click", Sel: "#open", TS: 200})
	c.addTab("B", "", true)
	c.navigated("B", "https://x.test/tab", false, "", 250) // first page of an opened tab
	c.event("B", recEvent{T: "click", Sel: "#hello", TS: 300})
	c.event("B", recEvent{T: "click", Sel: "#hello2", TS: 320})
	c.tabClosed("B", 400)
	c.event("A", recEvent{T: "click", Sel: "#dbl", TS: 500})
	// the user opens a tab and types a URL
	c.addTab("C", "chrome://newtab/", false)
	c.navigated("C", "https://y.test/", false, "", 600)
	c.event("C", recEvent{T: "click", Sel: "#c", TS: 700})
	c.event("A", recEvent{T: "click", Sel: "#a", TS: 800})
	got := recTypes(c.steps)
	want := "goto click click click select_tab:2 click click close_tab:2 click execute_js select_tab:2 click select_tab:1 click"
	if got != want {
		t.Fatalf("steps:\n got %s\nwant %s", got, want)
	}
	if c.steps[2].Params["selector"] != "#go" || !strings.Contains(c.steps[9].Params["code"], "window.open(\"https://y.test/\"") {
		t.Fatalf("params: %+v / %+v", c.steps[2], c.steps[9])
	}
}

func TestRecordMergeFills(t *testing.T) {
	steps := []RpaStep{
		recStep(stFill, "selector", "#a", "text", "1"),
		recStep(stWaitFor, "selector", "#x"),
		recStep(stFill, "selector", "#a", "text", "2"),
		recStep(stClick, "selector", "#go"),
		recStep(stFill, "selector", "#a", "text", "3"),
		recStep(stSelectTab, "index", "2"),
		recStep(stSelectTab, "index", "1"),
	}
	f := finalizeRecording(steps, RecordFinalizeOptions{MergeFills: true}, nil)
	if got := recTypes(f.Steps); got != "wait_for fill click fill select_tab:1" || f.Steps[1].Params["text"] != "2" {
		t.Fatalf("merged: %s %+v", got, f.Steps)
	}
	if got := recTypes(finalizeRecording(steps, RecordFinalizeOptions{}, nil).Steps); got != "fill wait_for fill click fill select_tab:1" {
		t.Fatalf("unmerged: %s", got)
	}
}

func TestRecordEditSteps(t *testing.T) {
	c := newRecCore(RecordOptions{})
	c.addTab("A", "", false)
	c.event("A", recEvent{T: "input", Sel: "#p", V: "x", PW: true, TS: 1})
	id := c.steps[0].ID
	s := c.steps[0]
	s.Params = map[string]string{"selector": "#pw", "text": "{{password}}"}
	c.update(s)
	if c.steps[0].Params["selector"] != "#pw" || c.steps[0].Params[recRecTag] != "password" {
		t.Fatalf("update: %+v", c.steps[0])
	}
	c.remove(id)
	if len(c.steps) != 0 || len(c.secrets) != 0 {
		t.Fatalf("remove: %+v %v", c.steps, c.secrets)
	}
}
