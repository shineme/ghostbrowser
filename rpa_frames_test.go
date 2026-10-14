package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseFramePath(t *testing.T) {
	cases := map[string][]string{
		"":                                 nil,
		"top":                              nil,
		" 主页面 ":                            nil,
		"iframe#pay":                       {"iframe#pay"},
		"frame:iframe#pay":                 {"iframe#pay"},
		"iframe#pay >> iframe[name=card]":  {"iframe#pay", "iframe[name=card]"},
		"iframe#pay>>iframe[name=card] >>": {"iframe#pay", "iframe[name=card]"},
		`iframe[title="a > b"] >> #x`:      {`iframe[title="a > b"]`, "#x"},
	}
	for in, want := range cases {
		got := parseFramePath(in)
		if strings.Join(got, "|") != strings.Join(want, "|") || len(got) != len(want) {
			t.Errorf("parseFramePath(%q) = %q, want %q", in, got, want)
		}
	}
	if j := joinFramePath([]string{"a", "b"}); j != "a >> b" {
		t.Errorf("join %q", j)
	}
	cur := []string{"a", "b"}
	for in, want := range map[string]string{"": "", "top": "", "parent": "a", "..": "a", "c": "a|b|c", "c >> d": "a|b|c|d"} {
		if got := strings.Join(switchFramePath(cur, in), "|"); got != want {
			t.Errorf("switchFramePath(%q) = %q, want %q", in, got, want)
		}
	}
	if got := switchFramePath(nil, "parent"); got != nil {
		t.Errorf("parent of top = %q", got)
	}
}

// frameFake is a driver that records the frame each call runs in.
type frameFake struct {
	*fakeDriver
	frame   []string
	missing map[string]bool
}

func (f *frameFake) UseFrame(p []string, _ time.Duration) error {
	f.frame = p
	if f.missing[joinFramePath(p)] {
		return errFrameNotReady{"missing"}
	}
	return nil
}

func (f *frameFake) Click(sel string, d bool) error {
	return f.fakeDriver.rec("click [" + joinFramePath(f.frame) + "] " + sel)
}

func (f *frameFake) Exists(sel string) (bool, error) {
	_ = f.fakeDriver.rec("exists [" + joinFramePath(f.frame) + "] " + sel)
	return true, nil
}

func TestEngineFrameContext(t *testing.T) {
	f := &frameFake{fakeDriver: newFake(), missing: map[string]bool{"iframe#gone": true}}
	cond := st("condition", "kind", "exists", "frame", "iframe#gone", "selector", "#x")
	cond.Else = []RpaStep{st("click", "selector", "#else")}
	wf := Workflow{Name: "f", Vars: map[string]string{"fid": "pay"}, Steps: []RpaStep{
		st("click", "selector", "#a"),
		st("click", "frame", "iframe#{{fid}}", "selector", "#b"),
		st("switch_frame", "frame", "iframe#outer"),
		st("click", "selector", "#c"),
		st("switch_frame", "frame", "iframe[name=inner]"),
		st("click", "selector", "#d"),
		st("click", "frame", "iframe#abs", "selector", "#e"), // step frame is absolute
		st("switch_frame", "frame", "parent"),
		st("click", "selector", "#f"),
		st("goto", "url", "http://x/"),
		st("click", "selector", "#g"), // goto keeps the frame context
		st("select_tab", "index", "1"),
		st("click", "selector", "#h"), // a tab switch resets it
		cond,
	}}
	var logs []string
	if _, err := runRpa(context.Background(), wf, f, func(e RpaStepEvent) {
		if e.Status == "ok" {
			logs = append(logs, e.Log)
		}
	}, nil); err != nil {
		t.Fatal(err)
	}
	want := []string{"click [] #a", "click [iframe#pay] #b", "click [iframe#outer] #c", "click [iframe#outer >> iframe[name=inner]] #d",
		"click [iframe#abs] #e", "click [iframe#outer] #f", "goto http://x/", "click [iframe#outer] #g", "selecttab", "click [] #h", "click [] #else"}
	if got := strings.Join(f.calls, "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("calls:\n%s\nwant:\n%s", got, strings.Join(want, "\n"))
	}
	joined := strings.Join(logs, "\n")
	if !strings.Contains(joined, "框架 iframe#pay › #b") || !strings.Contains(joined, "进入框架 iframe#outer › iframe[name=inner]") {
		t.Errorf("logs:\n%s", joined)
	}
	// A driver without frame support refuses frame steps.
	_, err := runRpa(context.Background(), Workflow{Name: "x", Steps: []RpaStep{st("click", "frame", "iframe#a", "selector", "#b")}}, newFake(), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "不支持") {
		t.Errorf("err %v", err)
	}
	if issues := validateWorkflow(Workflow{Name: "v", Steps: []RpaStep{st("switch_frame")}}); len(issues) != 0 {
		t.Errorf("switch_frame with no params should validate (back to top): %v", issues)
	}
}

func TestRecordFramePathsCore(t *testing.T) {
	c := newRecCore(RecordOptions{})
	c.addTab("T", "http://a/", false)
	c.event("T", recEvent{T: "input", Sel: "#n", V: "a", TS: 1, Frame: "iframe#x"})
	c.event("T", recEvent{T: "input", Sel: "#n", V: "ab", TS: 2, Frame: "iframe#x"})
	c.event("T", recEvent{T: "input", Sel: "#n", V: "z", TS: 3}) // same selector, top document: not merged
	c.event("T", recEvent{T: "click", Sel: "#go", TS: 4, Frame: "iframe#x >> iframe#y"})
	got := []string{}
	for _, s := range c.steps {
		got = append(got, s.Type+"["+s.Params["frame"]+"]"+s.Params["selector"]+"="+s.Params["text"])
	}
	want := "fill[iframe#x]#n=ab,fill[]#n=z,click[iframe#x >> iframe#y]#go="
	if strings.Join(got, ",") != want {
		t.Fatalf("got %s", strings.Join(got, ","))
	}
	f := finalizeRecording(append(c.steps, recStep(stFill, "selector", "#n", "text", "q", "frame", "iframe#x")), RecordFinalizeOptions{MergeFills: true}, nil)
	n := 0
	for _, s := range f.Steps {
		if s.Type == stFill {
			n++
		}
	}
	if n != 3 {
		t.Errorf("merge fills crossed frames: %d fills", n)
	}
}
