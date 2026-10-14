package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Group control ("同步器"): what the user does in the master browser's
// current tab is replayed in every follower's tab.
//
// Capture: a small script runs in an isolated world of the master tab
// (Page.addScriptToEvaluateOnNewDocument with worldName, so the page's own
// scripts never see it) and queues DOM events there; Go drains the queue
// every ~50 ms with Runtime.evaluate. Runtime.enable is never called on the
// master (Runtime.addBinding would need it to deliver bindingCalled), so the
// page cannot detect a debugger through it. Navigation of the master
// is watched with Page.frameNavigated / navigatedWithinDocument.
//
// Replay: each follower has its own queue and goroutine replaying events
// with Input.dispatchMouseEvent / dispatchKeyEvent / insertText, and
// Page.navigate when its URL differs from the master's. Selectors are
// resolved in an isolated world of the follower too.

// SyncFollowerState is one follower in SyncerState.
type SyncFollowerState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Events int64  `json:"events"`
	Error  string `json:"error,omitempty"`
}

// SyncerState is the syncer status emitted as "syncer:event".
type SyncerState struct {
	Running    bool                `json:"running"`
	Master     string              `json:"master"`
	MasterName string              `json:"masterName"`
	Followers  []SyncFollowerState `json:"followers"`
	Events     int64               `json:"events"`
	LastEvent  string              `json:"lastEvent"` // click|scroll|key|text|select|nav
	StartedAt  int64               `json:"startedAt"`
	Reason     string              `json:"reason,omitempty"` // why it stopped / last problem
}

// syncEvent is one captured action (shared shape from the capture script).
type syncEvent struct {
	T     string  `json:"t"`
	Sel   string  `json:"sel,omitempty"`
	RX    float64 `json:"rx,omitempty"`
	RY    float64 `json:"ry,omitempty"`
	VX    float64 `json:"vx,omitempty"`
	VY    float64 `json:"vy,omitempty"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	DX    float64 `json:"dx,omitempty"`
	DY    float64 `json:"dy,omitempty"`
	Btn   int     `json:"btn,omitempty"`
	N     int     `json:"n,omitempty"`
	Key   string  `json:"key,omitempty"`
	Code  string  `json:"code,omitempty"`
	KC    int     `json:"kc,omitempty"`
	Mods  int     `json:"mods,omitempty"`
	S     string  `json:"s,omitempty"`
	Value string  `json:"value,omitempty"`
	URL   string  `json:"url,omitempty"`
}

const syncWorld = "__gb_sync"

// syncCaptureScript runs in the master's isolated world. The DOM is shared
// with the page, so capture-phase listeners see every user action.
func syncCaptureScript() string {
	return `(() => {
  if (window.top !== window || window.__gbSyncOn) return;
  window.__gbSyncOn = true;
  window.__gbq = [];
  // The poller stamps __gbLast on every drain; once it stops (syncer
  // stopped) capture goes quiet instead of queueing actions for a restart.
  const send = (o) => { const q = window.__gbq, l = window.__gbLast; if ((!l || Date.now() - l < 3000) && q.length < 2000) q.push(o); };
  const esc = (s) => (window.CSS && CSS.escape) ? CSS.escape(s) : s.replace(/[^\w-]/g, '\\$&');
  const sel = (el) => {
    if (!(el instanceof Element)) return '';
    const parts = [];
    for (let e = el; e && e.nodeType === 1 && e !== document.documentElement; e = e.parentElement) {
      if (e.id && document.querySelectorAll('#' + esc(e.id)).length === 1) { parts.unshift('#' + esc(e.id)); break; }
      let p = e.tagName.toLowerCase();
      const par = e.parentElement;
      if (par) {
        const same = Array.from(par.children).filter(c => c.tagName === e.tagName);
        if (same.length > 1) p += ':nth-of-type(' + (same.indexOf(e) + 1) + ')';
      }
      parts.unshift(p);
    }
    return parts.join(' > ');
  };
  const mods = (e) => (e.altKey ? 1 : 0) | (e.ctrlKey ? 2 : 0) | (e.metaKey ? 4 : 0) | (e.shiftKey ? 8 : 0);
  const onPtr = (e) => {
    if (!e.isTrusted) return;
    const t = e.target instanceof Element ? e.target : null;
    const r = t ? t.getBoundingClientRect() : null;
    send({t: 'click', sel: sel(t), rx: r && r.width ? (e.clientX - r.left) / r.width : 0.5, ry: r && r.height ? (e.clientY - r.top) / r.height : 0.5,
      vx: e.clientX / innerWidth, vy: e.clientY / innerHeight, x: e.clientX, y: e.clientY, btn: e.button, n: e.detail || 1});
  };
  addEventListener('click', onPtr, true);
  // A right click fires both auxclick and contextmenu; replay it once.
  addEventListener('auxclick', (e) => { if (e.button !== 2) onPtr(e); }, true);
  addEventListener('contextmenu', onPtr, true);
  addEventListener('wheel', (e) => {
    if (!e.isTrusted) return;
    const k = e.deltaMode === 1 ? 40 : e.deltaMode === 2 ? innerHeight : 1;
    send({t: 'scroll', dx: e.deltaX * k, dy: e.deltaY * k, vx: e.clientX / innerWidth, vy: e.clientY / innerHeight});
  }, {capture: true, passive: true});
  let composing = false;
  addEventListener('compositionstart', () => { composing = true; }, true);
  addEventListener('compositionend', (e) => { composing = false; if (e.data) send({t: 'text', s: e.data}); }, true);
  addEventListener('keydown', (e) => {
    if (!e.isTrusted || composing || e.isComposing || e.keyCode === 229) return;
    const m = mods(e);
    // Printable keys carry key/code so followers get real keydown/keypress
    // (page shortcuts, space-to-scroll), not just inserted text.
    if (e.key.length === 1 && !(m & 7)) { send({t: 'text', s: e.key, key: e.key, code: e.code, kc: e.keyCode, mods: m}); return; }
    if (['Shift', 'Control', 'Alt', 'Meta', 'CapsLock'].includes(e.key)) return;
    // Ctrl/Cmd+V and Shift+Insert are replayed through the paste event below;
    // replaying the shortcut too would paste twice (the clipboard is shared).
    if (((m & 6) && e.code === 'KeyV') || ((m & 8) && e.key === 'Insert')) return;
    send({t: 'key', key: e.key, code: e.code, kc: e.keyCode, mods: m});
  }, true);
  addEventListener('paste', (e) => {
    if (!e.isTrusted) return;
    const s = e.clipboardData && e.clipboardData.getData('text/plain');
    if (s) send({t: 'text', s});
  }, true);
  addEventListener('change', (e) => {
    const t = e.target;
    if (t && t.tagName === 'SELECT') send({t: 'select', sel: sel(t), value: t.value});
  }, true);
})();`
}

// syncDrainExpr installs the capture script if this world doesn't have it yet
// and returns (and clears) the queued events as a JSON string.
func syncDrainExpr() string {
	return `(() => { ` + syncCaptureScript() + ` const q = window.__gbq || []; window.__gbq = []; window.__gbLast = Date.now(); return q.length ? JSON.stringify(q) : ''; })()`
}

func jsStr(s string) string { b, _ := json.Marshal(s); return string(b) }

type syncFollower struct {
	id, name string
	conn     *cdpConn
	sid      string
	tid      string
	q        chan syncEvent
	events   int64
	ok       bool
	err      string
	ctx      int // isolated world execution context, 0 = none yet
}

type syncer struct {
	mu        sync.Mutex
	running   bool
	master    string
	mname     string
	mconn     *cdpConn
	msid      string
	mctx      int // master isolated-world context the poller drains
	followers []*syncFollower
	events    int64
	last      string
	started   int64
	reason    string
	stop      chan struct{}
	stopOnce  sync.Once
	emit      func(SyncerState)
	lastEmit  time.Time
	emitTimer *time.Timer
}

var groupSync = &syncerHolder{}

// syncerHolder owns the single active syncer and the emit callback.
type syncerHolder struct {
	mu   sync.Mutex
	cur  *syncer
	emit func(SyncerState)
	prev SyncerState
}

func (h *syncerHolder) State() SyncerState {
	h.mu.Lock()
	s := h.cur
	prev := h.prev
	h.mu.Unlock()
	if s == nil {
		if prev.Followers == nil {
			prev.Followers = []SyncFollowerState{}
		}
		return prev
	}
	return s.state()
}

func (h *syncerHolder) Stop(reason string) {
	h.mu.Lock()
	s := h.cur
	h.mu.Unlock()
	if s != nil {
		s.shutdown(reason)
	}
}

// Start begins syncing master's current tab to followers' current tabs.
// wsOf returns the browser websocket of a running profile, name its display name.
func (h *syncerHolder) Start(master string, followers []string, wsOf func(string) (string, bool), name func(string) string) (SyncerState, error) {
	h.Stop("restart")
	mws, ok := wsOf(master)
	if !ok {
		return SyncerState{}, fmt.Errorf("主控环境没有在运行")
	}
	s := &syncer{master: master, mname: name(master), stop: make(chan struct{}), started: time.Now().UnixMilli(), followers: []*syncFollower{}}
	h.mu.Lock()
	s.emit = func(st SyncerState) {
		h.mu.Lock()
		if !st.Running {
			h.prev = st
			if h.cur == s {
				h.cur = nil
			}
		}
		e := h.emit
		h.mu.Unlock()
		if e != nil {
			e(st)
		}
	}
	h.mu.Unlock()
	seen := map[string]bool{master: true}
	var ferrs []string
	for _, id := range followers {
		if seen[id] {
			continue
		}
		seen[id] = true
		ws, ok := wsOf(id)
		if !ok {
			ferrs = append(ferrs, name(id)+"：没有在运行")
			continue
		}
		f, err := attachFollower(id, name(id), ws)
		if err != nil {
			ferrs = append(ferrs, name(id)+"："+err.Error())
			continue
		}
		s.followers = append(s.followers, f)
	}
	if len(s.followers) == 0 {
		msg := "没有可用的被控环境"
		if len(ferrs) > 0 {
			msg += "（" + strings.Join(ferrs, "；") + "）"
		}
		return SyncerState{}, fmt.Errorf("%s", msg)
	}
	if err := s.attachMaster(mws); err != nil {
		for _, f := range s.followers {
			f.conn.c.Close()
		}
		return SyncerState{}, fmt.Errorf("连接主控环境失败：%v", err)
	}
	if len(ferrs) > 0 {
		s.reason = "部分环境未加入：" + strings.Join(ferrs, "；")
	}
	s.running = true
	h.mu.Lock()
	h.cur = s
	h.mu.Unlock()
	for _, f := range s.followers {
		go s.replay(f)
	}
	st := s.state()
	s.emit(st)
	return st, nil
}

// firstPageTarget returns the first normal page tab of a browser.
func firstPageTarget(c *cdpConn) (string, string, error) {
	raw, err := c.call("", "Target.getTargets", nil)
	if err != nil {
		return "", "", err
	}
	var tl struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	for _, t := range tl.TargetInfos {
		if t.Type == "page" && !strings.HasPrefix(t.URL, "devtools://") {
			return t.TargetID, t.URL, nil
		}
	}
	return "", "", fmt.Errorf("浏览器没有打开的标签页")
}

func attachPage(c *cdpConn) (tid, sid string, err error) {
	tid, _, err = firstPageTarget(c)
	if err != nil {
		return
	}
	raw, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": tid, "flatten": true})
	if err != nil {
		return
	}
	var at struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &at)
	if at.SessionID == "" {
		err = fmt.Errorf("无法连接标签页")
	}
	return tid, at.SessionID, err
}

func attachFollower(id, name, ws string) (*syncFollower, error) {
	c, err := dialCDP(ws, nil)
	if err != nil {
		return nil, err
	}
	tid, sid, err := attachPage(c)
	if err != nil {
		c.c.Close()
		return nil, err
	}
	return &syncFollower{id: id, name: name, conn: c, sid: sid, tid: tid, q: make(chan syncEvent, 512), ok: true}, nil
}

func (s *syncer) attachMaster(ws string) error {
	c, err := dialCDP(ws, s.onMasterEvent)
	if err != nil {
		return err
	}
	_, sid, err := attachPage(c)
	if err != nil {
		c.c.Close()
		return err
	}
	s.mconn, s.msid = c, sid
	// Runtime bindings only report calls while the Runtime domain is enabled,
	// and Runtime.enable is detectable by pages, so events are queued inside
	// the isolated world and drained by polling with Runtime.evaluate (which
	// works without Runtime.enable). The new-document script makes capture
	// start before page scripts run on every navigation.
	for _, st := range []struct {
		m string
		p map[string]any
	}{
		{"Page.enable", nil},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": syncCaptureScript(), "worldName": syncWorld, "runImmediately": true}},
	} {
		if _, err := c.call(sid, st.m, st.p); err != nil {
			c.c.Close()
			return err
		}
	}
	// Prime the capture world before Start returns: whatever a previous
	// syncer left queued there is discarded, not replayed.
	if id, err := worldContext(c, sid); err == nil {
		if _, err := c.call(sid, "Runtime.evaluate", map[string]any{"expression": syncDrainExpr(), "contextId": id, "returnByValue": true}); err == nil {
			s.mctx = id
		}
	}
	go s.poll()
	go func() {
		select {
		case <-c.done:
			s.shutdown("主控浏览器已关闭")
		case <-s.stop:
		}
	}()
	return nil
}

// worldContext creates (or reuses, when Chromium keeps one per name) the sync
// isolated world in the main frame of a tab and returns its context id.
func worldContext(c *cdpConn, sid string) (int, error) {
	raw, err := c.call(sid, "Page.getFrameTree", nil)
	if err != nil {
		return 0, err
	}
	var ft struct {
		FrameTree struct {
			Frame struct {
				ID string `json:"id"`
			} `json:"frame"`
		} `json:"frameTree"`
	}
	_ = json.Unmarshal(raw, &ft)
	raw, err = c.call(sid, "Page.createIsolatedWorld", map[string]any{"frameId": ft.FrameTree.Frame.ID, "worldName": syncWorld})
	if err != nil {
		return 0, err
	}
	var w struct {
		ID int `json:"executionContextId"`
	}
	_ = json.Unmarshal(raw, &w)
	return w.ID, nil
}

func (s *syncer) poll() {
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-tick.C:
		}
		s.mu.Lock()
		ctx := s.mctx
		s.mu.Unlock()
		if ctx == 0 {
			id, err := worldContext(s.mconn, s.msid)
			if err != nil {
				if isSessionGone(err) {
					s.shutdown("主控标签页已关闭")
					return
				}
				continue
			}
			s.mu.Lock()
			s.mctx, ctx = id, id
			s.mu.Unlock()
		}
		raw, err := s.mconn.call(s.msid, "Runtime.evaluate", map[string]any{"expression": syncDrainExpr(), "contextId": ctx, "returnByValue": true})
		if err != nil {
			if isSessionGone(err) {
				s.shutdown("主控标签页已关闭")
				return
			}
			s.mu.Lock()
			if s.mctx == ctx {
				s.mctx = 0
			}
			s.mu.Unlock()
			continue
		}
		var r struct {
			Result struct {
				Value string `json:"value"`
			} `json:"result"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Result.Value == "" {
			continue
		}
		var evs []syncEvent
		_ = json.Unmarshal([]byte(r.Result.Value), &evs)
		for _, ev := range evs {
			if ev.T != "" && ev.T != "nav" {
				s.dispatch(ev)
			}
		}
	}
}

func (s *syncer) onMasterEvent(m cdpMsg) {
	switch m.Method {
	case "Page.frameNavigated":
		if m.Session != s.msid {
			return
		}
		var p struct {
			Frame struct {
				ParentID string `json:"parentId"`
				URL      string `json:"url"`
				Fragment string `json:"urlFragment"`
			} `json:"frame"`
		}
		if json.Unmarshal(m.Params, &p) == nil && p.Frame.ParentID == "" && p.Frame.URL != "" {
			s.mu.Lock()
			s.mctx = 0
			s.mu.Unlock()
			s.dispatch(syncEvent{T: "nav", URL: p.Frame.URL + p.Frame.Fragment})
		}
	case "Page.navigatedWithinDocument":
		if m.Session != s.msid {
			return
		}
		var p struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(m.Params, &p) == nil && p.URL != "" {
			s.dispatch(syncEvent{T: "nav", URL: p.URL})
		}
	case "Target.detachedFromTarget", "Inspector.detached":
		var p struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(m.Params, &p)
		if m.Method == "Inspector.detached" || p.SessionID == s.msid {
			s.shutdown("主控标签页已关闭")
		}
	}
}

func (s *syncer) dispatch(ev syncEvent) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.events++
	s.last = ev.T
	fs := append([]*syncFollower(nil), s.followers...)
	s.mu.Unlock()
	for _, f := range fs {
		select {
		case f.q <- ev:
		default: // follower can't keep up: skip this event rather than block the master
		}
	}
	s.emitSoon()
}

// emitSoon throttles state emission during bursts of events.
func (s *syncer) emitSoon() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.emitTimer != nil {
		return
	}
	wait := 300*time.Millisecond - time.Since(s.lastEmit)
	if wait < 0 {
		wait = 0
	}
	s.emitTimer = time.AfterFunc(wait, func() {
		s.mu.Lock()
		s.emitTimer = nil
		s.lastEmit = time.Now()
		run := s.running
		s.mu.Unlock()
		if run {
			s.emit(s.state())
		}
	})
}

func (s *syncer) state() SyncerState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := SyncerState{Running: s.running, Master: s.master, MasterName: s.mname, Events: s.events, LastEvent: s.last, StartedAt: s.started, Reason: s.reason, Followers: []SyncFollowerState{}}
	for _, f := range s.followers {
		st.Followers = append(st.Followers, SyncFollowerState{ID: f.id, Name: f.name, OK: f.ok, Events: atomic.LoadInt64(&f.events), Error: f.err})
	}
	return st
}

func (s *syncer) shutdown(reason string) {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		s.running = false
		if reason != "" && reason != "restart" {
			s.reason = reason
		}
		if s.emitTimer != nil {
			s.emitTimer.Stop()
			s.emitTimer = nil
		}
		fs := s.followers
		mc := s.mconn
		s.mu.Unlock()
		close(s.stop)
		if mc != nil {
			mc.c.Close()
		}
		for _, f := range fs {
			f.conn.c.Close()
		}
		s.emit(s.state())
	})
}

func (s *syncer) dropFollower(f *syncFollower, err error) {
	s.mu.Lock()
	f.ok = false
	f.err = err.Error()
	alive := 0
	for _, x := range s.followers {
		if x.ok {
			alive++
		}
	}
	s.mu.Unlock()
	f.conn.c.Close()
	if alive == 0 {
		s.shutdown("所有被控环境都已断开")
		return
	}
	s.emit(s.state())
}

func (s *syncer) replay(f *syncFollower) {
	for {
		select {
		case <-s.stop:
			return
		case <-f.conn.done:
			s.dropFollower(f, fmt.Errorf("浏览器已关闭"))
			return
		case ev := <-f.q:
			if err := f.apply(ev); err != nil {
				s.dropFollower(f, err)
				return
			}
			atomic.AddInt64(&f.events, 1)
		}
	}
}

// ---------- follower replay ----------

func (f *syncFollower) call(method string, p map[string]any) (json.RawMessage, error) {
	return f.conn.call(f.sid, method, p)
}

// eval runs an expression in an isolated world of the follower's main frame
// (never in the page's own world) and returns the JSON value.
func (f *syncFollower) eval(expr string) (json.RawMessage, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if f.ctx == 0 {
			id, err := worldContext(f.conn, f.sid)
			if err != nil {
				return nil, err
			}
			f.ctx = id
		}
		raw, err := f.call("Runtime.evaluate", map[string]any{"expression": expr, "contextId": f.ctx, "returnByValue": true})
		if err != nil {
			f.ctx = 0 // context gone after a navigation: make a new world and retry
			if isSessionGone(err) {
				return nil, err
			}
			continue
		}
		var r struct {
			Result struct {
				Value json.RawMessage `json:"value"`
			} `json:"result"`
		}
		_ = json.Unmarshal(raw, &r)
		return r.Result.Value, nil
	}
	return nil, nil // best effort: evaluation failures never drop a follower
}

func isSessionGone(err error) bool {
	m := err.Error()
	return strings.Contains(m, "browser closed") || strings.Contains(m, "ession with given id not found") || strings.Contains(m, "No target with given id")
}

// soft turns errors that only mean "the page was busy" into nil, so only a
// closed browser/tab drops a follower.
func soft(err error) error {
	if err != nil && isSessionGone(err) {
		return err
	}
	return nil
}

func (f *syncFollower) url() string {
	raw, err := f.conn.call("", "Target.getTargetInfo", map[string]any{"targetId": f.tid})
	if err != nil {
		return ""
	}
	var ti struct {
		TargetInfo struct {
			URL string `json:"url"`
		} `json:"targetInfo"`
	}
	_ = json.Unmarshal(raw, &ti)
	return ti.TargetInfo.URL
}

var mouseButtons = []string{"left", "middle", "right", "back", "forward"}

func cdpMods(m int) int { return m } // same bit layout as CDP (Alt=1 Ctrl=2 Meta=4 Shift=8)

func (f *syncFollower) apply(ev syncEvent) error {
	switch ev.T {
	case "click":
		x, y := ev.X, ev.Y
		pos := false
		if ev.Sel != "" {
			v, err := f.eval(fmt.Sprintf(`(() => { const e = document.querySelector(%s); if (!e) return null; e.scrollIntoView({block: 'nearest', inline: 'nearest'}); const r = e.getBoundingClientRect(); return [r.left + r.width * %g, r.top + r.height * %g]; })()`, jsStr(ev.Sel), ev.RX, ev.RY))
			if err != nil {
				return err
			}
			var p []float64
			if json.Unmarshal(v, &p) == nil && len(p) == 2 {
				x, y, pos = p[0], p[1], true
			}
		}
		if !pos && ev.VX > 0 && ev.VY > 0 {
			if v, err := f.eval(`[innerWidth, innerHeight]`); err != nil {
				return err
			} else {
				var p []float64
				if json.Unmarshal(v, &p) == nil && len(p) == 2 {
					x, y = ev.VX*p[0], ev.VY*p[1]
				}
			}
		}
		btn := "left"
		if ev.Btn >= 0 && ev.Btn < len(mouseButtons) {
			btn = mouseButtons[ev.Btn]
		}
		n := ev.N
		if n < 1 {
			n = 1
		}
		if _, err := f.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": x, "y": y}); soft(err) != nil {
			return err
		}
		if _, err := f.call("Input.dispatchMouseEvent", map[string]any{"type": "mousePressed", "x": x, "y": y, "button": btn, "clickCount": n}); soft(err) != nil {
			return err
		}
		_, err := f.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseReleased", "x": x, "y": y, "button": btn, "clickCount": n})
		return soft(err)
	case "scroll":
		x, y := 100.0, 100.0
		if v, err := f.eval(`[innerWidth, innerHeight]`); err != nil {
			return err
		} else {
			var p []float64
			if json.Unmarshal(v, &p) == nil && len(p) == 2 {
				x, y = ev.VX*p[0], ev.VY*p[1]
			}
		}
		_, err := f.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": x, "y": y, "deltaX": ev.DX, "deltaY": ev.DY})
		return soft(err)
	case "text":
		if ev.Key != "" && ev.Key == ev.S {
			// A typed printable key: replay as a key press so keydown/keypress
			// listeners fire on followers too (the default action inserts the text).
			p := map[string]any{"type": "keyDown", "key": ev.Key, "code": ev.Code, "text": ev.S, "unmodifiedText": ev.S, "windowsVirtualKeyCode": ev.KC, "nativeVirtualKeyCode": ev.KC, "modifiers": cdpMods(ev.Mods)}
			if _, err := f.call("Input.dispatchKeyEvent", p); soft(err) != nil {
				return err
			}
			_, err := f.call("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": ev.Key, "code": ev.Code, "windowsVirtualKeyCode": ev.KC, "nativeVirtualKeyCode": ev.KC, "modifiers": cdpMods(ev.Mods)})
			return soft(err)
		}
		_, err := f.call("Input.insertText", map[string]any{"text": ev.S})
		return soft(err)
	case "key":
		p := map[string]any{"type": "rawKeyDown", "key": ev.Key, "code": ev.Code, "windowsVirtualKeyCode": ev.KC, "nativeVirtualKeyCode": ev.KC, "modifiers": cdpMods(ev.Mods)}
		if ev.Key == "Enter" {
			p["type"], p["text"] = "keyDown", "\r"
		}
		if _, err := f.call("Input.dispatchKeyEvent", p); soft(err) != nil {
			return err
		}
		up := map[string]any{"type": "keyUp", "key": ev.Key, "code": ev.Code, "windowsVirtualKeyCode": ev.KC, "nativeVirtualKeyCode": ev.KC, "modifiers": cdpMods(ev.Mods)}
		_, err := f.call("Input.dispatchKeyEvent", up)
		return soft(err)
	case "select":
		_, err := f.eval(fmt.Sprintf(`(() => { const e = document.querySelector(%s); if (!e) return false; e.value = %s; e.dispatchEvent(new Event('input', {bubbles: true})); e.dispatchEvent(new Event('change', {bubbles: true})); return true; })()`, jsStr(ev.Sel), jsStr(ev.Value)))
		return err
	case "nav":
		// Give a replayed click the chance to navigate the follower itself first.
		time.Sleep(350 * time.Millisecond)
		cur := f.url()
		if cur == ev.URL || !(strings.HasPrefix(ev.URL, "http") || ev.URL == "about:blank") {
			return nil
		}
		f.ctx = 0
		_, err := f.call("Page.navigate", map[string]any{"url": ev.URL})
		return soft(err)
	}
	return nil
}
