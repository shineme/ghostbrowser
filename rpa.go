package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// RPA automation workflows: an ordered list of steps run against a profile's
// browser over CDP. Condition and loop steps hold nested step lists, so the
// whole workflow is a tree that the editor shows as an indented list.
//
// Variables: extract_text (and execute_js with saveAs) store values; any
// string parameter can reference them as {{name}}. Inside execute_js code a
// `vars` object holds every variable as well.

// Workflow is one saved automation.
type Workflow struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Steps       []RpaStep         `json:"steps"`
	Vars        map[string]string `json:"vars"` // initial variable values
	CreatedAt   int64             `json:"createdAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	LastRunAt   int64             `json:"lastRunAt"`
	LastStatus  string            `json:"lastStatus"` // "", ok, failed, partial, cancelled
	LastRunID   string            `json:"lastRunId"`
	RunCount    int               `json:"runCount"`
}

// RpaStep is one action. Params are plain strings (numbers and booleans are
// parsed by the engine) so the editor and JSON import/export stay simple.
type RpaStep struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Name            string            `json:"name,omitempty"` // optional label shown in the editor
	Disabled        bool              `json:"disabled,omitempty"`
	ContinueOnError bool              `json:"continueOnError,omitempty"`
	Params          map[string]string `json:"params"`
	Children        []RpaStep         `json:"children,omitempty"` // loop body, or condition "then"
	Else            []RpaStep         `json:"else,omitempty"`     // condition "else"
}

// Step types (same set as the reference RPA plugin).
const (
	stGoto        = "goto"
	stClick       = "click"
	stFill        = "fill"
	stWaitFor     = "wait_for"
	stDelay       = "delay"
	stScroll      = "scroll"
	stExtractText = "extract_text"
	stExecuteJS   = "execute_js"
	stScreenshot  = "screenshot"
	stUpload      = "upload"
	stSelectTab   = "select_tab"
	stCloseTab    = "close_tab"
	stCondition   = "condition"
	stLoop        = "loop"
	stBreakLoop   = "break_loop"
	stSwitchFrame = "switch_frame" // 进入框架 / 返回主页面
)

var rpaStepTypes = []string{stGoto, stClick, stFill, stWaitFor, stDelay, stScroll, stExtractText, stExecuteJS, stScreenshot, stUpload, stSelectTab, stCloseTab, stCondition, stLoop, stBreakLoop, stSwitchFrame}

// RpaDriver is what the engine needs from a browser. The CDP implementation
// lives in rpa_run.go; tests inject a fake.
type RpaDriver interface {
	Goto(url string, timeout time.Duration) error
	Click(sel string, double bool) error
	Fill(sel, text string, clear bool) error
	// WaitFor polls until sel reaches state: "visible" (default), "present" or "hidden".
	WaitFor(sel, state string, timeout time.Duration) error
	Exists(sel string) (bool, error)
	ScrollTo(sel string) error
	ScrollBy(x, y int, absolute bool) error
	// Text reads innerText (attr ""), "html", "value" or a named attribute.
	Text(sel, attr string) (string, error)
	// Eval runs a JS function body (or expression); vars is exposed as `vars`.
	// The result is returned as a string (JSON for non-strings).
	Eval(code string, vars map[string]string) (string, error)
	Screenshot(fullPage bool) ([]byte, error)
	Upload(sel string, files []string) error
	SelectTab(index int) error // 1-based
	CloseTab(index int) error  // 1-based, 0 = current
}

// rpaFramer is implemented by drivers that can run element steps inside an
// iframe (rpa_frames.go). path lists iframe selectors from the top document;
// nil means the top document. It waits up to timeout for the frame.
type rpaFramer interface {
	UseFrame(path []string, timeout time.Duration) error
}

// rpaElemShooter captures a single element (screenshot step with selector).
type rpaElemShooter interface {
	ScreenshotElement(sel string) ([]byte, error)
}

// RpaStepEvent reports one step's progress.
type RpaStepEvent struct {
	StepID string            `json:"stepId"`
	Index  int               `json:"index"` // pre-order position in the tree, 0-based
	Type   string            `json:"type"`
	Status string            `json:"status"` // running, ok, failed, skipped
	Log    string            `json:"log"`
	Ms     int64             `json:"ms"`
	Iter   int               `json:"iter,omitempty"` // innermost loop iteration (1-based)
	Vars   map[string]string `json:"vars,omitempty"`
	Shot   string            `json:"shot,omitempty"` // screenshot file
}

type rpaEngine struct {
	d      RpaDriver
	vars   map[string]string
	emit   func(RpaStepEvent)
	saveSh func(stepIndex int, png []byte) (string, error)
	index  map[string]int
	iter   int
	sleep  func(ctx context.Context, d time.Duration) error
	frame  []string // frame context set by switch_frame (nil = top document)
}

// rpaFrameSteps are the step types that act on page elements and accept a
// "frame" param.
var rpaFrameSteps = map[string]bool{stClick: true, stFill: true, stWaitFor: true, stScroll: true, stExtractText: true,
	stExecuteJS: true, stScreenshot: true, stUpload: true, stCondition: true, stLoop: true}

// stepFrame is the frame a step runs in: its own "frame" param (a path from
// the top document) or else the switch_frame context.
func (e *rpaEngine) stepFrame(s *RpaStep) []string {
	if p := parseFramePath(e.p(s, "frame")); len(p) > 0 {
		return p
	}
	return e.frame
}

// enterFrame points the driver at the step's frame before an element step.
func (e *rpaEngine) enterFrame(s *RpaStep, timeout time.Duration) ([]string, error) {
	path := e.stepFrame(s)
	f, ok := e.d.(rpaFramer)
	if !ok {
		if len(path) > 0 {
			return path, fmt.Errorf("当前浏览器不支持在框架里执行步骤")
		}
		return nil, nil
	}
	return path, f.UseFrame(path, timeout)
}

// framePrefix prefixes a log line with the frame it ran in.
func framePrefix(path []string, log string) string {
	if len(path) == 0 {
		return log
	}
	return "⧉ " + frameLabel(path) + " › " + log
}

var errBreakLoop = errors.New("break_loop")

const rpaMaxWhile = 1000

// runRpa executes the workflow and returns the final variables.
func runRpa(ctx context.Context, wf Workflow, d RpaDriver, emit func(RpaStepEvent), saveShot func(int, []byte) (string, error)) (map[string]string, error) {
	e := &rpaEngine{d: d, vars: map[string]string{}, emit: emit, saveSh: saveShot, index: map[string]int{}, sleep: ctxSleep}
	for k, v := range wf.Vars {
		e.vars[k] = v
	}
	if e.emit == nil {
		e.emit = func(RpaStepEvent) {}
	}
	if c, ok := d.(interface{ SetContext(context.Context) }); ok {
		c.SetContext(ctx)
	}
	n := 0
	walkSteps(wf.Steps, func(s *RpaStep) { e.index[s.ID] = n; n++ })
	err := e.runList(ctx, wf.Steps)
	if errors.Is(err, errBreakLoop) {
		err = fmt.Errorf("「跳出循环」只能放在循环里")
	}
	return e.vars, err
}

func ctxSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func walkSteps(list []RpaStep, fn func(*RpaStep)) {
	for i := range list {
		fn(&list[i])
		walkSteps(list[i].Children, fn)
		walkSteps(list[i].Else, fn)
	}
}

func (e *rpaEngine) snapshot() map[string]string {
	out := make(map[string]string, len(e.vars))
	for k, v := range e.vars {
		out[k] = v
	}
	return out
}

func (e *rpaEngine) runList(ctx context.Context, list []RpaStep) error {
	for i := range list {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := e.runStep(ctx, &list[i]); err != nil {
			return err
		}
	}
	return nil
}

func (e *rpaEngine) runStep(ctx context.Context, s *RpaStep) error {
	ev := RpaStepEvent{StepID: s.ID, Index: e.index[s.ID], Type: s.Type, Iter: e.iter}
	if s.Disabled {
		ev.Status, ev.Log = "skipped", "已停用"
		e.emit(ev)
		return nil
	}
	ev.Status = "running"
	e.emit(ev)
	t0 := time.Now()
	log, shot, err := e.exec(ctx, s)
	ev.Ms = time.Since(t0).Milliseconds()
	ev.Iter = e.iter
	if errors.Is(err, errBreakLoop) {
		ev.Status, ev.Log = "ok", "跳出循环"
		e.emit(ev)
		return err
	}
	ev.Vars = e.snapshot()
	ev.Shot = shot
	if err != nil {
		if errors.Is(err, context.Canceled) {
			ev.Status, ev.Log = "failed", "已取消"
			e.emit(ev)
			return err
		}
		ev.Status, ev.Log = "failed", err.Error()
		if s.ContinueOnError {
			ev.Log += "（已设置出错继续）"
			e.emit(ev)
			return nil
		}
		e.emit(ev)
		return fmt.Errorf("%s：%w", stepLabel(s), err)
	}
	ev.Status, ev.Log = "ok", log
	e.emit(ev)
	return nil
}

func stepLabel(s *RpaStep) string {
	if s.Name != "" {
		return s.Name
	}
	return rpaTypeNames[s.Type]
}

var rpaTypeNames = map[string]string{
	stGoto: "打开网址", stClick: "点击", stFill: "输入文本", stWaitFor: "等待元素", stDelay: "延时", stScroll: "滚动",
	stExtractText: "提取文本", stExecuteJS: "执行 JS", stScreenshot: "截图", stUpload: "上传文件", stSelectTab: "切换标签页",
	stCloseTab: "关闭标签页", stCondition: "条件判断", stLoop: "循环", stBreakLoop: "跳出循环", stSwitchFrame: "切换框架",
}

// p returns an interpolated parameter.
func (e *rpaEngine) p(s *RpaStep, k string) string { return interpolate(s.Params[k], e.vars) }

func (e *rpaEngine) pInt(s *RpaStep, k string, def int) int {
	v := strings.TrimSpace(e.p(s, k))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		f, ferr := strconv.ParseFloat(v, 64)
		if ferr != nil {
			return def
		}
		n = int(f)
	}
	return n
}

func (e *rpaEngine) pBool(s *RpaStep, k string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s.Params[k])) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return def
}

func (e *rpaEngine) timeout(s *RpaStep, def int) time.Duration {
	return time.Duration(e.pInt(s, "timeout", def)) * time.Millisecond
}

func need(v, what string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("缺少%s", what)
	}
	return nil
}

func (e *rpaEngine) exec(ctx context.Context, s *RpaStep) (log, shot string, err error) {
	if rpaFrameSteps[s.Type] && s.Type != stCondition && s.Type != stLoop && !(s.Type == stWaitFor && strings.TrimSpace(e.p(s, "selector")) == "") {
		to := time.Duration(0)
		if s.Type == stWaitFor {
			to = e.timeout(s, 10000)
		}
		var path []string
		if path, err = e.enterFrame(s, to); err != nil {
			return framePrefix(path, ""), "", err
		}
		log, shot, err = e.execStep(ctx, s)
		return framePrefix(path, log), shot, err
	}
	switch s.Type {
	case stSelectTab, stCloseTab:
		e.frame = nil // a frame path belongs to the tab it was set in
	case stSwitchFrame:
		next := switchFramePath(e.frame, e.p(s, "frame"))
		if len(next) > 0 {
			f, ok := e.d.(rpaFramer)
			if !ok {
				return "", "", fmt.Errorf("当前浏览器不支持框架")
			}
			if err = f.UseFrame(next, e.timeout(s, 10000)); err != nil {
				return "", "", err
			}
		}
		e.frame = next
		if len(next) == 0 {
			return "返回主页面", "", nil
		}
		return "进入框架 " + frameLabel(next), "", nil
	}
	return e.execStep(ctx, s)
}

func (e *rpaEngine) execStep(ctx context.Context, s *RpaStep) (log, shot string, err error) {
	d := e.d
	switch s.Type {
	case stGoto:
		u := strings.TrimSpace(e.p(s, "url"))
		if err = need(u, "网址"); err != nil {
			return
		}
		if !strings.Contains(u, ":") {
			u = "https://" + u
		}
		return u, "", d.Goto(u, e.timeout(s, 30000))
	case stClick:
		sel := e.p(s, "selector")
		if err = need(sel, "选择器"); err != nil {
			return
		}
		return sel, "", d.Click(sel, e.pBool(s, "double", false))
	case stFill:
		sel := e.p(s, "selector")
		if err = need(sel, "选择器"); err != nil {
			return
		}
		text := e.p(s, "text")
		return fmt.Sprintf("%s ← %s", sel, clip(text, 60)), "", d.Fill(sel, text, e.pBool(s, "clear", true))
	case stWaitFor:
		sel := e.p(s, "selector")
		to := e.timeout(s, 10000)
		if strings.TrimSpace(sel) == "" {
			return fmt.Sprintf("等待 %d ms", to.Milliseconds()), "", e.sleep(ctx, to)
		}
		state := s.Params["state"]
		word := map[string]string{"hidden": "已消失", "present": "已出现"}[state]
		if word == "" {
			word = "已可见"
		}
		return sel + " " + word, "", d.WaitFor(sel, state, to)
	case stDelay:
		ms := e.pInt(s, "ms", 1000)
		if mx := e.pInt(s, "maxMs", 0); mx > ms {
			ms += int(time.Now().UnixNano() % int64(mx-ms+1))
		}
		return fmt.Sprintf("%d ms", ms), "", e.sleep(ctx, time.Duration(ms)*time.Millisecond)
	case stScroll:
		// mode: by | to | el. The editor keeps a hidden selector when the user
		// switches away from "el", so the selector only counts in el mode (or
		// with no mode at all, for hand-written JSON).
		mode := s.Params["mode"]
		if sel := e.p(s, "selector"); (mode == "el" || mode == "") && strings.TrimSpace(sel) != "" {
			return "滚动到 " + sel, "", d.ScrollTo(sel)
		} else if mode == "el" {
			return "", "", need(sel, "元素选择器")
		}
		x, y := e.pInt(s, "x", 0), e.pInt(s, "y", 600)
		abs := mode == "to"
		return fmt.Sprintf("%s (%d, %d)", map[bool]string{true: "滚动到", false: "滚动"}[abs], x, y), "", d.ScrollBy(x, y, abs)
	case stExtractText:
		sel, name := e.p(s, "selector"), strings.TrimSpace(s.Params["var"])
		if err = need(sel, "选择器"); err != nil {
			return
		}
		if err = need(name, "变量名"); err != nil {
			return
		}
		var v string
		if v, err = d.Text(sel, strings.TrimSpace(s.Params["attr"])); err != nil {
			return
		}
		if e.pBool(s, "trim", true) {
			v = strings.TrimSpace(v)
		}
		e.vars[name] = v
		return fmt.Sprintf("%s = %s", name, clip(v, 80)), "", nil
	case stExecuteJS:
		code := s.Params["code"] // not interpolated: use the vars object in code
		if err = need(code, "代码"); err != nil {
			return
		}
		var v string
		if v, err = d.Eval(code, e.snapshot()); err != nil {
			return
		}
		if name := strings.TrimSpace(s.Params["saveAs"]); name != "" {
			e.vars[name] = v
			return fmt.Sprintf("%s = %s", name, clip(v, 80)), "", nil
		}
		return "返回 " + clip(v, 80), "", nil
	case stScreenshot:
		var png []byte
		if sel := strings.TrimSpace(e.p(s, "selector")); sel != "" {
			es, ok := d.(rpaElemShooter)
			if !ok {
				return "", "", fmt.Errorf("当前浏览器不支持元素截图")
			}
			if png, err = es.ScreenshotElement(sel); err != nil {
				return
			}
		} else if png, err = d.Screenshot(e.pBool(s, "fullPage", false)); err != nil {
			return
		}
		if e.saveSh != nil {
			if shot, err = e.saveSh(e.index[s.ID], png); err != nil {
				return
			}
		}
		if name := strings.TrimSpace(s.Params["saveAs"]); name != "" && shot != "" {
			e.vars[name] = shot
		}
		return fmt.Sprintf("%d KB", (len(png)+1023)/1024), shot, nil
	case stUpload:
		sel := e.p(s, "selector")
		if err = need(sel, "选择器"); err != nil {
			return
		}
		files := splitFiles(e.p(s, "files"))
		if len(files) == 0 {
			return "", "", fmt.Errorf("缺少文件路径")
		}
		return strings.Join(files, "、"), "", d.Upload(sel, files)
	case stSelectTab:
		i := e.pInt(s, "index", 0)
		if i < 1 {
			return "", "", fmt.Errorf("标签页序号必须从 1 开始")
		}
		return fmt.Sprintf("第 %d 个标签页", i), "", d.SelectTab(i)
	case stCloseTab:
		i := e.pInt(s, "index", 0)
		return map[bool]string{true: "当前标签页", false: fmt.Sprintf("第 %d 个标签页", i)}[i <= 0], "", d.CloseTab(i)
	case stCondition:
		var ok bool
		if ok, err = e.cond(s); err != nil {
			return
		}
		if ok {
			err = e.runList(ctx, s.Children)
			return "条件成立", "", err
		}
		err = e.runList(ctx, s.Else)
		return "条件不成立", "", err
	case stLoop:
		return e.loop(ctx, s)
	case stBreakLoop:
		return "", "", errBreakLoop
	}
	return "", "", fmt.Errorf("未知步骤类型 %q", s.Type)
}

// cond evaluates a condition/while test: kind exists | not_exists | js | equals | contains.
func (e *rpaEngine) cond(s *RpaStep) (bool, error) {
	switch s.Params["kind"] {
	case "", "exists", "not_exists":
		sel := e.p(s, "selector")
		if err := need(sel, "选择器"); err != nil {
			return false, err
		}
		ok := false
		path, err := e.enterFrame(s, time.Millisecond)
		if err == nil {
			ok, err = e.d.Exists(sel)
		} else if len(path) > 0 && !fatal(err) {
			err = nil // the frame is not there, so neither is the element
		}
		if s.Params["kind"] == "not_exists" {
			ok = !ok
		}
		return ok, err
	case "js":
		code := s.Params["code"]
		if err := need(code, "判断代码"); err != nil {
			return false, err
		}
		if _, err := e.enterFrame(s, 0); err != nil {
			return false, err
		}
		v, err := e.d.Eval("const __r = await (async () => {\n"+jsBody(code)+"\n})();\nreturn !!__r;", e.snapshot())
		if err != nil {
			return false, err
		}
		return jsTruthy(v), nil
	case "equals":
		return e.p(s, "left") == e.p(s, "right"), nil
	case "contains":
		return strings.Contains(e.p(s, "left"), e.p(s, "right")), nil
	}
	return false, fmt.Errorf("未知条件 %q", s.Params["kind"])
}

// jsTruthy applies JS truthiness to a value returned as string/JSON.
func jsTruthy(v string) bool {
	switch strings.TrimSpace(v) {
	case "", "false", "0", "-0", "null", "undefined", "NaN", `""`:
		return false
	}
	return true
}

func (e *rpaEngine) loop(ctx context.Context, s *RpaStep) (string, string, error) {
	mode := s.Params["mode"]
	idxVar := strings.TrimSpace(s.Params["indexVar"])
	if idxVar == "" {
		idxVar = "loop_index"
	}
	outer := e.iter
	defer func() { e.iter = outer }()
	max := rpaMaxWhile
	count := 0
	if mode == "" || mode == "count" {
		count = e.pInt(s, "count", 1)
		if count < 0 {
			count = 0
		}
		max = count
	} else if m := e.pInt(s, "max", rpaMaxWhile); m > 0 {
		max = m
	}
	done := 0
	for i := 1; i <= max; i++ {
		if mode == "while" {
			ok, err := e.cond(s)
			if err != nil {
				return "", "", err
			}
			if !ok {
				break
			}
		}
		e.iter = i
		e.vars[idxVar] = strconv.Itoa(i)
		err := e.runList(ctx, s.Children)
		done = i
		if errors.Is(err, errBreakLoop) {
			return fmt.Sprintf("第 %d 轮跳出", i), "", nil
		}
		if err != nil {
			return "", "", err
		}
	}
	if mode == "while" && done == max {
		return fmt.Sprintf("已循环 %d 轮（达到上限）", done), "", nil
	}
	return fmt.Sprintf("已循环 %d 轮", done), "", nil
}

var varName = regexp.MustCompile(`^[A-Za-z_\p{Han}][\w\p{Han}.-]*$`)

var tplVar = regexp.MustCompile(`\{\{\s*([A-Za-z_\p{Han}][\w\p{Han}.-]*)\s*\}\}`)

// interpolate replaces {{name}} with the variable's value. Unknown names are
// left untouched so a typo stays visible in the log.
func interpolate(s string, vars map[string]string) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	return tplVar.ReplaceAllStringFunc(s, func(m string) string {
		name := tplVar.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
}

func splitFiles(s string) []string {
	out := []string{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == ';' || r == '|' }) {
		if f = strings.Trim(strings.TrimSpace(f), `"`); f != "" {
			out = append(out, f)
		}
	}
	return out
}

func clip(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

// ---------- validation / normalisation ----------

// RpaIssue is a problem found in a workflow before running it.
type RpaIssue struct {
	StepID  string `json:"stepId"`
	Message string `json:"message"`
}

func validateWorkflow(wf Workflow) []RpaIssue {
	issues := []RpaIssue{}
	if strings.TrimSpace(wf.Name) == "" {
		issues = append(issues, RpaIssue{Message: "流程名称不能为空"})
	}
	if len(wf.Steps) == 0 {
		issues = append(issues, RpaIssue{Message: "流程里还没有步骤"})
	}
	var walk func(list []RpaStep, inLoop bool)
	walk = func(list []RpaStep, inLoop bool) {
		for _, s := range list {
			add := func(m string) { issues = append(issues, RpaIssue{StepID: s.ID, Message: m}) }
			req := func(k, what string) {
				if strings.TrimSpace(s.Params[k]) == "" {
					add(stepLabel(&s) + "：缺少" + what)
				}
			}
			if _, ok := rpaTypeNames[s.Type]; !ok {
				add(fmt.Sprintf("未知步骤类型 %q", s.Type))
				continue
			}
			if s.Disabled {
				continue
			}
			// Variable names must be usable as {{name}} later on.
			for _, k := range []string{"var", "saveAs", "indexVar"} {
				if n := strings.TrimSpace(s.Params[k]); n != "" && !varName.MatchString(n) {
					add(fmt.Sprintf("%s：变量名「%s」要以字母、下划线或汉字开头", stepLabel(&s), n))
				}
			}
			switch s.Type {
			case stGoto:
				req("url", "网址")
			case stClick, stFill:
				req("selector", "选择器")
			case stScroll:
				if s.Params["mode"] == "el" {
					req("selector", "元素选择器")
				}
			case stExtractText:
				req("selector", "选择器")
				req("var", "变量名")
			case stExecuteJS:
				req("code", "代码")
			case stUpload:
				req("selector", "选择器")
				req("files", "文件路径")
			case stSelectTab:
				req("index", "标签页序号")
			case stCondition:
				switch s.Params["kind"] {
				case "js":
					req("code", "判断代码")
				case "equals", "contains":
					req("left", "比较值")
				default:
					req("selector", "选择器")
				}
			case stLoop:
				if s.Params["mode"] == "while" {
					switch s.Params["kind"] {
					case "js":
						req("code", "判断代码")
					case "equals", "contains":
						req("left", "比较值")
					default:
						req("selector", "选择器")
					}
				}
			case stBreakLoop:
				if !inLoop {
					add("「跳出循环」必须放在循环里")
				}
			}
			walk(s.Children, inLoop || s.Type == stLoop)
			walk(s.Else, inLoop)
		}
	}
	walk(wf.Steps, false)
	return issues
}

// normaliseSteps fills IDs and params maps and drops children on leaf steps.
func normaliseSteps(list []RpaStep, fresh bool) []RpaStep {
	out := make([]RpaStep, 0, len(list))
	for _, s := range list {
		if s.ID == "" || fresh {
			s.ID = newID()
		}
		s.Type = strings.TrimSpace(s.Type)
		if s.Params == nil {
			s.Params = map[string]string{}
		}
		if s.Type == stLoop || s.Type == stCondition {
			s.Children = normaliseSteps(s.Children, fresh)
		} else {
			s.Children = nil
		}
		if s.Type == stCondition {
			s.Else = normaliseSteps(s.Else, fresh)
		} else {
			s.Else = nil
		}
		out = append(out, s)
	}
	return out
}

func countSteps(list []RpaStep) int {
	n := 0
	walkSteps(list, func(*RpaStep) { n++ })
	return n
}

// ---------- import / export ----------

type rpaExport struct {
	Format   string   `json:"format"`
	Version  int      `json:"version"`
	Workflow Workflow `json:"workflow"`
}

const rpaFormat = "ghostbrowser-rpa"

func exportWorkflowJSON(wf Workflow) string {
	wf.LastRunAt, wf.LastStatus, wf.LastRunID, wf.RunCount = 0, "", "", 0
	b, _ := json.MarshalIndent(rpaExport{Format: rpaFormat, Version: 1, Workflow: wf}, "", "  ")
	return string(b)
}

// parseWorkflowJSON accepts our export envelope or a bare workflow object.
func parseWorkflowJSON(raw string) (Workflow, error) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
	if raw == "" {
		return Workflow{}, fmt.Errorf("JSON 为空")
	}
	var env struct {
		Format   string          `json:"format"`
		Workflow json.RawMessage `json:"workflow"`
	}
	var wf Workflow
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return Workflow{}, fmt.Errorf("不是有效的 JSON：%v", err)
	}
	body := []byte(raw)
	if len(env.Workflow) > 0 {
		body = env.Workflow
	}
	if err := json.Unmarshal(body, &wf); err != nil {
		return Workflow{}, fmt.Errorf("流程格式不对：%v", err)
	}
	if len(wf.Steps) == 0 {
		return Workflow{}, fmt.Errorf("JSON 里没有步骤（steps）")
	}
	bad := ""
	walkSteps(wf.Steps, func(s *RpaStep) {
		if _, ok := rpaTypeNames[s.Type]; !ok && bad == "" {
			bad = s.Type
		}
	})
	if bad != "" {
		return Workflow{}, fmt.Errorf("不支持的步骤类型 %q", bad)
	}
	if strings.TrimSpace(wf.Name) == "" {
		wf.Name = "导入的流程"
	}
	wf.Steps = normaliseSteps(wf.Steps, true)
	wf.ID, wf.LastRunAt, wf.LastStatus, wf.LastRunID, wf.RunCount = "", 0, "", "", 0
	return wf, nil
}
