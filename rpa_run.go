package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ---------- run records ----------

// RpaRun is one execution of a workflow over one or more profiles.
type RpaRun struct {
	ID           string       `json:"id"`
	WorkflowID   string       `json:"workflowId"`
	WorkflowName string       `json:"workflowName"`
	Status       string       `json:"status"` // running, ok, failed, partial, cancelled
	StartedAt    int64        `json:"startedAt"`
	EndedAt      int64        `json:"endedAt"`
	Concurrency  int          `json:"concurrency"`
	StepCount    int          `json:"stepCount"`
	Targets      []*RpaTarget `json:"targets"`
}

// RpaTarget is the result for one profile inside a run.
type RpaTarget struct {
	ProfileID   string            `json:"profileId"`
	ProfileName string            `json:"profileName"`
	Status      string            `json:"status"` // pending, starting, running, ok, failed, cancelled
	Error       string            `json:"error"`
	StartedAt   int64             `json:"startedAt"`
	EndedAt     int64             `json:"endedAt"`
	AutoStarted bool              `json:"autoStarted"`
	Steps       []RpaStepEvent    `json:"steps"` // finished step executions (loops add one per iteration)
	Vars        map[string]string `json:"vars"`
}

// RpaRunOptions controls a multi-profile run.
type RpaRunOptions struct {
	Concurrency int  `json:"concurrency"` // profiles run in parallel (1 = one after another)
	AutoStart   bool `json:"autoStart"`   // start profiles whose browser is not open
	CloseAfter  bool `json:"closeAfter"`  // close browsers the run started itself
}

// RpaEvent is emitted as "rpa:event".
type RpaEvent struct {
	RunID     string            `json:"runId"`
	ProfileID string            `json:"profileId"`
	Phase     string            `json:"phase"` // step, target, run
	Status    string            `json:"status"`
	StepID    string            `json:"stepId,omitempty"`
	Index     int               `json:"index"`
	Type      string            `json:"type,omitempty"`
	Log       string            `json:"log,omitempty"`
	Ms        int64             `json:"ms,omitempty"`
	Iter      int               `json:"iter,omitempty"`
	Vars      map[string]string `json:"vars,omitempty"`
	Shot      string            `json:"shot,omitempty"`
	Error     string            `json:"error,omitempty"`
}

const rpaMaxStepRecords = 3000

// ---------- service ----------

type rpaDriverCloser interface {
	RpaDriver
	Close()
}

type rpaService struct {
	db        *rpaStore
	dir       string
	profile   func(id string) (*Profile, bool)
	session   func(id string) (string, bool) // open browser's CDP websocket
	start     func(id string) (string, error)
	stop      func(id string)
	emit      func(RpaEvent)
	newDriver func(ws string) (rpaDriverCloser, error)

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
	busy    map[string]string // profile id → run id driving its browser
}

// claim marks a profile as driven by runID; two runs on one browser would
// fight over the same tab.
func (s *rpaService) claim(pid, runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.busy == nil {
		s.busy = map[string]string{}
	}
	if _, ok := s.busy[pid]; ok {
		return false
	}
	s.busy[pid] = runID
	return true
}

func (s *rpaService) release(pid string) {
	s.mu.Lock()
	delete(s.busy, pid)
	s.mu.Unlock()
}

func (s *rpaService) Run(wfID string, profileIDs []string, opt RpaRunOptions) (string, error) {
	wf, ok := s.db.Get(wfID)
	if !ok {
		return "", fmt.Errorf("流程不存在或已删除")
	}
	if issues := validateWorkflow(wf); len(issues) > 0 {
		return "", fmt.Errorf("流程有 %d 处需要修改：%s", len(issues), issues[0].Message)
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range profileIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("请至少选择一个环境")
	}
	if opt.Concurrency < 1 {
		opt.Concurrency = 1
	}
	if opt.Concurrency > 10 {
		opt.Concurrency = 10
	}
	run := &RpaRun{ID: newID(), WorkflowID: wf.ID, WorkflowName: wf.Name, Status: "running", StartedAt: time.Now().Unix(), Concurrency: opt.Concurrency, StepCount: countSteps(wf.Steps)}
	for _, id := range ids {
		name := id
		if p, ok := s.profile(id); ok {
			name = p.Name
		}
		run.Targets = append(run.Targets, &RpaTarget{ProfileID: id, ProfileName: name, Status: "pending", Steps: []RpaStepEvent{}, Vars: map[string]string{}})
	}
	s.db.addRun(run)
	s.db.markRun(wf.ID, run.ID, "running", run.StartedAt)
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancels[run.ID] = cancel
	s.mu.Unlock()
	s.emit(RpaEvent{RunID: run.ID, Phase: "run", Status: "running"})

	go func() {
		defer cancel()
		sem := make(chan struct{}, opt.Concurrency)
		var wg sync.WaitGroup
		for _, id := range ids {
			wg.Add(1)
			sem <- struct{}{}
			go func(id string) {
				defer wg.Done()
				defer func() { <-sem }()
				s.runTarget(ctx, run.ID, wf, id, opt)
			}(id)
		}
		wg.Wait()
		final := ""
		s.db.updateRun(run.ID, true, func(r *RpaRun) {
			okN, failN, cancN := 0, 0, 0
			for _, t := range r.Targets {
				switch t.Status {
				case "ok":
					okN++
				case "cancelled":
					cancN++
				default:
					failN++
				}
			}
			switch {
			case ctx.Err() != nil && cancN > 0:
				r.Status = "cancelled"
			case failN == 0 && cancN == 0:
				r.Status = "ok"
			case okN == 0:
				r.Status = "failed"
			default:
				r.Status = "partial"
			}
			r.EndedAt = time.Now().Unix()
			final = r.Status
		})
		s.db.markRun(wf.ID, run.ID, final, run.StartedAt)
		s.mu.Lock()
		delete(s.cancels, run.ID)
		s.mu.Unlock()
		s.emit(RpaEvent{RunID: run.ID, Phase: "run", Status: final})
	}()
	return run.ID, nil
}

func (s *rpaService) Cancel(runID string) bool {
	s.mu.Lock()
	c, ok := s.cancels[runID]
	s.mu.Unlock()
	if ok {
		c()
	}
	return ok
}

func (s *rpaService) setTarget(runID, pid string, persist bool, fn func(*RpaTarget)) {
	s.db.updateRun(runID, persist, func(r *RpaRun) {
		for _, t := range r.Targets {
			if t.ProfileID == pid {
				fn(t)
			}
		}
	})
}

func (s *rpaService) runTarget(ctx context.Context, runID string, wf Workflow, pid string, opt RpaRunOptions) {
	finish := func(status, msg string, vars map[string]string) {
		s.setTarget(runID, pid, true, func(t *RpaTarget) {
			t.Status, t.Error, t.EndedAt = status, msg, time.Now().Unix()
			if vars != nil {
				t.Vars = vars
			}
		})
		s.emit(RpaEvent{RunID: runID, ProfileID: pid, Phase: "target", Status: status, Error: msg, Vars: vars})
	}
	if ctx.Err() != nil {
		finish("cancelled", "已取消", nil)
		return
	}
	if _, ok := s.profile(pid); !ok {
		finish("failed", "环境不存在或已删除", nil)
		return
	}
	if !s.claim(pid, runID) {
		finish("failed", "这个环境正在运行另一个自动化流程，请等它结束或先停止", nil)
		return
	}
	defer s.release(pid)
	ws, open := s.session(pid)
	started := false
	if !open {
		if !opt.AutoStart {
			finish("failed", "浏览器未打开：请先启动这个环境，或勾选「自动启动未运行的环境」", nil)
			return
		}
		s.setTarget(runID, pid, false, func(t *RpaTarget) { t.Status, t.StartedAt, t.AutoStarted = "starting", time.Now().Unix(), true })
		s.emit(RpaEvent{RunID: runID, ProfileID: pid, Phase: "target", Status: "starting"})
		var err error
		if ws, err = s.start(pid); err != nil {
			finish("failed", "启动失败："+err.Error(), nil)
			return
		}
		started = true
	}
	if started && opt.CloseAfter {
		defer s.stop(pid)
	}
	if !strings.HasPrefix(ws, "ws") {
		finish("failed", "拿不到浏览器的调试地址（CDP）", nil)
		return
	}
	drv, err := s.newDriver(ws)
	if err != nil {
		finish("failed", "连接浏览器失败："+err.Error(), nil)
		return
	}
	defer drv.Close()
	s.setTarget(runID, pid, false, func(t *RpaTarget) {
		t.Status = "running"
		if t.StartedAt == 0 {
			t.StartedAt = time.Now().Unix()
		}
	})
	s.emit(RpaEvent{RunID: runID, ProfileID: pid, Phase: "target", Status: "running"})

	shotDir := filepath.Join(s.dir, "rpa", "shots", runID)
	saveShot := func(idx int, png []byte) (string, error) {
		if err := os.MkdirAll(shotDir, 0o755); err != nil {
			return "", err
		}
		p := filepath.Join(shotDir, fmt.Sprintf("%s-%02d-%d.png", pid, idx+1, time.Now().UnixMilli()))
		return p, os.WriteFile(p, png, 0o644)
	}
	emitStep := func(ev RpaStepEvent) {
		if ev.Status != "running" {
			s.setTarget(runID, pid, false, func(t *RpaTarget) {
				if len(t.Steps) < rpaMaxStepRecords {
					t.Steps = append(t.Steps, ev)
				}
				if ev.Vars != nil {
					t.Vars = ev.Vars
				}
			})
		}
		s.emit(RpaEvent{RunID: runID, ProfileID: pid, Phase: "step", Status: ev.Status, StepID: ev.StepID, Index: ev.Index, Type: ev.Type, Log: ev.Log, Ms: ev.Ms, Iter: ev.Iter, Vars: ev.Vars, Shot: ev.Shot})
	}
	vars, err := runRpa(ctx, wf, drv, emitStep, saveShot)
	switch {
	case ctx.Err() != nil:
		finish("cancelled", "已取消", vars)
	case err != nil:
		finish("failed", err.Error(), vars)
	default:
		finish("ok", "", vars)
	}
}

// shotDataURL returns a screenshot saved by a run as a data: URL, only for
// files inside the RPA screenshot folder.
func (s *rpaService) shotDataURL(path string) (string, error) {
	root, _ := filepath.Abs(filepath.Join(s.dir, "rpa", "shots"))
	abs, err := filepath.Abs(path)
	if err != nil || !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return "", fmt.Errorf("不是自动化截图")
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b), nil
}

// ---------- CDP driver ----------

// cdpDriver drives one page of an open browser through its own CDP
// connection (separate from the fingerprint emulator's), attached with
// flatten so every call goes to the page session.
type cdpDriver struct {
	conn     *cdpConn
	sid      string
	target   string
	mu       sync.Mutex
	order    []string // page targets, oldest first
	implicit time.Duration
	ctx      context.Context // the run's context (SetContext); nil = never cancelled
	unwatch  func() bool
}

// errBrowserGone is returned once the browser (or our CDP connection) is gone.
var errBrowserGone = errors.New("浏览器已关闭或连接已断开")

// errTabGone is returned when the tab the driver is attached to was closed.
var errTabGone = errors.New("当前标签页已关闭，请先用「切换标签页」选择一个标签页")

// SetContext ties the driver to a run: cancelling ctx closes the CDP
// connection so an in-flight wait or navigation returns at once instead of
// running to its own timeout.
func (d *cdpDriver) SetContext(ctx context.Context) {
	if d.unwatch != nil {
		d.unwatch()
	}
	d.ctx = ctx
	d.unwatch = context.AfterFunc(ctx, func() { _ = d.conn.c.Close() })
}

func (d *cdpDriver) gone() bool {
	select {
	case <-d.conn.done:
		return true
	default:
		return false
	}
}

// wrapErr turns transport errors into the run's cancellation or a Chinese message.
func (d *cdpDriver) wrapErr(method string, err error) error {
	if err == nil {
		return nil
	}
	if d.ctx != nil && d.ctx.Err() != nil {
		return d.ctx.Err()
	}
	msg := err.Error()
	switch {
	case d.gone() || strings.Contains(msg, "browser closed") || strings.Contains(msg, "use of closed network connection"):
		return errBrowserGone
	case strings.HasSuffix(msg, ": timeout"):
		return fmt.Errorf("浏览器 15 秒内没有响应（%s）", method)
	case strings.Contains(msg, "Session with given id not found") || strings.Contains(msg, "No target with given id"):
		// The page sessions go first when the whole browser shuts down, so
		// give the connection a moment to drop before blaming the tab.
		select {
		case <-d.conn.done:
			return errBrowserGone
		case <-time.After(time.Second):
			return errTabGone
		}
	}
	return err
}

// fatal reports errors that make polling pointless.
func fatal(err error) bool {
	return errors.Is(err, errBrowserGone) || errors.Is(err, errTabGone) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// rootCall sends a browser-level (session-less) command.
func (d *cdpDriver) rootCall(method string, params any) (json.RawMessage, error) {
	r, err := d.conn.call("", method, params)
	return r, d.wrapErr(method, err)
}

func newCDPDriver(ws string) (rpaDriverCloser, error) {
	d := &cdpDriver{implicit: 8 * time.Second}
	conn, err := dialCDP(ws, d.onEvent)
	if err != nil {
		return nil, err
	}
	d.conn = conn
	pages, err := d.pages()
	if err != nil {
		conn.c.Close()
		return nil, err
	}
	// A browser that just opened its debugging port can list no page for a
	// moment (Chromium 110–120 register the first tab late); wait for it
	// instead of opening a second, empty tab.
	for i := 0; i < 15 && len(pages) == 0; i++ {
		time.Sleep(200 * time.Millisecond)
		if pages, err = d.pages(); err != nil {
			conn.c.Close()
			return nil, err
		}
	}
	if len(pages) == 0 {
		r, err := conn.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
		if err != nil {
			conn.c.Close()
			return nil, err
		}
		var t struct{ TargetID string }
		_ = json.Unmarshal(r, &t)
		pages = []string{t.TargetID}
	}
	// Target.getTargets returns pages in no useful order (verified against
	// Chromium 155: it is neither creation nor activation order), so number
	// the tabs that are already open by when their document loaded and
	// attach to the one the user is looking at. Tabs opened later are
	// appended in creation order from Target.targetCreated events.
	cur := d.orderExisting(pages)
	_, _ = conn.call("", "Target.setDiscoverTargets", map[string]any{"discover": true})
	if err := d.attach(cur); err != nil {
		conn.c.Close()
		return nil, err
	}
	return d, nil
}

// orderExisting fills d.order with pages sorted by performance.timeOrigin
// (oldest first) and returns the tab to drive: the focused one, else a
// visible one, else the newest. Pages that cannot be probed keep their
// getTargets position after the probed ones.
func (d *cdpDriver) orderExisting(pages []string) string {
	type info struct {
		id      string
		origin  float64
		visible bool
		focus   bool
		ok      bool
	}
	infos := make([]info, len(pages))
	for i, id := range pages {
		infos[i] = info{id: id}
		r, err := d.conn.call("", "Target.attachToTarget", map[string]any{"targetId": id, "flatten": true})
		if err != nil {
			continue
		}
		var at struct{ SessionID string }
		_ = json.Unmarshal(r, &at)
		ev, err := d.conn.call(at.SessionID, "Runtime.evaluate", map[string]any{"expression": "[performance.timeOrigin, document.visibilityState === 'visible', document.hasFocus()]", "returnByValue": true, "timeout": 2000})
		_, _ = d.conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": at.SessionID})
		if err != nil {
			continue
		}
		var v struct {
			Result struct {
				Value []json.RawMessage `json:"value"`
			} `json:"result"`
		}
		if json.Unmarshal(ev, &v) != nil || len(v.Result.Value) != 3 {
			continue
		}
		_ = json.Unmarshal(v.Result.Value[0], &infos[i].origin)
		_ = json.Unmarshal(v.Result.Value[1], &infos[i].visible)
		_ = json.Unmarshal(v.Result.Value[2], &infos[i].focus)
		infos[i].ok = true
	}
	sort.SliceStable(infos, func(a, b int) bool {
		if infos[a].ok != infos[b].ok {
			return infos[a].ok
		}
		return infos[a].ok && infos[a].origin < infos[b].origin
	})
	d.order = d.order[:0]
	for _, in := range infos {
		d.order = append(d.order, in.id)
	}
	pick := -1
	for i := len(infos) - 1; i >= 0 && pick < 0; i-- {
		if infos[i].focus {
			pick = i
		}
	}
	for i := len(infos) - 1; i >= 0 && pick < 0; i-- {
		if infos[i].visible {
			pick = i
		}
	}
	if pick < 0 {
		for i := len(infos) - 1; i >= 0; i-- {
			if infos[i].ok {
				pick = i
				break
			}
		}
	}
	if pick < 0 {
		pick = len(infos) - 1
	}
	return infos[pick].id
}

func (d *cdpDriver) Close() {
	if d.unwatch != nil {
		d.unwatch()
	}
	if d.sid != "" && !d.gone() {
		_, _ = d.conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": d.sid})
	}
	d.conn.c.Close()
}

func (d *cdpDriver) onEvent(m cdpMsg) {
	var ev struct {
		TargetInfo struct {
			TargetID, Type, URL string
		} `json:"targetInfo"`
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(m.Params, &ev)
	d.mu.Lock()
	defer d.mu.Unlock()
	switch m.Method {
	case "Target.targetCreated":
		if ev.TargetInfo.Type == "page" && !strings.HasPrefix(ev.TargetInfo.URL, "devtools://") {
			for _, id := range d.order {
				if id == ev.TargetInfo.TargetID {
					return
				}
			}
			d.order = append(d.order, ev.TargetInfo.TargetID)
		}
	case "Target.targetDestroyed":
		for i, id := range d.order {
			if id == ev.TargetID {
				d.order = append(d.order[:i], d.order[i+1:]...)
				break
			}
		}
	}
}

func (d *cdpDriver) pages() ([]string, error) {
	raw, err := d.rootCall("Target.getTargets", nil)
	if err != nil {
		return nil, err
	}
	var tl struct {
		TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	out := []string{}
	for _, t := range tl.TargetInfos {
		if t.Type == "page" && !strings.HasPrefix(t.URL, "devtools://") {
			out = append(out, t.TargetID)
		}
	}
	return out, nil
}

func (d *cdpDriver) attach(target string) error {
	r, err := d.rootCall("Target.attachToTarget", map[string]any{"targetId": target, "flatten": true})
	if err != nil {
		return err
	}
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	if d.sid != "" {
		_, _ = d.conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": d.sid})
	}
	d.sid, d.target = at.SessionID, target
	_, _ = d.conn.call("", "Target.activateTarget", map[string]any{"targetId": target})
	return nil
}

func (d *cdpDriver) call(method string, params any) (json.RawMessage, error) {
	r, err := d.conn.call(d.sid, method, params)
	return r, d.wrapErr(method, err)
}

// eval evaluates an expression in the page and decodes its value into out.
func (d *cdpDriver) eval(expr string, out any) error {
	raw, err := d.call("Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true, "userGesture": true})
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exc *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if r.Exc != nil {
		msg := r.Exc.Exception.Description
		if msg == "" {
			msg = r.Exc.Text
		}
		msg = strings.TrimPrefix(strings.SplitN(msg, "\n", 2)[0], "Error: ")
		return fmt.Errorf("%s", msg)
	}
	if out != nil && len(r.Result.Value) > 0 {
		return json.Unmarshal(r.Result.Value, out)
	}
	return nil
}

func jsq(s string) string { b, _ := json.Marshal(s); return string(b) }

// rpaFind resolves a selector: CSS by default, "xpath:" or a leading "//" for
// XPath, "text:" for the smallest element whose visible text contains it.
// "a || b" tries each alternative in turn (the recorder stores a fallback
// XPath this way).
const rpaFind = `const __gbq = (sel) => {
  sel = String(sel).trim();
  if (sel.includes(' || ')) {
    for (const p of sel.split(' || ')) { let e = null; try { e = __gbq(p); } catch (err) {} if (e) return e; }
    return null;
  }
  if (sel.startsWith('xpath:') || sel.startsWith('//') || sel.startsWith('(//')) {
    return document.evaluate(sel.replace(/^xpath:/, ''), document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
  }
  if (sel.startsWith('text:')) {
    const t = sel.slice(5).trim(); let best = null, bl = Infinity;
    for (const el of document.body ? document.body.querySelectorAll('*') : []) {
      const s = (el.innerText || '').trim();
      if (s && s.includes(t) && s.length < bl) { best = el; bl = s.length; }
    }
    return best;
  }
  return document.querySelector(sel);
};
` + rpaSettle + `const __gbvis = (el) => { if (!el) return false; const st = getComputedStyle(el); if (st.visibility === 'hidden' || st.display === 'none') return false; const r = el.getBoundingClientRect(); return r.width > 0 || r.height > 0; };
`

// onEl runs body with `el` bound to the selector's element (error if missing).
func onEl(sel, body string) string {
	return "(async () => {" + rpaFind + "const el = __gbq(" + jsq(sel) + "); if (!el) throw new Error('找不到元素：' + " + jsq(sel) + ");" + body + "})()"
}

// waitPresent gives the page a moment to render the element before acting.
func (d *cdpDriver) waitPresent(sel string) error {
	return d.WaitFor(sel, "present", d.implicit)
}

func (d *cdpDriver) Goto(url string, timeout time.Duration) error {
	raw, err := d.call("Page.navigate", map[string]any{"url": url})
	if err != nil {
		return err
	}
	var nav struct {
		ErrorText string `json:"errorText"`
	}
	_ = json.Unmarshal(raw, &nav)
	if nav.ErrorText != "" {
		return fmt.Errorf("打开失败：%s", nav.ErrorText)
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var st string
		err := d.eval(`document.readyState`, &st)
		if err == nil && st == "complete" {
			return nil
		}
		if fatal(err) {
			return err
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("页面 %d 秒内没有加载完成", int(timeout.Seconds()))
}

func (d *cdpDriver) Click(sel string, double bool) error {
	if err := d.waitPresent(sel); err != nil {
		return err
	}
	var pt struct {
		X, Y float64
		OK   bool
	}
	if err := d.eval(onEl(sel, `el.scrollIntoView({block: 'center', inline: 'center'}); await new Promise(r => setTimeout(r, 120));
		const r = el.getBoundingClientRect(); const x = r.left + r.width / 2, y = r.top + r.height / 2;
		const hit = document.elementFromPoint(x, y);
		return { X: x, Y: y, OK: r.width > 0 && r.height > 0 && !!hit && (hit === el || el.contains(hit) || hit.contains(el)) };`), &pt); err != nil {
		return err
	}
	if !pt.OK {
		// Covered or zero-size: fall back to a DOM click.
		return d.eval(onEl(sel, `el.click(); if (`+fmt.Sprint(double)+`) el.dispatchEvent(new MouseEvent('dblclick', {bubbles: true}));`), nil)
	}
	clicks := 1
	if double {
		clicks = 2
	}
	_, _ = d.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": pt.X, "y": pt.Y})
	for c := 1; c <= clicks; c++ {
		for _, t := range []string{"mousePressed", "mouseReleased"} {
			if _, err := d.call("Input.dispatchMouseEvent", map[string]any{"type": t, "x": pt.X, "y": pt.Y, "button": "left", "buttons": 1, "clickCount": c}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *cdpDriver) Fill(sel, text string, clear bool) error {
	if err := d.waitPresent(sel); err != nil {
		return err
	}
	var kind string
	if err := d.eval(onEl(sel, `el.scrollIntoView({block: 'center'}); el.focus();
		if (el.tagName === 'SELECT') {
			const v = `+jsq(text)+`; const o = [...el.options].find(o => o.value === v || o.text.trim() === v);
			if (!o) throw new Error('下拉框里没有选项：' + v);
			el.value = o.value; el.dispatchEvent(new Event('input', {bubbles: true})); el.dispatchEvent(new Event('change', {bubbles: true}));
			return 'select';
		}
		if (`+fmt.Sprint(clear)+`) {
			if ('value' in el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA')) {
				const proto = el.tagName === 'INPUT' ? HTMLInputElement.prototype : HTMLTextAreaElement.prototype;
				Object.getOwnPropertyDescriptor(proto, 'value').set.call(el, '');
				el.dispatchEvent(new Event('input', {bubbles: true}));
			} else if (el.isContentEditable) {
				document.execCommand('selectAll', false); document.execCommand('delete', false);
			}
		} else if (el.setSelectionRange && typeof el.value === 'string') { try { el.setSelectionRange(el.value.length, el.value.length); } catch (e) {} }
		return 'text';`), &kind); err != nil {
		return err
	}
	if kind == "select" || text == "" {
		return nil
	}
	if _, err := d.call("Input.insertText", map[string]any{"text": text}); err != nil {
		return err
	}
	return d.eval(onEl(sel, `el.dispatchEvent(new Event('change', {bubbles: true}));`), nil)
}

func (d *cdpDriver) WaitFor(sel, state string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	expr := "(() => {" + rpaFind + "const el = __gbq(" + jsq(sel) + "); return [!!el, __gbvis(el)]; })()"
	for {
		var st [2]bool
		err := d.eval(expr, &st)
		if err == nil {
			switch state {
			case "present":
				if st[0] {
					return nil
				}
			case "hidden":
				if !st[1] {
					return nil
				}
			default:
				if st[1] {
					return nil
				}
			}
		} else if fatal(err) {
			return err
		} else if strings.Contains(err.Error(), "SyntaxError") || strings.Contains(err.Error(), "is not a valid") {
			return fmt.Errorf("选择器写法有误：%s", sel)
		}
		if time.Now().After(deadline) {
			word := map[string]string{"present": "出现", "hidden": "消失"}[state]
			if word == "" {
				word = "可见"
			}
			return fmt.Errorf("等待 %s %s超时（%.0f 秒）", sel, word, timeout.Seconds())
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func (d *cdpDriver) Exists(sel string) (bool, error) {
	var ok bool
	err := d.eval("(() => {"+rpaFind+"return !!__gbq("+jsq(sel)+"); })()", &ok)
	return ok, err
}

func (d *cdpDriver) ScrollTo(sel string) error {
	if err := d.waitPresent(sel); err != nil {
		return err
	}
	return d.eval(onEl(sel, `el.scrollIntoView({behavior: 'smooth', block: 'center'}); await __gbsettle();`), nil)
}

// rpaSettle waits (up to 3 s) until no scroll event has fired for 160 ms, so the next step sees the final position of
// a smooth scroll. Scroll events from any scroller restart the quiet period.
const rpaSettle = `const __gbsettle = () => new Promise(res => {
  let last = performance.now(); const t0 = last;
  const on = () => { last = performance.now(); };
  addEventListener('scroll', on, {capture: true, passive: true});
  const tick = () => {
    const now = performance.now();
    if (now - last > 160 || now - t0 > 3000) { removeEventListener('scroll', on, {capture: true}); res(); } else setTimeout(tick, 40);
  };
  setTimeout(tick, 40);
});
`

func (d *cdpDriver) ScrollBy(x, y int, absolute bool) error {
	fn := "scrollBy"
	if absolute {
		fn = "scrollTo"
	}
	return d.eval(fmt.Sprintf(`(async () => { %s window.%s({left: %d, top: %d, behavior: 'smooth'}); await __gbsettle(); })()`, rpaSettle, fn, x, y), nil)
}

func (d *cdpDriver) Text(sel, attr string) (string, error) {
	if err := d.waitPresent(sel); err != nil {
		return "", err
	}
	var v string
	err := d.eval(onEl(sel, `const a = `+jsq(attr)+`;
		if (a === '' || a === 'text') return el.innerText ?? el.textContent ?? '';
		if (a === 'html') return el.innerHTML;
		if (a === 'value') return String(el.value ?? '');
		return el.getAttribute(a) ?? '';`), &v)
	return v, err
}

var jsReturn = regexp.MustCompile(`\breturn\b`)

// jsBody turns user code into an async function body. Code without a
// return keyword is evaluated and its last value returned. Code that does
// contain "return" may still be a plain expression whose return sits in a
// nested function (arr.map(x => { return x.id })), so it is tried as an
// expression first; only when that cannot parse because of a top-level return
// or await is it used as the function body. A parse error happens before
// anything runs, so the code never executes twice.
func jsBody(code string) string {
	if !jsReturn.MatchString(code) {
		return "return eval(" + jsq(code) + ");"
	}
	return "try { return eval(" + jsq(code) + "); } catch (__e) { if (!(__e instanceof SyntaxError && /Illegal return|await is only valid|Unexpected reserved word/.test(__e.message))) throw __e; }\n" + code
}

func (d *cdpDriver) Eval(code string, vars map[string]string) (string, error) {
	vb, _ := json.Marshal(vars)
	expr := "(async () => { const vars = " + string(vb) + ";\n" + jsBody(code) + "\n})().then(v => typeof v === 'string' ? v : v === undefined ? '' : JSON.stringify(v))"
	var out string
	err := d.eval(expr, &out)
	return out, err
}

func (d *cdpDriver) Screenshot(fullPage bool) ([]byte, error) {
	params := map[string]any{"format": "png"}
	if fullPage {
		raw, err := d.call("Page.getLayoutMetrics", nil)
		if err == nil {
			var m struct {
				CSS struct{ Width, Height float64 } `json:"cssContentSize"`
			}
			_ = json.Unmarshal(raw, &m)
			if m.CSS.Width > 0 && m.CSS.Height > 0 {
				params["captureBeyondViewport"] = true
				params["clip"] = map[string]any{"x": 0, "y": 0, "width": m.CSS.Width, "height": m.CSS.Height, "scale": 1}
			}
		}
	}
	raw, err := d.call("Page.captureScreenshot", params)
	if err != nil {
		return nil, err
	}
	var r struct{ Data string }
	_ = json.Unmarshal(raw, &r)
	return base64.StdEncoding.DecodeString(r.Data)
}

func (d *cdpDriver) Upload(sel string, files []string) error {
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("文件不存在：%s", f)
		}
	}
	if err := d.waitPresent(sel); err != nil {
		return err
	}
	raw, err := d.call("Runtime.evaluate", map[string]any{"expression": "(() => {" + rpaFind + "return __gbq(" + jsq(sel) + "); })()"})
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			ObjectID string `json:"objectId"`
			Subtype  string `json:"subtype"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &r)
	if r.Result.ObjectID == "" || r.Result.Subtype == "null" {
		return fmt.Errorf("找不到元素：%s", sel)
	}
	_, err = d.call("DOM.setFileInputFiles", map[string]any{"files": files, "objectId": r.Result.ObjectID})
	return err
}

func (d *cdpDriver) tabs() []string {
	// Merge in tabs the event stream has not reported yet (and drop closed ones).
	live, err := d.pages()
	d.mu.Lock()
	defer d.mu.Unlock()
	if err == nil {
		alive := map[string]bool{}
		for _, id := range live {
			alive[id] = true
		}
		kept := []string{}
		known := map[string]bool{}
		for _, id := range d.order {
			if alive[id] {
				kept = append(kept, id)
				known[id] = true
			}
		}
		for _, id := range live {
			if !known[id] {
				kept = append(kept, id)
			}
		}
		d.order = kept
	}
	return append([]string(nil), d.order...)
}

func (d *cdpDriver) SelectTab(index int) error {
	tabs := d.tabs()
	if index < 1 || index > len(tabs) {
		return fmt.Errorf("只有 %d 个标签页，没有第 %d 个", len(tabs), index)
	}
	return d.attach(tabs[index-1])
}

func (d *cdpDriver) CloseTab(index int) error {
	tabs := d.tabs()
	target := d.target
	if index > 0 {
		if index > len(tabs) {
			return fmt.Errorf("只有 %d 个标签页，没有第 %d 个", len(tabs), index)
		}
		target = tabs[index-1]
	}
	if len(tabs) <= 1 {
		return fmt.Errorf("这是最后一个标签页，关闭会退出浏览器")
	}
	if _, err := d.rootCall("Target.closeTarget", map[string]any{"targetId": target}); err != nil {
		return err
	}
	if target == d.target {
		d.sid = "" // the session died with the tab
		for i := len(tabs) - 1; i >= 0; i-- {
			if tabs[i] != target {
				return d.attach(tabs[i])
			}
		}
	}
	return nil
}
