package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RPA recorder ("录制"): the user works in a profile's browser and every
// action becomes a workflow step. A capture script runs in an isolated world
// of every tab (recCaptureScript); Go drains its queue by polling
// Runtime.evaluate, like the syncer, so Runtime.enable is never called.
// Navigation is classified from Page.frameRequestedNavigation: a navigation
// the page itself asked for (link, form, script) is a consequence of a
// recorded action and is not recorded as 打开网址; one the browser started
// (typed URL, back/forward) is.

// RecordOptions are chosen when recording starts.
type RecordOptions struct {
	Overlay     bool   `json:"overlay"`     // floating badge in the recorded browser
	Scroll      bool   `json:"scroll"`      // record page scrolling
	PasswordVar bool   `json:"passwordVar"` // default of the review toggle (passwords are never sent to the UI)
	MaxDelayMs  int    `json:"maxDelayMs"`  // cap for recorded think time (default 5000)
	StartURL    string `json:"startUrl"`    // optional page to open first
}

// RecordState is emitted as "rpa:record" and returned by the bindings.
type RecordState struct {
	Active      bool              `json:"active"`
	Paused      bool              `json:"paused"`
	ProfileID   string            `json:"profileId"`
	ProfileName string            `json:"profileName"`
	StartedAt   int64             `json:"startedAt"` // unix ms
	PausedMs    int64             `json:"pausedMs"`  // total paused time so far
	PausedAt    int64             `json:"pausedAt"`  // unix ms, 0 when running
	Steps       []RpaStep         `json:"steps"`
	Vars        map[string]string `json:"vars"`
	Tabs        int               `json:"tabs"`
	Reason      string            `json:"reason,omitempty"` // why it stopped
	StoppedBy   string            `json:"stoppedBy,omitempty"`
	Options     RecordOptions     `json:"options"`
}

// RecordFinalizeOptions are the review screen toggles.
type RecordFinalizeOptions struct {
	KeepDelays  bool `json:"keepDelays"`  // 保留等待时间
	MergeFills  bool `json:"mergeFills"`  // 合并输入
	PasswordVar bool `json:"passwordVar"` // 密码转为变量
}

// RecordFinalized is what the editor merges in.
type RecordFinalized struct {
	Steps []RpaStep         `json:"steps"`
	Vars  map[string]string `json:"vars"`
}

// recEvent is one action from the capture script.
type recEvent struct {
	T     string   `json:"t"`
	Sel   string   `json:"sel,omitempty"`
	Dyn   bool     `json:"dyn,omitempty"`
	V     string   `json:"v,omitempty"`
	PW    bool     `json:"pw,omitempty"`
	Files []string `json:"files,omitempty"`
	X     int      `json:"x,omitempty"`
	Y     int      `json:"y,omitempty"`
	A     string   `json:"a,omitempty"`
	Text  string   `json:"text,omitempty"`
	TS    int64    `json:"ts,omitempty"`
}

// ---------- core: events → steps (no CDP, unit tested) ----------

const (
	recThinkMinMs = 1200
	recRecTag     = "_rec" // marker param: think | password | enter | newtab
)

type recTabInfo struct {
	url    string
	http   bool // has shown an http(s) page
	opener bool // opened by the page (link/window.open)
	down   *recEvent
}

type recCore struct {
	opts     RecordOptions
	steps    []RpaStep
	tabs     []string // tab order as the replay driver numbers them
	info     map[string]*recTabInfo
	cur      string
	lastTS   int64
	secrets  map[string]string // step id → real password
	vars     map[string]string
	nUpload  int
	nExtract int
}

func newRecCore(opts RecordOptions) *recCore {
	if opts.MaxDelayMs <= 0 {
		opts.MaxDelayMs = 5000
	}
	return &recCore{opts: opts, info: map[string]*recTabInfo{}, secrets: map[string]string{}, vars: map[string]string{}, steps: []RpaStep{}}
}

func recStep(t string, kv ...string) RpaStep {
	s := RpaStep{ID: newID(), Type: t, Params: map[string]string{}}
	for i := 0; i+1 < len(kv); i += 2 {
		s.Params[kv[i]] = kv[i+1]
	}
	return s
}

func (c *recCore) tab(id string) *recTabInfo {
	ti := c.info[id]
	if ti == nil {
		ti = &recTabInfo{}
		c.info[id] = ti
	}
	return ti
}

func (c *recCore) tabIndex(id string) int {
	for i, t := range c.tabs {
		if t == id {
			return i
		}
	}
	return -1
}

// addTab registers a tab (existing ones at start, then new ones in creation order).
func (c *recCore) addTab(id, url string, opener bool) {
	if c.tabIndex(id) < 0 {
		c.tabs = append(c.tabs, id)
	}
	ti := c.tab(id)
	ti.opener = ti.opener || opener
	if url != "" {
		ti.url = url
		ti.http = ti.http || isHTTPURL(url)
	}
	if c.cur == "" {
		c.cur = id
	}
}

func isHTTPURL(u string) bool { return strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") }

// lastReal is the index of the last step that is not a recorded pause.
func (c *recCore) lastReal() int {
	for i := len(c.steps) - 1; i >= 0; i-- {
		if c.steps[i].Params[recRecTag] != "think" {
			return i
		}
	}
	return -1
}

// trimThink drops pauses after the last real step (typing continued).
func (c *recCore) trimThink() {
	c.steps = c.steps[:c.lastReal()+1]
}

func (c *recCore) think(ts int64) {
	if c.lastTS > 0 && ts > 0 && len(c.steps) > 0 {
		if gap := ts - c.lastTS; gap >= recThinkMinMs {
			if gap > int64(c.opts.MaxDelayMs) {
				gap = int64(c.opts.MaxDelayMs)
			}
			gap = gap / 100 * 100
			d := recStep(stDelay, "ms", strconv.FormatInt(gap, 10), recRecTag, "think")
			d.Name = "停顿"
			c.steps = append(c.steps, d)
		}
	}
	if ts > 0 {
		c.lastTS = ts
	}
}

// ensureTab switches the workflow to tab before a step that happens there.
func (c *recCore) ensureTab(tab string) {
	if tab == "" || tab == c.cur {
		return
	}
	if c.cur == "" {
		c.cur = tab
		return
	}
	i := c.tabIndex(tab)
	if i < 0 {
		c.tabs = append(c.tabs, tab)
		i = len(c.tabs) - 1
	}
	c.cur = tab
	idx := strconv.Itoa(i + 1)
	if l := len(c.steps) - 1; l >= 0 && c.steps[l].Type == stSelectTab {
		c.steps[l].Params["index"] = idx
		return
	}
	c.steps = append(c.steps, recStep(stSelectTab, "index", idx))
}

func (c *recCore) push(tab string, ts int64, s RpaStep) {
	c.think(ts)
	c.ensureTab(tab)
	c.steps = append(c.steps, s)
}

// coalesce updates the last step when it is of type t on sel in the current tab.
func (c *recCore) coalesce(tab, t, sel string, ts int64, fn func(*RpaStep)) bool {
	i := c.lastReal()
	if i < 0 || tab != c.cur || c.steps[i].Type != t || c.steps[i].Params["selector"] != sel {
		return false
	}
	c.trimThink()
	fn(&c.steps[i])
	if ts > 0 {
		c.lastTS = ts
	}
	return true
}

// event handles one captured action from tab.
func (c *recCore) event(tab string, ev recEvent) {
	ti := c.tab(tab)
	switch ev.T {
	case "down":
		e := ev
		ti.down = &e
	case "click":
		ti.down = nil
		if ev.Dyn {
			if i := c.lastReal(); i < 0 || c.steps[i].Type != stWaitFor || c.steps[i].Params["selector"] != ev.Sel {
				c.push(tab, ev.TS, recStep(stWaitFor, "selector", ev.Sel, "state", "visible", "timeout", "10000"))
			}
		}
		c.push(tab, ev.TS, recStep(stClick, "selector", ev.Sel))
	case "dbl":
		if !c.coalesce(tab, stClick, ev.Sel, ev.TS, func(s *RpaStep) { s.Params["double"] = "true" }) {
			c.push(tab, ev.TS, recStep(stClick, "selector", ev.Sel, "double", "true"))
		}
	case "input", "select":
		text := ev.V
		if ev.PW {
			text = "{{password}}"
		}
		var id string
		if !c.coalesce(tab, stFill, ev.Sel, ev.TS, func(s *RpaStep) { s.Params["text"] = text; id = s.ID }) {
			s := recStep(stFill, "selector", ev.Sel, "text", text)
			id = s.ID
			c.push(tab, ev.TS, s)
		}
		if ev.PW {
			c.secrets[id] = ev.V
			for i := range c.steps {
				if c.steps[i].ID == id {
					c.steps[i].Params[recRecTag] = "password"
				}
			}
			c.vars["password"] = ""
		}
	case "enter":
		s := recStep(stExecuteJS, "code", recEnterJS(ev.Sel), recRecTag, "enter", "selector", ev.Sel)
		s.Name = "按回车键"
		c.push(tab, ev.TS, s)
	case "upload":
		c.nUpload++
		v := "upload_file"
		if c.nUpload > 1 {
			v += strconv.Itoa(c.nUpload)
		}
		s := recStep(stUpload, "selector", ev.Sel, "files", "{{"+v+"}}")
		if len(ev.Files) > 0 {
			s.Name = "上传文件（录制时：" + clip(strings.Join(ev.Files, "、"), 40) + "）"
		}
		c.vars[v] = ""
		c.push(tab, ev.TS, s)
	case "scroll":
		if !c.opts.Scroll {
			return
		}
		x, y := strconv.Itoa(ev.X), strconv.Itoa(ev.Y)
		i := c.lastReal()
		if i >= 0 && tab == c.cur && c.steps[i].Type == stScroll && c.steps[i].Params["mode"] == "to" {
			c.trimThink()
			c.steps[i].Params["x"], c.steps[i].Params["y"] = x, y
			c.lastTS = ev.TS
			return
		}
		c.push(tab, ev.TS, recStep(stScroll, "mode", "to", "x", x, "y", y))
	case "extract":
		c.nExtract++
		c.push(tab, ev.TS, recStep(stExtractText, "selector", ev.Sel, "var", "text"+strconv.Itoa(c.nExtract)))
	case "assert":
		c.push(tab, ev.TS, recStep(stWaitFor, "selector", ev.Sel, "state", "visible", "timeout", "10000"))
	}
}

// navigated handles a main-frame navigation. renderer is true when the page
// itself asked for it (Page.frameRequestedNavigation came first).
func (c *recCore) navigated(tab, url string, renderer bool, reason string, ts int64) {
	ti := c.tab(tab)
	prev := ti.url
	ti.url = url
	if !isHTTPURL(url) {
		return
	}
	first := !ti.http
	ti.http = true
	if renderer {
		// The click that caused it may have died with the old document
		// (only its pointerdown made it out): record the click from that.
		if d := ti.down; d != nil && ts-d.TS < 3000 && (reason == "anchorClick" || strings.HasPrefix(reason, "formSubmission")) {
			c.event(tab, recEvent{T: "click", Sel: d.Sel, Dyn: d.Dyn, TS: d.TS})
		}
		ti.down = nil
		return
	}
	ti.down = nil
	if first && ti.opener {
		return
	}
	if first && c.cur != "" && tab != c.cur {
		// A tab the user opened and typed a URL into: open it from the
		// current tab so the replay gets the same tab numbering.
		s := recStep(stExecuteJS, "code", "window.open("+jsq(url)+", '_blank'); return '已打开新标签页';", recRecTag, "newtab")
		s.Name = "新标签页打开 " + clip(url, 60)
		c.think(ts)
		c.steps = append(c.steps, s)
		return
	}
	if url == prev {
		return // reload
	}
	c.push(tab, ts, recStep(stGoto, "url", url))
}

// tabClosed records closing a tab the workflow knows about.
func (c *recCore) tabClosed(tab string, ts int64) {
	i := c.tabIndex(tab)
	if i < 0 {
		return
	}
	c.think(ts)
	c.steps = append(c.steps, recStep(stCloseTab, "index", strconv.Itoa(i+1)))
	c.tabs = append(c.tabs[:i], c.tabs[i+1:]...)
	delete(c.info, tab)
	if c.cur == tab {
		c.cur = ""
		if len(c.tabs) > 0 {
			c.cur = c.tabs[len(c.tabs)-1] // the driver moves to the last tab
		}
	}
}

func (c *recCore) remove(id string) {
	for i := range c.steps {
		if c.steps[i].ID == id {
			c.steps = append(c.steps[:i], c.steps[i+1:]...)
			delete(c.secrets, id)
			return
		}
	}
}

func (c *recCore) update(s RpaStep) {
	for i := range c.steps {
		if c.steps[i].ID == s.ID {
			if tag := c.steps[i].Params[recRecTag]; tag != "" && s.Params[recRecTag] == "" {
				if s.Params == nil {
					s.Params = map[string]string{}
				}
				s.Params[recRecTag] = tag
			}
			c.steps[i] = s
			return
		}
	}
}

func cloneSteps(in []RpaStep) []RpaStep {
	b, _ := json.Marshal(in)
	out := []RpaStep{}
	_ = json.Unmarshal(b, &out)
	return out
}

// finalizeRecording applies the review toggles and strips recorder markers.
func finalizeRecording(steps []RpaStep, o RecordFinalizeOptions, secrets map[string]string) RecordFinalized {
	in := cloneSteps(steps)
	out := []RpaStep{}
	for _, s := range in {
		if s.Type == stDelay && s.Params[recRecTag] == "think" && !o.KeepDelays {
			continue
		}
		if s.Params[recRecTag] == "password" && !o.PasswordVar {
			if v, ok := secrets[s.ID]; ok {
				s.Params["text"] = v
			}
		}
		out = append(out, s)
	}
	if o.MergeFills {
		// A fill overwritten later by another fill of the same field, with
		// nothing in between that could act on the first value, is dropped.
		barrier := map[string]bool{stClick: true, stGoto: true, stExecuteJS: true, stSelectTab: true, stCloseTab: true, stUpload: true, stCondition: true, stLoop: true}
		keep := make([]bool, len(out))
		for i := range out {
			keep[i] = true
			if out[i].Type != stFill {
				continue
			}
			for j := i + 1; j < len(out); j++ {
				if barrier[out[j].Type] {
					break
				}
				if out[j].Type == stFill && out[j].Params["selector"] == out[i].Params["selector"] {
					keep[i] = false
					break
				}
			}
		}
		merged := []RpaStep{}
		for i, s := range out {
			if keep[i] {
				merged = append(merged, s)
			}
		}
		out = merged
	}
	// Two tab switches in a row: only the last one matters.
	tidy := []RpaStep{}
	for _, s := range out {
		if n := len(tidy); n > 0 && s.Type == stSelectTab && tidy[n-1].Type == stSelectTab {
			tidy[n-1] = s
			continue
		}
		tidy = append(tidy, s)
	}
	vars := map[string]string{}
	for i := range tidy {
		for k, v := range tidy[i].Params {
			for _, m := range tplVar.FindAllStringSubmatch(v, -1) {
				if m[1] == "password" || strings.HasPrefix(m[1], "upload_file") {
					vars[m[1]] = ""
				}
			}
			_ = k
		}
		delete(tidy[i].Params, recRecTag)
		if tidy[i].Type == stExecuteJS {
			delete(tidy[i].Params, "selector")
		}
	}
	return RecordFinalized{Steps: tidy, Vars: vars}
}

// ---------- CDP side ----------

type recTabConn struct {
	id     string
	sid    string
	ctx    int
	reqAt  time.Time
	reason string
	dialog bool
	stall  time.Time
}

type recorder struct {
	mu        sync.Mutex
	core      *recCore
	pid       string
	pname     string
	conn      *cdpConn
	tabs      map[string]*recTabConn // target id
	bySid     map[string]string
	newTabReq time.Time // a page asked for a new tab
	evq       chan cdpMsg
	stopc     chan struct{}
	done      chan struct{}
	stopOnce  sync.Once
	paused    bool
	pausedAt  time.Time
	pausedMs  int64
	started   time.Time
	reason    string
	stoppedBy string
	active    bool
	emit      func(RecordState)
	release   func()
	dirty     bool
}

// recHolder owns the single active recording (one browser at a time).
type recHolderT struct {
	mu   sync.Mutex
	cur  *recorder
	last *recorder // kept after stop for finalize (passwords)
	emit func(RecordState)
}

var recordings = &recHolderT{}

func (h *recHolderT) setEmit(f func(RecordState)) {
	h.mu.Lock()
	h.emit = f
	h.mu.Unlock()
}

func (h *recHolderT) active() *recorder {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.cur
}

func (h *recHolderT) State() RecordState {
	h.mu.Lock()
	r := h.cur
	if r == nil {
		r = h.last
	}
	h.mu.Unlock()
	if r == nil {
		return RecordState{Steps: []RpaStep{}, Vars: map[string]string{}}
	}
	return r.state()
}

// Start attaches to the browser at ws and begins recording.
func (h *recHolderT) Start(pid, pname, ws string, opts RecordOptions, release func()) (RecordState, error) {
	if h.active() != nil {
		return RecordState{}, fmt.Errorf("已经在录制另一个环境，请先停止")
	}
	r := &recorder{core: newRecCore(opts), pid: pid, pname: pname, tabs: map[string]*recTabConn{}, bySid: map[string]string{},
		evq: make(chan cdpMsg, 4096), stopc: make(chan struct{}), done: make(chan struct{}), started: time.Now(), release: release}
	r.emit = func(st RecordState) {
		h.mu.Lock()
		e := h.emit
		h.mu.Unlock()
		if e != nil {
			e(st)
		}
	}
	if err := r.attach(ws); err != nil {
		return RecordState{}, err
	}
	h.mu.Lock()
	h.cur, h.last = r, r
	h.mu.Unlock()
	go r.loop(func() {
		h.mu.Lock()
		if h.cur == r {
			h.cur = nil
		}
		h.mu.Unlock()
	})
	st := r.state()
	r.emit(st)
	return st, nil
}

func (r *recorder) onEvent(m cdpMsg) {
	switch m.Method {
	case "Target.attachedToTarget", "Target.detachedFromTarget", "Page.frameRequestedNavigation", "Page.frameNavigated",
		"Page.javascriptDialogOpening", "Page.javascriptDialogClosed":
		select {
		case r.evq <- m:
		default:
		}
	}
}

func (r *recorder) attach(ws string) error {
	c, err := dialCDP(ws, r.onEvent)
	if err != nil {
		return fmt.Errorf("连接浏览器失败：%v", err)
	}
	r.conn = c
	d := &cdpDriver{conn: c}
	pages, err := d.pages()
	for i := 0; err == nil && i < 15 && len(pages) == 0; i++ {
		time.Sleep(200 * time.Millisecond)
		pages, err = d.pages()
	}
	if err != nil {
		c.c.Close()
		return fmt.Errorf("读取标签页失败：%v", err)
	}
	if len(pages) == 0 {
		raw, err := c.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
		if err != nil {
			c.c.Close()
			return err
		}
		var t struct{ TargetID string }
		_ = json.Unmarshal(raw, &t)
		pages = []string{t.TargetID}
	}
	cur := d.orderExisting(pages)
	for _, id := range d.order {
		r.core.addTab(id, "", false)
	}
	r.core.cur = cur
	_, _ = c.call("", "Target.activateTarget", map[string]any{"targetId": cur})
	// Auto-attach reports existing tabs now and pauses new ones until the
	// capture script is installed (so even a tab's first page is recorded).
	if _, err := c.call("", "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true}); err != nil {
		c.c.Close()
		return fmt.Errorf("连接浏览器失败：%v", err)
	}
	r.active = true
	return nil
}

func (r *recorder) call(sid, method string, params any) (json.RawMessage, error) {
	return r.conn.call(sid, method, params)
}

func (r *recorder) loop(onExit func()) {
	defer close(r.done)
	defer onExit()
	tick := time.NewTicker(60 * time.Millisecond)
	defer tick.Stop()
	startDone := false
	for {
		select {
		case <-r.stopc:
			r.drainAll()
			r.shutdownPages()
			r.finish()
			return
		case <-r.conn.done:
			r.mu.Lock()
			if r.reason == "" {
				r.reason = "浏览器已关闭，录制已结束"
				r.stoppedBy = "browser"
			}
			r.mu.Unlock()
			r.finish()
			return
		case m := <-r.evq:
			r.handle(m)
		case <-tick.C:
			if !startDone && len(r.tabs) > 0 {
				startDone = true
				r.begin()
			}
			r.drainAll()
		}
		if r.takeDirty() {
			r.emit(r.state())
		}
		if r.wantStop() {
			r.drainAll()
			r.shutdownPages()
			r.finish()
			return
		}
	}
}

// begin records where the session starts: the current page (so the
// workflow replays from the same place) or the requested start URL.
func (r *recorder) begin() {
	r.mu.Lock()
	cur := r.core.cur
	ti := r.core.tab(cur)
	r.mu.Unlock()
	if u := strings.TrimSpace(r.core.opts.StartURL); u != "" {
		if !strings.Contains(u, ":") {
			u = "https://" + u
		}
		if tc := r.tabs[cur]; tc != nil {
			_, _ = r.call(tc.sid, "Page.navigate", map[string]any{"url": u})
		}
		return
	}
	if isHTTPURL(ti.url) {
		r.mu.Lock()
		r.core.push(cur, time.Now().UnixMilli(), recStep(stGoto, "url", ti.url))
		r.dirty = true
		r.mu.Unlock()
	}
}

func (r *recorder) takeDirty() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := r.dirty
	r.dirty = false
	return d
}

func (r *recorder) wantStop() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stoppedBy == "overlay"
}

func (r *recorder) handle(m cdpMsg) {
	switch m.Method {
	case "Target.attachedToTarget":
		var p struct {
			SessionID  string `json:"sessionId"`
			TargetInfo struct {
				TargetID string `json:"targetId"`
				Type     string `json:"type"`
				URL      string `json:"url"`
				OpenerID string `json:"openerId"`
			} `json:"targetInfo"`
			Waiting bool `json:"waitingForDebugger"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return
		}
		ti := p.TargetInfo
		if ti.Type != "page" || strings.HasPrefix(ti.URL, "devtools://") {
			if p.Waiting {
				_, _ = r.call(p.SessionID, "Runtime.runIfWaitingForDebugger", nil)
			}
			_, _ = r.call("", "Target.detachFromTarget", map[string]any{"sessionId": p.SessionID})
			return
		}
		r.mu.Lock()
		o := r.core.opts
		paused := r.paused
		n := len(r.core.steps)
		r.mu.Unlock()
		_, _ = r.call(p.SessionID, "Page.enable", nil)
		_, _ = r.call(p.SessionID, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": recCaptureScript(o.Overlay, o.Scroll), "worldName": recWorld, "runImmediately": true})
		if p.Waiting {
			_, _ = r.call(p.SessionID, "Runtime.runIfWaitingForDebugger", nil)
		}
		_ = paused
		_ = n
		r.mu.Lock()
		existing := r.core.tabIndex(ti.TargetID) >= 0
		opener := !existing && (ti.OpenerID != "" || time.Since(r.newTabReq) < 3*time.Second)
		r.core.addTab(ti.TargetID, ti.URL, opener)
		r.tabs[ti.TargetID] = &recTabConn{id: ti.TargetID, sid: p.SessionID}
		r.bySid[p.SessionID] = ti.TargetID
		r.dirty = true
		r.mu.Unlock()
	case "Target.detachedFromTarget":
		var p struct {
			SessionID string `json:"sessionId"`
			TargetID  string `json:"targetId"`
		}
		_ = json.Unmarshal(m.Params, &p)
		tid := r.bySid[p.SessionID]
		if tid == "" {
			return
		}
		delete(r.bySid, p.SessionID)
		delete(r.tabs, tid)
		r.mu.Lock()
		if !r.paused {
			r.core.tabClosed(tid, time.Now().UnixMilli())
		} else if i := r.core.tabIndex(tid); i >= 0 {
			r.core.tabs = append(r.core.tabs[:i], r.core.tabs[i+1:]...)
		}
		r.dirty = true
		r.mu.Unlock()
	case "Page.frameRequestedNavigation":
		var p struct {
			FrameID     string `json:"frameId"`
			Reason      string `json:"reason"`
			Disposition string `json:"disposition"`
		}
		_ = json.Unmarshal(m.Params, &p)
		tc := r.tabs[r.bySid[m.Session]]
		if tc == nil {
			return
		}
		// Grab the click/Enter that caused it before the document unloads.
		r.drain(tc)
		if p.Disposition != "" && p.Disposition != "currentTab" {
			r.newTabReq = time.Now()
			return
		}
		if p.FrameID == tc.id {
			tc.reqAt, tc.reason = time.Now(), p.Reason
		}
	case "Page.frameNavigated":
		var p struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
				URL      string `json:"url"`
				Fragment string `json:"urlFragment"`
			} `json:"frame"`
		}
		_ = json.Unmarshal(m.Params, &p)
		tc := r.tabs[r.bySid[m.Session]]
		if tc == nil || p.Frame.ParentID != "" {
			return
		}
		tc.ctx = 0
		renderer := !tc.reqAt.IsZero() && time.Since(tc.reqAt) < 10*time.Second
		reason := tc.reason
		tc.reqAt, tc.reason = time.Time{}, ""
		r.mu.Lock()
		if r.paused {
			ti := r.core.tab(tc.id)
			ti.url, ti.http = p.Frame.URL+p.Frame.Fragment, ti.http || isHTTPURL(p.Frame.URL)
		} else {
			r.core.navigated(tc.id, p.Frame.URL+p.Frame.Fragment, renderer, reason, time.Now().UnixMilli())
		}
		r.dirty = true
		r.mu.Unlock()
	case "Page.javascriptDialogOpening", "Page.javascriptDialogClosed":
		if tc := r.tabs[r.bySid[m.Session]]; tc != nil {
			tc.dialog = m.Method == "Page.javascriptDialogOpening"
		}
	}
}

func (r *recorder) drainAll() {
	r.mu.Lock()
	ids := append([]string(nil), r.core.tabs...)
	r.mu.Unlock()
	for _, id := range ids {
		if tc := r.tabs[id]; tc != nil {
			r.drain(tc)
		}
	}
}

// recWorldCtx returns the capture world's context in a tab's main frame
// (Chromium reuses the world addScriptToEvaluateOnNewDocument created).
func recWorldCtx(c *cdpConn, sid, frameID string) (int, error) {
	raw, err := c.call(sid, "Page.createIsolatedWorld", map[string]any{"frameId": frameID, "worldName": recWorld})
	if err != nil {
		return 0, err
	}
	var w struct {
		ID int `json:"executionContextId"`
	}
	_ = json.Unmarshal(raw, &w)
	if w.ID == 0 {
		return 0, fmt.Errorf("no context")
	}
	return w.ID, nil
}

func (r *recorder) drain(tc *recTabConn) {
	if tc.dialog || time.Now().Before(tc.stall) {
		return
	}
	if tc.ctx == 0 {
		id, err := recWorldCtx(r.conn, tc.sid, tc.id)
		if err != nil {
			return
		}
		tc.ctx = id
	}
	r.mu.Lock()
	n := countSteps(r.core.steps)
	paused, o := r.paused, r.core.opts
	r.mu.Unlock()
	t0 := time.Now()
	raw, err := r.call(tc.sid, "Runtime.evaluate", map[string]any{"expression": recDrainExpr(n, paused, o.Overlay, o.Scroll), "contextId": tc.ctx, "returnByValue": true})
	if err != nil {
		tc.ctx = 0
		if time.Since(t0) > 5*time.Second {
			tc.stall = time.Now().Add(2 * time.Second)
		}
		return
	}
	var res struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &res) != nil || res.Result.Value == "" {
		return
	}
	var evs []recEvent
	if json.Unmarshal([]byte(res.Result.Value), &evs) != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ev := range evs {
		if ev.T == "ctl" {
			switch ev.A {
			case "pause":
				r.setPaused(true)
			case "resume":
				r.setPaused(false)
			case "stop":
				r.stoppedBy = "overlay"
			}
			r.dirty = true
			continue
		}
		if r.paused {
			continue
		}
		r.core.event(tc.id, ev)
		r.dirty = true
	}
}

// setPaused needs r.mu held.
func (r *recorder) setPaused(p bool) {
	if p == r.paused {
		return
	}
	r.paused = p
	if p {
		r.pausedAt = time.Now()
	} else {
		r.pausedMs += time.Since(r.pausedAt).Milliseconds()
		r.pausedAt = time.Time{}
		r.core.lastTS = 0 // a pause is not think time
		for _, ti := range r.core.info {
			ti.down = nil
		}
	}
}

func (r *recorder) shutdownPages() {
	for _, tc := range r.tabs {
		if tc.ctx != 0 && !tc.dialog {
			_, _ = r.call(tc.sid, "Runtime.evaluate", map[string]any{"expression": recDestroyExpr, "contextId": tc.ctx})
		}
	}
	_, _ = r.call("", "Target.setAutoAttach", map[string]any{"autoAttach": false, "waitForDebuggerOnStart": false, "flatten": true})
	for sid := range r.bySid {
		_, _ = r.call("", "Target.detachFromTarget", map[string]any{"sessionId": sid})
	}
}

// finish closes the connection and releases the profile (loop goroutine).
func (r *recorder) finish() {
	r.mu.Lock()
	r.active = false
	if r.paused {
		r.setPaused(false)
	}
	r.mu.Unlock()
	_ = r.conn.c.Close()
	if r.release != nil {
		r.release()
	}
	r.emit(r.state())
}

// Stop ends the recording and returns the final state.
func (r *recorder) Stop() RecordState {
	r.stopOnce.Do(func() {
		r.mu.Lock()
		if r.stoppedBy == "" {
			r.stoppedBy = "user"
		}
		r.mu.Unlock()
		close(r.stopc)
	})
	select {
	case <-r.done:
	case <-time.After(8 * time.Second):
		_ = r.conn.c.Close() // unblock a hung call; the loop then finishes
		<-r.done
	}
	return r.state()
}

func (r *recorder) Pause(p bool) RecordState {
	r.mu.Lock()
	if r.active {
		r.setPaused(p)
	}
	r.mu.Unlock()
	st := r.state()
	r.emit(st)
	return st
}

func (r *recorder) state() RecordState {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := RecordState{Active: r.active, Paused: r.paused, ProfileID: r.pid, ProfileName: r.pname, StartedAt: r.started.UnixMilli(), PausedMs: r.pausedMs,
		Steps: cloneSteps(r.core.steps), Vars: map[string]string{}, Tabs: len(r.core.tabs), Reason: r.reason, StoppedBy: r.stoppedBy, Options: r.core.opts}
	if r.paused {
		st.PausedAt = r.pausedAt.UnixMilli()
	}
	for k, v := range r.core.vars {
		st.Vars[k] = v
	}
	return st
}

func (r *recorder) edit(fn func(c *recCore)) RecordState {
	r.mu.Lock()
	fn(r.core)
	r.mu.Unlock()
	st := r.state()
	r.emit(st)
	return st
}
