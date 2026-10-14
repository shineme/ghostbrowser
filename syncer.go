package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Group control ("多窗口同步器"): what the user does in any tab of the master
// browser is replayed in the matching tab of every follower. Tabs are mapped
// master targetId -> follower targetId: a new tab, a tab switch and a closed
// tab in the master are mirrored (when tab sync is on), and every captured
// event goes to the follower tab mapped to the master tab it came from.
// Per-type toggles (click/input/key/scroll/tabs) and a random per-follower
// delay range can be changed while running (SyncOptions).
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
	ID        string `json:"id"`
	Name      string `json:"name"`
	OK        bool   `json:"ok"`
	Events    int64  `json:"events"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latencyMs"` // capture -> replayed, last event (includes the random delay)
	Tabs      int    `json:"tabs"`      // master tabs mapped to a tab of this follower
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
	Options    SyncOptions         `json:"options"`
	Tabs       int                 `json:"tabs"` // master's open tabs
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
	// iframe events (syncer_frames.go): frame path, the point as a fraction
	// of the master's top-level viewport, and the frame's own viewport size.
	Frame string  `json:"frame,omitempty"`
	TVX   float64 `json:"tvx,omitempty"`
	TVY   float64 `json:"tvy,omitempty"`
	FW    float64 `json:"fw,omitempty"`
	FH    float64 `json:"fh,omitempty"`

	tab    string    // master targetId the event came from (tab events: the tab concerned)
	at     time.Time // when Go received it; replay delay is measured from here
	opener bool      // tabnew: opened by the page (link/window.open), not by the user
}

const syncWorld = "__gb_sync"

// syncCaptureScript runs in the master's isolated world. The DOM is shared
// with the page, so capture-phase listeners see every user action.
func syncCaptureScript() string {
	return `(() => {
  if (window.__gbSyncOn) return; // runs in iframes too; Go tags their events
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
    send({t: 'scroll', dx: e.deltaX * k, dy: e.deltaY * k, vx: e.clientX / innerWidth, vy: e.clientY / innerHeight, fw: innerWidth, fh: innerHeight});
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
// and returns "v|" or "h|" (tab visible = active, or hidden) followed by the
// queued events as JSON (empty when none), clearing the queue.
func syncDrainExpr() string {
	return `(() => { ` + syncCaptureScript() + ` const q = window.__gbq || []; window.__gbq = []; window.__gbLast = Date.now(); return (document.visibilityState === 'visible' ? 'v|' : 'h|') + (q.length ? JSON.stringify(q) : ''); })()`
}

func jsStr(s string) string { b, _ := json.Marshal(s); return string(b) }

// ---------- options ----------

// SyncOptions are the syncer's per-type switches and the random replay delay.
// They can be changed while the syncer runs (SetOptions).
type SyncOptions struct {
	Click    bool `json:"click"`    // 同步点击 (left/right/middle clicks)
	Input    bool `json:"input"`    // 同步输入 (typed text, IME, paste, <select>)
	Key      bool `json:"key"`      // 同步按键 (Enter, Backspace, arrows, shortcuts)
	Scroll   bool `json:"scroll"`   // 同步滚动 (mouse wheel)
	Tabs     bool `json:"tabs"`     // 同步标签页 (new / switch / close tab)
	DelayMin int  `json:"delayMin"` // ms, random delay per follower per event
	DelayMax int  `json:"delayMax"`
}

const syncMaxDelay = 10000

func defaultSyncOptions() SyncOptions {
	return SyncOptions{Click: true, Input: true, Key: true, Scroll: true, Tabs: true}
}

// norm clamps the delay range to 0..10 s and orders it.
func (o SyncOptions) norm() SyncOptions {
	o.DelayMin, o.DelayMax = clampI(o.DelayMin, 0, syncMaxDelay), clampI(o.DelayMax, 0, syncMaxDelay)
	if o.DelayMin > o.DelayMax {
		o.DelayMin, o.DelayMax = o.DelayMax, o.DelayMin
	}
	return o
}

// allows reports whether an event type is synced under these options.
// Navigation of a mapped tab is always followed.
func (o SyncOptions) allows(t string) bool {
	switch t {
	case "click":
		return o.Click
	case "scroll":
		return o.Scroll
	case "text", "select":
		return o.Input
	case "key":
		return o.Key
	case "tabnew", "tabswitch", "tabclose":
		return o.Tabs
	}
	return true
}

// sampleDelay draws a uniform delay in [min, max] ms.
func sampleDelay(o SyncOptions, r *rand.Rand) time.Duration {
	o = o.norm()
	d := o.DelayMin
	if o.DelayMax > o.DelayMin {
		d += r.Intn(o.DelayMax - o.DelayMin + 1)
	}
	return time.Duration(d) * time.Millisecond
}

// ---------- state ----------

// fTab is a follower tab with its attached session.
type fTab struct {
	tid, sid string
	ctx      int        // isolated world execution context, 0 = none yet
	drv      *cdpDriver // iframe resolver (syncer_frames.go)
}

type syncFollower struct {
	id, name string
	ws       string
	conn     *cdpConn
	q        chan syncEvent
	events   int64
	latency  int64
	ok       bool
	err      string

	mu     sync.Mutex
	tabs   map[string]string // master targetId -> follower targetId
	sess   map[string]*fTab  // follower targetId -> session
	gone   map[string]bool   // follower targets already destroyed
	opened chan string       // follower tabs opened by a page, not yet mapped
	cur    *fTab             // tab the event being applied goes to
	rnd    *rand.Rand
	next   time.Time // replay schedule keeps events in order
}

// masterTab is one tab of the master browser being captured.
type masterTab struct {
	tid, sid string
	ctx      int
}

type syncer struct {
	mu         sync.Mutex
	running    bool
	master     string
	mname      string
	mconn      *cdpConn
	mtabs      map[string]*masterTab // by targetId, in attach order (see order)
	order      []string
	mgone      map[string]bool
	active     string // master's active (visible) tab
	opts       SyncOptions
	tabsPaused int32 // >0 while a tab action runs: tab events are not mirrored
	followers  []*syncFollower
	events     int64
	last       string
	started    int64
	reason     string
	stop       chan struct{}
	stopOnce   sync.Once
	emit       func(SyncerState)
	lastEmit   time.Time
	emitTimer  *time.Timer
	ctl        chan cdpMsg // master Target.* events, handled in order by one goroutine
	mws        string      // master CDP endpoint
	fmu        sync.Mutex  // guards fr
	fr         *frameReg   // master iframes (syncer_frames.go)
}

var groupSync = &syncerHolder{opts: defaultSyncOptions()}

// syncerHolder owns the single active syncer and the emit callback.
type syncerHolder struct {
	mu   sync.Mutex
	cur  *syncer
	emit func(SyncerState)
	prev SyncerState
	opts SyncOptions // options for the next start (and the running syncer)
}

func (h *syncerHolder) State() SyncerState {
	h.mu.Lock()
	s := h.cur
	prev := h.prev
	opts := h.opts
	h.mu.Unlock()
	if s == nil {
		if prev.Followers == nil {
			prev.Followers = []SyncFollowerState{}
		}
		prev.Running = false
		prev.Options = opts
		return prev
	}
	return s.state()
}

// Options returns the current sync options.
func (h *syncerHolder) Options() SyncOptions {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.opts
}

// SetOptions changes the options, also for a running syncer (applies to the
// next captured event).
func (h *syncerHolder) SetOptions(o SyncOptions) SyncerState {
	o = o.norm()
	h.mu.Lock()
	h.opts = o
	s := h.cur
	h.mu.Unlock()
	if s != nil {
		s.mu.Lock()
		s.opts = o
		s.mu.Unlock()
		st := s.state()
		s.emit(st)
		return st
	}
	return h.State()
}

func (h *syncerHolder) Stop(reason string) {
	h.mu.Lock()
	s := h.cur
	h.mu.Unlock()
	if s != nil {
		s.shutdown(reason)
	}
}

// running returns the active syncer, or nil.
func (h *syncerHolder) running() *syncer {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cur != nil {
		h.cur.mu.Lock()
		run := h.cur.running
		h.cur.mu.Unlock()
		if run {
			return h.cur
		}
	}
	return nil
}

// Start begins syncing master's tabs to followers' tabs.
// wsOf returns the browser websocket of a running profile, name its display name.
func (h *syncerHolder) Start(master string, followers []string, wsOf func(string) (string, bool), name func(string) string) (SyncerState, error) {
	h.Stop("restart")
	mws, ok := wsOf(master)
	if !ok {
		return SyncerState{}, fmt.Errorf("主控环境没有在运行")
	}
	s := &syncer{master: master, mname: name(master), stop: make(chan struct{}), started: time.Now().UnixMilli(), followers: []*syncFollower{},
		mtabs: map[string]*masterTab{}, mgone: map[string]bool{}, ctl: make(chan cdpMsg, 256), opts: h.Options()}
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
	for _, f := range s.followers {
		s.mapTabs(f)
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
	go s.poll()
	go s.control()
	st := s.state()
	s.emit(st)
	return st, nil
}

// ---------- tabs ----------

type pageTab struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	OpenerID string `json:"openerId"`
}

// isPageTab: a normal tab (not DevTools, not browser UI).
func isPageTab(t pageTab) bool {
	return t.Type == "page" && !strings.HasPrefix(t.URL, "devtools://")
}

// pageTabs lists a browser's normal tabs in creation order.
func pageTabs(c *cdpConn) ([]pageTab, error) {
	raw, err := c.call("", "Target.getTargets", nil)
	if err != nil {
		return nil, err
	}
	var tl struct {
		TargetInfos []pageTab `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	out := []pageTab{}
	for _, t := range tl.TargetInfos {
		if isPageTab(t) {
			out = append(out, t)
		}
	}
	return out, nil
}

// firstPageTarget returns the first normal page tab of a browser.
func firstPageTarget(c *cdpConn) (string, string, error) {
	tabs, err := pageTabs(c)
	if err != nil {
		return "", "", err
	}
	if len(tabs) == 0 {
		return "", "", fmt.Errorf("浏览器没有打开的标签页")
	}
	return tabs[0].TargetID, tabs[0].URL, nil
}

func attachTarget(c *cdpConn, tid string) (string, error) {
	raw, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": tid, "flatten": true})
	if err != nil {
		return "", err
	}
	var at struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &at)
	if at.SessionID == "" {
		return "", fmt.Errorf("无法连接标签页")
	}
	return at.SessionID, nil
}

func attachPage(c *cdpConn) (tid, sid string, err error) {
	tid, _, err = firstPageTarget(c)
	if err != nil {
		return
	}
	sid, err = attachTarget(c, tid)
	return
}

// tabVisible reports document.visibilityState == "visible" for a session
// (the active tab of a non-minimised window).
func tabVisible(c *cdpConn, sid string) bool {
	raw, err := c.call(sid, "Runtime.evaluate", map[string]any{"expression": "document.visibilityState", "returnByValue": true})
	if err != nil {
		return false
	}
	var r struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.Result.Value == "visible"
}

func attachFollower(id, name, ws string) (*syncFollower, error) {
	f := &syncFollower{id: id, name: name, q: make(chan syncEvent, 512), ok: true, tabs: map[string]string{}, sess: map[string]*fTab{},
		gone: map[string]bool{}, opened: make(chan string, 16), rnd: rand.New(rand.NewSource(time.Now().UnixNano() ^ int64(len(id))<<40 ^ int64(rand.Int63())))}
	c, err := dialCDP(ws, f.onEvent)
	if err != nil {
		return nil, err
	}
	f.conn, f.ws = c, ws
	tabs, err := pageTabs(c)
	if err == nil && len(tabs) == 0 {
		err = fmt.Errorf("浏览器没有打开的标签页")
	}
	if err == nil {
		_, err = c.call("", "Target.setDiscoverTargets", map[string]any{"discover": true})
	}
	if err != nil {
		c.c.Close()
		return nil, err
	}
	return f, nil
}

func (f *syncFollower) onEvent(m cdpMsg) {
	var p struct {
		TargetInfo pageTab `json:"targetInfo"`
		TargetID   string  `json:"targetId"`
	}
	_ = json.Unmarshal(m.Params, &p)
	switch m.Method {
	case "Target.targetCreated":
		if isPageTab(p.TargetInfo) && p.TargetInfo.OpenerID != "" {
			select {
			case f.opened <- p.TargetInfo.TargetID:
			default:
			}
		}
	case "Target.targetDestroyed":
		f.mu.Lock()
		f.gone[p.TargetID] = true
		delete(f.sess, p.TargetID)
		for m, t := range f.tabs {
			if t == p.TargetID {
				delete(f.tabs, m)
			}
		}
		f.mu.Unlock()
	}
}

// session returns (attaching on first use) the session of a follower tab.
func (f *syncFollower) session(tid string) (*fTab, error) {
	f.mu.Lock()
	t := f.sess[tid]
	f.mu.Unlock()
	if t != nil {
		return t, nil
	}
	sid, err := attachTarget(f.conn, tid)
	if err != nil {
		return nil, err
	}
	guardUA(f.conn, f.ws, sid)
	t = &fTab{tid: tid, sid: sid}
	f.mu.Lock()
	f.sess[tid] = t
	f.mu.Unlock()
	return t, nil
}

// activeOf picks the visible tab of a list of sessions (fallback: first).
func activeOf(c *cdpConn, tabs []pageTab) string {
	for _, t := range tabs {
		sid, err := attachTarget(c, t.TargetID)
		if err != nil {
			continue
		}
		v := tabVisible(c, sid)
		_, _ = c.call("", "Target.detachFromTarget", map[string]any{"sessionId": sid})
		if v {
			return t.TargetID
		}
	}
	if len(tabs) > 0 {
		return tabs[0].TargetID
	}
	return ""
}

// mapTabs (re)builds a follower's tab mapping: the master's active tab goes
// to the follower's active tab, then tabs showing the same URL pair up, then
// the rest in the order CDP lists them.
func (s *syncer) mapTabs(f *syncFollower) {
	tabs, err := pageTabs(f.conn)
	if err != nil {
		return
	}
	murl := map[string]string{}
	if mts, err := pageTabs(s.mconn); err == nil {
		for _, t := range mts {
			murl[t.TargetID] = t.URL
		}
	}
	s.mu.Lock()
	order := append([]string(nil), s.order...)
	active := s.active
	s.mu.Unlock()
	fact := activeOf(f.conn, tabs)
	m := map[string]string{}
	used := map[string]bool{}
	if active != "" && fact != "" {
		m[active], used[fact] = fact, true
	}
	for _, mt := range order {
		if _, ok := m[mt]; ok || murl[mt] == "" {
			continue
		}
		for _, ft := range tabs {
			if !used[ft.TargetID] && ft.URL == murl[mt] {
				m[mt], used[ft.TargetID] = ft.TargetID, true
				break
			}
		}
	}
	i := 0
	for _, mt := range order {
		if _, ok := m[mt]; ok {
			continue
		}
		for i < len(tabs) && used[tabs[i].TargetID] {
			i++
		}
		if i >= len(tabs) {
			break
		}
		m[mt], used[tabs[i].TargetID] = tabs[i].TargetID, true
		i++
	}
	f.mu.Lock()
	f.tabs = m
	f.mu.Unlock()
	for {
		select {
		case <-f.opened:
			continue
		default:
		}
		break
	}
}

// ---------- master ----------

func (s *syncer) attachMaster(ws string) error {
	c, err := dialCDP(ws, s.onMasterEvent)
	if err != nil {
		return err
	}
	s.mconn, s.mws = c, ws
	s.fmu.Lock()
	s.initFrames()
	s.fmu.Unlock()
	tabs, err := pageTabs(c)
	if err == nil && len(tabs) == 0 {
		err = fmt.Errorf("浏览器没有打开的标签页")
	}
	if err == nil {
		_, err = c.call("", "Target.setDiscoverTargets", map[string]any{"discover": true})
	}
	if err != nil {
		c.c.Close()
		return err
	}
	for _, t := range tabs {
		mt, err := s.attachMasterTab(t.TargetID)
		if err != nil {
			continue
		}
		s.mu.Lock()
		s.mtabs[t.TargetID] = mt
		s.order = append(s.order, t.TargetID)
		s.mu.Unlock()
		if s.active == "" && tabVisible(c, mt.sid) {
			s.active = t.TargetID
		}
	}
	if len(s.order) == 0 {
		c.c.Close()
		return fmt.Errorf("无法连接标签页")
	}
	if s.active == "" {
		s.active = s.order[0]
	}
	go func() {
		select {
		case <-c.done:
			s.shutdown("主控浏览器已关闭")
		case <-s.stop:
		}
	}()
	return nil
}

// attachMasterTab installs capture in a master tab. Runtime bindings only
// report calls while the Runtime domain is enabled, and Runtime.enable is
// detectable by pages, so events are queued inside the isolated world and
// drained by polling with Runtime.evaluate (which works without
// Runtime.enable). The new-document script makes capture start before page
// scripts run on every navigation.
func (s *syncer) attachMasterTab(tid string) (*masterTab, error) {
	c := s.mconn
	sid, err := attachTarget(c, tid)
	if err != nil {
		return nil, err
	}
	for _, st := range []struct {
		m string
		p map[string]any
	}{
		{"Page.enable", nil},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": syncCaptureScript(), "worldName": syncWorld, "runImmediately": true}},
	} {
		if _, err := c.call(sid, st.m, st.p); err != nil {
			return nil, err
		}
	}
	s.prepareMasterSession(sid)
	mt := &masterTab{tid: tid, sid: sid}
	// Prime the capture world: whatever a previous syncer left queued there
	// is discarded, not replayed.
	if id, err := worldContext(c, sid); err == nil {
		if _, err := c.call(sid, "Runtime.evaluate", map[string]any{"expression": syncDrainExpr(), "contextId": id, "returnByValue": true}); err == nil {
			mt.ctx = id
		}
	}
	return mt, nil
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

func (s *syncer) masterTabs() []*masterTab {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*masterTab, 0, len(s.order))
	for _, id := range s.order {
		if t := s.mtabs[id]; t != nil {
			out = append(out, t)
		}
	}
	return out
}

func (s *syncer) dropMasterTab(tid string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mgone[tid] = true
	if _, ok := s.mtabs[tid]; !ok {
		return false
	}
	delete(s.mtabs, tid)
	for i, id := range s.order {
		if id == tid {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
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
		for _, mt := range s.masterTabs() {
			s.drain(mt)
			s.drainFrames(mt)
		}
	}
}

func (s *syncer) drain(mt *masterTab) {
	s.mu.Lock()
	ctx := mt.ctx
	s.mu.Unlock()
	if ctx == 0 {
		id, err := worldContext(s.mconn, mt.sid)
		if err != nil {
			return
		}
		s.mu.Lock()
		mt.ctx, ctx = id, id
		s.mu.Unlock()
	}
	raw, err := s.mconn.call(mt.sid, "Runtime.evaluate", map[string]any{"expression": syncDrainExpr(), "contextId": ctx, "returnByValue": true})
	if err != nil {
		s.mu.Lock()
		if mt.ctx == ctx {
			mt.ctx = 0
		}
		s.mu.Unlock()
		return
	}
	var r struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Result.Value) < 2 {
		return
	}
	vis, body := r.Result.Value[:1] == "v", r.Result.Value[2:]
	if vis {
		s.mu.Lock()
		switched := s.active != mt.tid && s.mtabs[mt.tid] != nil
		if switched {
			s.active = mt.tid
		}
		s.mu.Unlock()
		if switched {
			s.dispatch(syncEvent{T: "tabswitch", tab: mt.tid})
		}
	}
	if body == "" {
		return
	}
	var evs []syncEvent
	_ = json.Unmarshal([]byte(body), &evs)
	for _, ev := range evs {
		if ev.T != "" && ev.T != "nav" {
			ev.tab = mt.tid
			s.dispatch(ev)
		}
	}
}

func (s *syncer) tabOfSession(sid string) *masterTab {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.mtabs {
		if t.sid == sid {
			return t
		}
	}
	return nil
}

func (s *syncer) onMasterEvent(m cdpMsg) {
	if s.onMasterFrameEvent(m) {
		return
	}
	switch m.Method {
	case "Page.frameNavigated":
		mt := s.tabOfSession(m.Session)
		if mt == nil {
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
			mt.ctx = 0
			s.mu.Unlock()
			s.dispatch(syncEvent{T: "nav", URL: p.Frame.URL + p.Frame.Fragment, tab: mt.tid})
		}
	case "Page.navigatedWithinDocument":
		mt := s.tabOfSession(m.Session)
		if mt == nil {
			return
		}
		var p struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(m.Params, &p) == nil && p.URL != "" {
			s.dispatch(syncEvent{T: "nav", URL: p.URL, tab: mt.tid})
		}
	case "Target.targetCreated", "Target.targetDestroyed":
		select {
		case s.ctl <- m:
		case <-s.stop:
		}
	}
}

// control handles master tab creation/closing in order (attaching needs CDP
// round trips, which can't run on the connection's reader).
func (s *syncer) control() {
	for {
		var m cdpMsg
		select {
		case <-s.stop:
			return
		case m = <-s.ctl:
		}
		var p struct {
			TargetInfo pageTab `json:"targetInfo"`
			TargetID   string  `json:"targetId"`
		}
		_ = json.Unmarshal(m.Params, &p)
		switch m.Method {
		case "Target.targetCreated":
			t := p.TargetInfo
			s.mu.Lock()
			known := s.mtabs[t.TargetID] != nil || s.mgone[t.TargetID]
			s.mu.Unlock()
			if !isPageTab(t) || known {
				continue
			}
			mt, err := s.attachMasterTab(t.TargetID)
			if err != nil {
				continue
			}
			s.dispatch(syncEvent{T: "tabnew", tab: t.TargetID, URL: t.URL, opener: t.OpenerID != ""})
			s.mu.Lock()
			if !s.mgone[t.TargetID] {
				s.mtabs[t.TargetID] = mt
				s.order = append(s.order, t.TargetID)
			}
			s.mu.Unlock()
			s.emitSoon()
		case "Target.targetDestroyed":
			if s.dropMasterTab(p.TargetID) {
				s.dispatch(syncEvent{T: "tabclose", tab: p.TargetID})
				s.emitSoon()
			}
		}
	}
}

func (s *syncer) dispatch(ev syncEvent) {
	s.mu.Lock()
	if !s.running || !s.opts.allows(ev.T) {
		s.mu.Unlock()
		return
	}
	if strings.HasPrefix(ev.T, "tab") && atomic.LoadInt32(&s.tabsPaused) > 0 {
		s.mu.Unlock()
		return
	}
	s.events++
	s.last = ev.T
	ev.at = time.Now()
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

// pauseTabs stops tab mirroring while a tab action changes tabs itself;
// the returned func re-maps every follower and resumes.
func (s *syncer) pauseTabs() func() {
	atomic.AddInt32(&s.tabsPaused, 1)
	return func() {
		// let the Target events of the action arrive before re-mapping
		time.Sleep(400 * time.Millisecond)
		if tabs, err := pageTabs(s.mconn); err == nil {
			s.mu.Lock()
			act := s.active
			s.mu.Unlock()
			if a := activeOf(s.mconn, tabs); a != "" {
				act = a
			}
			s.mu.Lock()
			if s.mtabs[act] != nil {
				s.active = act
			}
			s.mu.Unlock()
		}
		s.mu.Lock()
		fs := append([]*syncFollower(nil), s.followers...)
		s.mu.Unlock()
		for _, f := range fs {
			if f.ok {
				s.mapTabs(f)
			}
		}
		atomic.AddInt32(&s.tabsPaused, -1)
		s.emitSoon()
	}
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
	st := SyncerState{Running: s.running, Master: s.master, MasterName: s.mname, Events: s.events, LastEvent: s.last, StartedAt: s.started, Reason: s.reason,
		Followers: []SyncFollowerState{}, Options: s.opts, Tabs: len(s.order)}
	for _, f := range s.followers {
		f.mu.Lock()
		n := 0
		for mt := range f.tabs {
			if s.mtabs[mt] != nil {
				n++
			}
		}
		f.mu.Unlock()
		st.Followers = append(st.Followers, SyncFollowerState{ID: f.id, Name: f.name, OK: f.ok, Events: atomic.LoadInt64(&f.events), Error: f.err,
			LatencyMs: atomic.LoadInt64(&f.latency), Tabs: n})
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
			// Human-like jitter: each follower waits its own random delay
			// after the master's action. Events keep their order, so an
			// event never runs before the previous one (latency stays <= max).
			s.mu.Lock()
			o := s.opts
			s.mu.Unlock()
			due := ev.at.Add(sampleDelay(o, f.rnd))
			if due.Before(f.next) {
				due = f.next
			}
			f.next = due
			if w := time.Until(due); w > 0 {
				select {
				case <-s.stop:
					return
				case <-time.After(w):
				}
			}
			if err := f.apply(ev); err != nil {
				if isSessionGone(err) && f.conn.isOpen() {
					// only that tab closed; the browser is still there
					continue
				}
				s.dropFollower(f, err)
				return
			}
			atomic.StoreInt64(&f.latency, time.Since(ev.at).Milliseconds())
			atomic.AddInt64(&f.events, 1)
		}
	}
}

func (c *cdpConn) isOpen() bool {
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

// ---------- follower replay ----------

func (f *syncFollower) call(method string, p map[string]any) (json.RawMessage, error) {
	return f.conn.call(f.cur.sid, method, p)
}

// eval runs an expression in an isolated world of the follower's main frame
// (never in the page's own world) and returns the JSON value.
func (f *syncFollower) eval(expr string) (json.RawMessage, error) {
	for attempt := 0; attempt < 2; attempt++ {
		if f.cur.ctx == 0 {
			id, err := worldContext(f.conn, f.cur.sid)
			if err != nil {
				return nil, err
			}
			f.cur.ctx = id
		}
		raw, err := f.call("Runtime.evaluate", map[string]any{"expression": expr, "contextId": f.cur.ctx, "returnByValue": true})
		if err != nil {
			f.cur.ctx = 0 // context gone after a navigation: make a new world and retry
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
	raw, err := f.conn.call("", "Target.getTargetInfo", map[string]any{"targetId": f.cur.tid})
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

// apply replays one event in the follower tab mapped to the event's master tab.
func (f *syncFollower) apply(ev syncEvent) error {
	switch ev.T {
	case "tabnew", "tabswitch", "tabclose":
		return f.applyTab(ev)
	}
	f.mu.Lock()
	tid := f.tabs[ev.tab]
	f.mu.Unlock()
	if tid == "" {
		return nil // master tab has no counterpart here (统一标签页 aligns them)
	}
	t, err := f.session(tid)
	if err != nil {
		if isSessionGone(err) && f.conn.isOpen() {
			return nil
		}
		return soft(err)
	}
	f.cur = t
	if ev.Frame != "" {
		if handled, err := f.applyFrame(ev); handled {
			return err
		}
	}
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
		f.cur.ctx = 0
		_, err := f.call("Page.navigate", map[string]any{"url": ev.URL})
		return soft(err)
	}
	return nil
}

// applyTab mirrors a master tab being opened, activated or closed.
func (f *syncFollower) applyTab(ev syncEvent) error {
	f.mu.Lock()
	tid := f.tabs[ev.tab]
	f.mu.Unlock()
	switch ev.T {
	case "tabnew":
		if tid != "" {
			return nil
		}
		nt := ""
		if ev.opener {
			// The replayed click usually opens the follower's tab by itself:
			// adopt it instead of opening a second one.
			deadline := time.After(1500 * time.Millisecond)
		wait:
			for {
				select {
				case id := <-f.opened:
					f.mu.Lock()
					mapped, gone := false, f.gone[id]
					for _, v := range f.tabs {
						mapped = mapped || v == id
					}
					f.mu.Unlock()
					if !mapped && !gone {
						nt = id
						break wait
					}
				case <-deadline:
					break wait
				}
			}
		}
		if nt == "" {
			u := ev.URL
			if u == "" {
				u = "about:blank"
			}
			raw, err := f.conn.call("", "Target.createTarget", map[string]any{"url": u})
			if err != nil {
				return soft(err)
			}
			var r struct {
				TargetID string `json:"targetId"`
			}
			_ = json.Unmarshal(raw, &r)
			nt = r.TargetID
		}
		if nt != "" {
			f.mu.Lock()
			f.tabs[ev.tab] = nt
			f.mu.Unlock()
		}
	case "tabswitch":
		if tid != "" {
			_, err := f.conn.call("", "Target.activateTarget", map[string]any{"targetId": tid})
			return soft(err)
		}
	case "tabclose":
		if tid == "" {
			return nil
		}
		f.mu.Lock()
		delete(f.tabs, ev.tab)
		f.mu.Unlock()
		// Never close a follower's last tab (that would close its window).
		if tabs, err := pageTabs(f.conn); err == nil && len(tabs) <= 1 {
			return nil
		}
		_, err := f.conn.call("", "Target.closeTarget", map[string]any{"targetId": tid})
		return soft(err)
	}
	return nil
}
