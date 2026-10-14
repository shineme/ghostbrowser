package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeDriver records calls and serves canned texts / existence answers.
type fakeDriver struct {
	calls  []string
	texts  map[string]string
	exists map[string]bool
	fail   map[string]error // by method name
	evalFn func(code string, vars map[string]string) (string, error)
}

func newFake() *fakeDriver {
	return &fakeDriver{texts: map[string]string{}, exists: map[string]bool{}, fail: map[string]error{}}
}

func (f *fakeDriver) rec(s string) error {
	f.calls = append(f.calls, s)
	return f.fail[strings.SplitN(s, " ", 2)[0]]
}
func (f *fakeDriver) Goto(u string, _ time.Duration) error { return f.rec("goto " + u) }
func (f *fakeDriver) Click(sel string, _ bool) error       { return f.rec("click " + sel) }
func (f *fakeDriver) Fill(sel, t string, _ bool) error     { return f.rec("fill " + sel + "=" + t) }
func (f *fakeDriver) WaitFor(sel, _ string, _ time.Duration) error {
	return f.rec("wait " + sel)
}
func (f *fakeDriver) Exists(sel string) (bool, error) { return f.exists[sel], f.rec("exists " + sel) }
func (f *fakeDriver) ScrollTo(sel string) error       { return f.rec("scrollto " + sel) }
func (f *fakeDriver) ScrollBy(x, y int, _ bool) error { return f.rec("scroll") }
func (f *fakeDriver) Text(sel, _ string) (string, error) {
	return f.texts[sel], f.rec("text " + sel)
}
func (f *fakeDriver) Eval(code string, vars map[string]string) (string, error) {
	if err := f.rec("eval " + code); err != nil {
		return "", err
	}
	if f.evalFn != nil {
		return f.evalFn(code, vars)
	}
	return "null", nil
}
func (f *fakeDriver) Screenshot(bool) ([]byte, error)    { return []byte("png"), f.rec("shot") }
func (f *fakeDriver) Upload(sel string, _ []string) error { return f.rec("upload " + sel) }
func (f *fakeDriver) SelectTab(i int) error               { return f.rec("selecttab") }
func (f *fakeDriver) CloseTab(i int) error                { return f.rec("closetab") }

func st(typ string, kv ...string) RpaStep {
	p := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		p[kv[i]] = kv[i+1]
	}
	return RpaStep{ID: newID(), Type: typ, Params: p}
}

func withKids(s RpaStep, kids ...RpaStep) RpaStep { s.Children = kids; return s }

func run(t *testing.T, d RpaDriver, steps ...RpaStep) (map[string]string, []RpaStepEvent, error) {
	t.Helper()
	var evs []RpaStepEvent
	vars, err := runRpa(context.Background(), Workflow{Name: "t", Steps: steps, Vars: map[string]string{"site": "example.com"}}, d,
		func(e RpaStepEvent) { evs = append(evs, e) }, nil)
	return vars, evs, err
}

func countCalls(f *fakeDriver, prefix string) int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func TestRpaInterpolation(t *testing.T) {
	if got := interpolate("a {{x}} b {{ y }} {{missing}}", map[string]string{"x": "1", "y": "2"}); got != "a 1 b 2 {{missing}}" {
		t.Fatalf("interpolate = %q", got)
	}
	if got := interpolate("{{用户名}}", map[string]string{"用户名": "张三"}); got != "张三" {
		t.Fatalf("han var = %q", got)
	}
	f := newFake()
	f.texts["#name"] = "  Alice  "
	vars, _, err := run(t, f,
		st(stGoto, "url", "{{site}}/login"),
		st(stExtractText, "selector", "#name", "var", "who"),
		st(stFill, "selector", "#q", "text", "hi {{who}}"))
	if err != nil {
		t.Fatal(err)
	}
	if vars["who"] != "Alice" {
		t.Fatalf("extract_text var = %q", vars["who"])
	}
	want := []string{"goto https://example.com/login", "text #name", "fill #q=hi Alice"}
	if strings.Join(f.calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %v", f.calls)
	}
}

func TestRpaCondition(t *testing.T) {
	f := newFake()
	f.exists["#yes"] = true
	c1 := withKids(st(stCondition, "kind", "exists", "selector", "#yes"), st(stClick, "selector", "#then1"))
	c1.Else = []RpaStep{st(stClick, "selector", "#else1")}
	c2 := withKids(st(stCondition, "kind", "exists", "selector", "#no"), st(stClick, "selector", "#then2"))
	c2.Else = []RpaStep{st(stClick, "selector", "#else2")}
	c3 := withKids(st(stCondition, "kind", "equals", "left", "{{site}}", "right", "example.com"), st(stClick, "selector", "#eq"))
	c4 := withKids(st(stCondition, "kind", "js", "code", "return 0"), st(stClick, "selector", "#js"))
	f.evalFn = func(string, map[string]string) (string, error) { return "false", nil }
	if _, _, err := run(t, f, c1, c2, c3, c4); err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, c := range f.calls {
		if strings.HasPrefix(c, "click") {
			got = append(got, c)
		}
	}
	if strings.Join(got, "|") != "click #then1|click #else2|click #eq" {
		t.Fatalf("branches = %v", got)
	}
}

func TestRpaLoopCountAndWhileCap(t *testing.T) {
	f := newFake()
	vars, _, err := run(t, f, withKids(st(stLoop, "mode", "count", "count", "4", "indexVar", "i"), st(stClick, "selector", "#b{{i}}")))
	if err != nil {
		t.Fatal(err)
	}
	if countCalls(f, "click") != 4 || f.calls[3] != "click #b4" || vars["i"] != "4" {
		t.Fatalf("count loop calls=%v vars=%v", f.calls, vars)
	}
	// while with an always-true condition stops at max
	f = newFake()
	f.exists["#more"] = true
	_, evs, err := run(t, f, withKids(st(stLoop, "mode", "while", "kind", "exists", "selector", "#more", "max", "7"), st(stClick, "selector", "#next")))
	if err != nil {
		t.Fatal(err)
	}
	if countCalls(f, "click") != 7 {
		t.Fatalf("while cap: %d clicks", countCalls(f, "click"))
	}
	if last := evs[len(evs)-1]; !strings.Contains(last.Log, "上限") {
		t.Fatalf("expected cap log, got %q", last.Log)
	}
	// while with no max uses the global cap
	f = newFake()
	f.exists["#more"] = true
	if _, _, err := run(t, f, withKids(st(stLoop, "mode", "while", "selector", "#more"), st(stDelay, "ms", "0"))); err != nil {
		t.Fatal(err)
	}
	if n := countCalls(f, "exists"); n != rpaMaxWhile {
		t.Fatalf("default while cap: %d checks", n)
	}
	// while condition false from the start: zero iterations
	f = newFake()
	if _, _, err := run(t, f, withKids(st(stLoop, "mode", "while", "selector", "#none"), st(stClick, "selector", "#x"))); err != nil || countCalls(f, "click") != 0 {
		t.Fatalf("false while ran body: %v %v", err, f.calls)
	}
}

func TestRpaBreakLoop(t *testing.T) {
	f := newFake()
	f.exists["#stop"] = true
	// break nested in a condition inside the loop, triggered on iteration 3
	cond := withKids(st(stCondition, "kind", "equals", "left", "{{loop_index}}", "right", "3"), st(stBreakLoop))
	after := st(stClick, "selector", "#after")
	vars, _, err := run(t, f,
		withKids(st(stLoop, "count", "10"), st(stClick, "selector", "#body"), cond),
		after)
	if err != nil {
		t.Fatal(err)
	}
	if countCalls(f, "click #body") != 3 || countCalls(f, "click #after") != 1 || vars["loop_index"] != "3" {
		t.Fatalf("break inside loop: calls=%v", f.calls)
	}
	// break outside a loop is an error at run time and in validation
	f = newFake()
	_, _, err = run(t, f, st(stBreakLoop), st(stClick, "selector", "#never"))
	if err == nil || !strings.Contains(err.Error(), "循环") || countCalls(f, "click") != 0 {
		t.Fatalf("break outside loop: err=%v calls=%v", err, f.calls)
	}
	iss := validateWorkflow(Workflow{Name: "x", Steps: []RpaStep{st(stBreakLoop)}})
	if len(iss) != 1 || !strings.Contains(iss[0].Message, "循环") {
		t.Fatalf("validate break outside: %v", iss)
	}
	// break in a condition's else inside a loop is fine
	c := st(stCondition, "kind", "equals", "left", "a", "right", "b")
	c.Else = []RpaStep{st(stBreakLoop)}
	if iss := validateWorkflow(Workflow{Name: "x", Steps: []RpaStep{withKids(st(stLoop, "count", "2"), c)}}); len(iss) != 0 {
		t.Fatalf("validate break in loop/else: %v", iss)
	}
}

func TestRpaExecuteJSSaveAs(t *testing.T) {
	f := newFake()
	f.evalFn = func(code string, vars map[string]string) (string, error) { return "v=" + vars["site"], nil }
	vars, _, err := run(t, f, st(stExecuteJS, "code", "return vars.site", "saveAs", "out"))
	if err != nil || vars["out"] != "v=example.com" {
		t.Fatalf("execute_js saveAs: %v %v", err, vars)
	}
}

func TestRpaContinueOnError(t *testing.T) {
	f := newFake()
	f.fail["click"] = errors.New("元素不存在")
	bad := st(stClick, "selector", "#missing")
	bad.ContinueOnError = true
	_, evs, err := run(t, f, bad, st(stFill, "selector", "#q", "text", "x"))
	if err != nil {
		t.Fatalf("continueOnError should swallow: %v", err)
	}
	if countCalls(f, "fill") != 1 {
		t.Fatalf("next step did not run: %v", f.calls)
	}
	failed := false
	for _, e := range evs {
		if e.Status == "failed" && e.StepID == bad.ID {
			failed = true
		}
	}
	if !failed {
		t.Fatal("failed event not emitted for continued step")
	}
	// without the flag the run stops with the step label in the error
	f = newFake()
	f.fail["click"] = errors.New("元素不存在")
	_, _, err = run(t, f, st(stClick, "selector", "#missing"), st(stFill, "selector", "#q"))
	if err == nil || !strings.Contains(err.Error(), "点击") || countCalls(f, "fill") != 0 {
		t.Fatalf("stop on error: err=%v calls=%v", err, f.calls)
	}
	// disabled steps are skipped
	f = newFake()
	dis := st(stClick, "selector", "#x")
	dis.Disabled = true
	if _, _, err := run(t, f, dis); err != nil || len(f.calls) != 0 {
		t.Fatalf("disabled step ran: %v", f.calls)
	}
}

func TestRpaCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := runRpa(ctx, Workflow{Steps: []RpaStep{st(stDelay, "ms", "5000")}}, newFake(), nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestRpaValidate(t *testing.T) {
	if iss := validateWorkflow(Workflow{}); len(iss) != 2 {
		t.Fatalf("empty workflow issues = %v", iss)
	}
	good := Workflow{Name: "ok", Steps: []RpaStep{
		st(stGoto, "url", "https://a.com"),
		st(stExtractText, "selector", "h1", "var", "t"),
		withKids(st(stLoop, "mode", "while", "kind", "js", "code", "return true"), st(stDelay)),
	}}
	if iss := validateWorkflow(good); len(iss) != 0 {
		t.Fatalf("good workflow issues = %v", iss)
	}
	badGoto := st(stGoto)
	badExtract := st(stExtractText, "selector", "h1")
	unknown := st("teleport")
	disabled := st(stClick)
	disabled.Disabled = true
	nestedBad := withKids(st(stCondition, "selector", "#a"), st(stFill))
	iss := validateWorkflow(Workflow{Name: "x", Steps: []RpaStep{badGoto, badExtract, unknown, disabled, nestedBad}})
	ids := map[string]bool{}
	for _, i := range iss {
		ids[i.StepID] = true
	}
	if len(iss) != 4 || !ids[badGoto.ID] || !ids[badExtract.ID] || !ids[unknown.ID] || ids[disabled.ID] || !ids[nestedBad.Children[0].ID] {
		t.Fatalf("issues = %+v", iss)
	}
}

func TestRpaExportImportRoundTrip(t *testing.T) {
	c := withKids(st(stCondition, "kind", "exists", "selector", "#a"), st(stClick, "selector", "#b"))
	c.Else = []RpaStep{st(stDelay, "ms", "100")}
	wf := Workflow{ID: "orig", Name: "登录流程", Description: "d", Vars: map[string]string{"u": "me"},
		LastStatus: "ok", RunCount: 5, LastRunID: "r1",
		Steps: []RpaStep{st(stGoto, "url", "https://a.com"), withKids(st(stLoop, "count", "2"), c, st(stBreakLoop))}}
	js := exportWorkflowJSON(wf)
	if !strings.Contains(js, `"format": "ghostbrowser-rpa"`) || strings.Contains(js, `"runCount": 5`) {
		t.Fatalf("export envelope wrong:\n%s", js)
	}
	got, err := parseWorkflowJSON(js)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "" || got.Name != wf.Name || got.Vars["u"] != "me" || got.RunCount != 0 {
		t.Fatalf("import meta: %+v", got)
	}
	if countSteps(got.Steps) != countSteps(wf.Steps) || got.Steps[1].Children[0].Else[0].Params["ms"] != "100" ||
		got.Steps[1].Children[0].Children[0].Params["selector"] != "#b" {
		t.Fatalf("import tree: %+v", got.Steps)
	}
	if got.Steps[0].ID == wf.Steps[0].ID || got.Steps[0].ID == "" {
		t.Fatal("import should assign fresh step ids")
	}
	// bare workflow object works too; bad inputs are rejected
	if w, err := parseWorkflowJSON(`{"steps":[{"type":"delay"}]}`); err != nil || w.Name != "导入的流程" || w.Steps[0].Params == nil {
		t.Fatalf("bare import: %+v %v", w, err)
	}
	for _, bad := range []string{"", "not json", `{"steps":[]}`, `{"steps":[{"type":"fly"}]}`} {
		if _, err := parseWorkflowJSON(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestRpaStorePersist(t *testing.T) {
	dir := t.TempDir()
	db := newRpaStore(dir)
	w, err := db.Put(Workflow{Name: "a", Steps: []RpaStep{{Type: stDelay}}})
	if err != nil || w.ID == "" || w.Steps[0].ID == "" {
		t.Fatalf("put: %+v %v", w, err)
	}
	db2 := newRpaStore(dir)
	if got, ok := db2.Get(w.ID); !ok || got.Name != "a" {
		t.Fatalf("reload: %+v %v", got, ok)
	}
	if err := db2.Delete(w.ID); err != nil || len(newRpaStore(dir).List()) != 0 {
		t.Fatalf("delete: %v", err)
	}
}

type fakeCloser struct{ *fakeDriver }

func (fakeCloser) Close() {}

func TestRpaServiceRun(t *testing.T) {
	dir := t.TempDir()
	evs := make(chan RpaEvent, 100)
	svc := &rpaService{
		db: newRpaStore(dir), dir: dir,
		profile: func(id string) (*Profile, bool) { return &Profile{ID: id, Name: "P-" + id}, id != "gone" },
		session: func(id string) (string, bool) { return "ws://x/" + id, id == "open" },
		start:   func(string) (string, error) { return "", errors.New("no") },
		stop:    func(string) {},
		emit:    func(e RpaEvent) { evs <- e },
		newDriver: func(string) (rpaDriverCloser, error) {
			f := newFake()
			f.texts["h1"] = "Title"
			return fakeCloser{f}, nil
		},
		cancels: map[string]context.CancelFunc{},
	}
	w, err := svc.db.Put(Workflow{Name: "w", Steps: []RpaStep{st(stGoto, "url", "a.com"), st(stExtractText, "selector", "h1", "var", "t")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Run(w.ID, nil, RpaRunOptions{}); err == nil {
		t.Fatal("run with no profiles should fail")
	}
	runID, err := svc.Run(w.ID, []string{"open", "closed", "open"}, RpaRunOptions{Concurrency: 2})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for done := false; !done; {
		select {
		case e := <-evs:
			done = e.Phase == "run" && e.Status != "running"
		case <-deadline:
			t.Fatal("run did not finish")
		}
	}
	r, ok := svc.db.Run(runID)
	if !ok || r.Status != "partial" || len(r.Targets) != 2 || r.StepCount != 2 {
		t.Fatalf("run = %+v", r)
	}
	for _, tg := range r.Targets {
		switch tg.ProfileID {
		case "open":
			if tg.Status != "ok" || tg.Vars["t"] != "Title" || len(tg.Steps) != 2 || tg.ProfileName != "P-open" {
				t.Fatalf("open target = %+v", tg)
			}
		case "closed":
			if tg.Status != "failed" || !strings.Contains(tg.Error, "未打开") {
				t.Fatalf("closed target = %+v", tg)
			}
		}
	}
	if got, _ := svc.db.Get(w.ID); got.LastStatus != "partial" || got.RunCount != 1 || got.LastRunID != runID {
		t.Fatalf("workflow stats = %+v", got)
	}
	if runs := newRpaStore(dir).Runs(w.ID); len(runs) != 1 {
		t.Fatalf("persisted runs = %d", len(runs))
	}
}

// The editor keeps the selector field when the user switches scroll mode
// away from "滚动到元素"; only el mode (or no mode) may use it.
func TestRpaScrollModeIgnoresHiddenSelector(t *testing.T) {
	f := newFake()
	_, _, err := run(t, f, st(stScroll, "mode", "by", "selector", "#x", "y", "300"), st(stScroll, "mode", "el", "selector", "#x"), st(stScroll, "selector", "#y"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls, ","); got != "scroll,scrollto #x,scrollto #y" {
		t.Fatalf("calls = %s", got)
	}
	if _, _, err := run(t, newFake(), st(stScroll, "mode", "el")); err == nil || !strings.Contains(err.Error(), "缺少元素选择器") {
		t.Fatalf("el mode without selector: %v", err)
	}
	is := validateWorkflow(Workflow{Name: "x", Steps: []RpaStep{st(stScroll, "mode", "el"),
		st(stExtractText, "selector", "#a", "var", "1abc"), st(stExecuteJS, "code", "1", "saveAs", "价格"), withKids(st(stLoop, "count", "1", "indexVar", "a b"))}})
	msgs := []string{}
	for _, i := range is {
		msgs = append(msgs, i.Message)
	}
	j := strings.Join(msgs, "|")
	if len(is) != 3 || !strings.Contains(j, "缺少元素选择器") || !strings.Contains(j, "「1abc」") || !strings.Contains(j, "「a b」") {
		t.Fatalf("issues = %v", msgs)
	}
}

func TestRpaJSBodyShapes(t *testing.T) {
	if got := jsBody("document.title"); got != `return eval("document.title");` {
		t.Fatalf("expr = %s", got)
	}
	// code with return is tried as an expression first, then as a body
	got := jsBody("xs.map(x => { return x })")
	if !strings.HasPrefix(got, "try { return eval(") || !strings.HasSuffix(got, "\nxs.map(x => { return x })") {
		t.Fatalf("body = %s", got)
	}
}

func TestRpaServiceClaim(t *testing.T) {
	s := &rpaService{}
	if !s.claim("p", "r1") || s.claim("p", "r2") {
		t.Fatal("second claim on a busy profile should fail")
	}
	s.release("p")
	if !s.claim("p", "r3") {
		t.Fatal("claim after release")
	}
}
