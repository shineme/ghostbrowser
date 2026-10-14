package main

import (
	"encoding/json"
	"strings"
	"time"
)

// Recorder iframe support.
//
// The capture script runs in every frame (addScriptToEvaluateOnNewDocument
// with a world name covers a session's same-process child frames; OOPIFs
// arrive as "iframe" targets through Target.setAutoAttach on each page and
// iframe session and get the same script). Go drains each frame's queue in
// its isolated world and stamps every event with the frame's path: the
// selectors of the ancestor <iframe> elements, generated in each parent
// document by recFrameSelFn.

type recFrame struct {
	id        string // frame id (= target id for an OOPIF's main frame)
	sid       string // session that owns the frame's document
	parent    string // parent frame id
	parentSid string // session that owns the parent document
	tab       string // page target the frame belongs to
	oopif     bool
	ctx       int
	path      string
	pathOK    bool
	pickSeq   int
	retryAt   time.Time
}

// recFrameSelFn builds a robust selector for an iframe element (bound to
// this), unique in its document: id, name, title, a stable part of src, then
// a structural path.
const recFrameSelFn = `function() {
  const el = this, doc = el.ownerDocument, tag = el.tagName.toLowerCase();
  const esc = (s) => (window.CSS && CSS.escape) ? CSS.escape(s) : String(s).replace(/[^\w-]/g, '\\$&');
  const cstr = (s) => String(s).replace(/\\/g, '\\\\').replace(/"/g, '\\"');
  const bad = (v) => !v || v.length > 48 || /\d{4,}|^\d|[:{}]|[a-f0-9]{10,}/i.test(v);
  const uniq = (s) => { try { const n = doc.querySelectorAll(s); return n.length === 1 && n[0] === el; } catch (e) { return false; } };
  if (el.id && !bad(el.id) && uniq(tag + '#' + esc(el.id))) return tag + '#' + esc(el.id);
  for (const a of ['name', 'title', 'data-testid', 'aria-label']) {
    const v = el.getAttribute(a);
    if (v && v.length <= 60 && !bad(v.replace(/\s/g, ''))) { const s = tag + '[' + a + '="' + cstr(v) + '"]'; if (uniq(s)) return s; }
  }
  const src = el.getAttribute('src') || '';
  if (src && !src.startsWith('data:') && !src.startsWith('javascript:')) {
    try {
      const u = new URL(src, doc.baseURI);
      const segs = u.pathname.split('/').filter(p => p && !/\d{4,}|[a-f0-9]{12,}/i.test(p));
      const cands = [];
      if (segs.length) cands.push(u.host + '/' + segs.join('/'), '/' + segs.join('/'), segs[segs.length - 1]);
      cands.push(u.host);
      for (const c of cands) { const s = tag + '[src*="' + cstr(c) + '"]'; if (c && uniq(s)) return s; }
    } catch (e) {}
  }
  const parts = [];
  for (let e = el; e && e.nodeType === 1 && e !== doc.documentElement; e = e.parentElement) {
    if (e !== el && e.id && !bad(e.id) && uniq('#' + esc(e.id) + ' ' + parts.join(' > '))) { parts.unshift('#' + esc(e.id)); break; }
    let p = e.tagName.toLowerCase();
    const sib = e.parentElement ? Array.from(e.parentElement.children).filter(c => c.tagName === e.tagName) : [e];
    if (sib.length > 1) p += ':nth-of-type(' + (sib.indexOf(e) + 1) + ')';
    parts.unshift(p);
    if (uniq(parts.join(' > '))) return parts.join(' > ');
  }
  return parts.join(' > ');
}`

// recFrameDrainExpr drains an iframe's queue; it returns "\x00" when the
// world has no live script yet (the caller then sends the full script).
func recFrameDrainExpr(n int, paused, overlay, scroll bool, mode string) string {
	return "(() => { const R = window.__gbRec; if (!R || R.dead || R.doc !== document) return '\\u0000'; return R.drain(" +
		jsonArgs(n, paused, overlay, scroll, mode) + "); })()"
}

func jsonArgs(v ...any) string {
	out := make([]string, len(v))
	for i, x := range v {
		b, _ := json.Marshal(x)
		out[i] = string(b)
	}
	return strings.Join(out, ", ")
}

// tabOfSession maps a session (page or iframe target) to its tab.
func (r *recorder) tabOfSession(sid string) string {
	if t := r.bySid[sid]; t != "" {
		return t
	}
	return r.fsess[sid]
}

// setupFrameSession prepares an OOPIF target session like a page session.
func (r *recorder) setupFrameSession(parentSid, sid, targetID, parentFrame string, waiting bool) {
	tab := r.tabOfSession(parentSid)
	o := r.core.opts
	_, _ = r.call(sid, "Page.enable", nil)
	_, _ = r.call(sid, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": recCaptureScript(o.Overlay, o.Scroll), "worldName": recWorld, "runImmediately": true})
	_, _ = r.call(sid, "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true})
	guardUA(r.conn, r.ws, sid)
	if waiting {
		_, _ = r.call(sid, "Runtime.runIfWaitingForDebugger", nil)
	}
	if tab == "" {
		return
	}
	r.fsess[sid] = tab
	r.frames[targetID] = &recFrame{id: targetID, sid: sid, parent: parentFrame, parentSid: parentSid, tab: tab, oopif: true}
	r.treeDirty[sid] = true
}

// frameGone forgets a frame and its descendants.
func (r *recorder) frameGone(id string) {
	delete(r.frames, id)
	for k, f := range r.frames {
		if f.parent == id {
			r.frameGone(k)
		}
	}
}

// sessionGone forgets an iframe target session.
func (r *recorder) sessionGone(sid string) {
	delete(r.fsess, sid)
	delete(r.treeDirty, sid)
	for k, f := range r.frames {
		if f.sid == sid || f.parentSid == sid {
			delete(r.frames, k)
		}
	}
}

// refreshTree registers a session's same-process child frames.
func (r *recorder) refreshTree(sid string) {
	tab := r.tabOfSession(sid)
	if tab == "" {
		return
	}
	raw, err := r.call(sid, "Page.getFrameTree", nil)
	if err != nil {
		return
	}
	type node struct {
		Frame struct {
			ID       string `json:"id"`
			ParentID string `json:"parentId"`
		} `json:"frame"`
		Children []json.RawMessage `json:"childFrames"`
	}
	seen := map[string]bool{}
	var walk func(raw json.RawMessage, root bool)
	walk = func(raw json.RawMessage, root bool) {
		var n node
		if json.Unmarshal(raw, &n) != nil {
			return
		}
		seen[n.Frame.ID] = true
		if !root {
			if f := r.frames[n.Frame.ID]; f == nil || (!f.oopif && f.sid != sid) {
				r.frames[n.Frame.ID] = &recFrame{id: n.Frame.ID, sid: sid, parent: n.Frame.ParentID, parentSid: sid, tab: tab}
			}
		}
		for _, c := range n.Children {
			walk(c, false)
		}
	}
	var top struct {
		FrameTree json.RawMessage `json:"frameTree"`
	}
	_ = json.Unmarshal(raw, &top)
	walk(top.FrameTree, true)
	for k, f := range r.frames {
		if f.sid == sid && !f.oopif && !seen[k] {
			delete(r.frames, k)
		}
	}
}

// framePath computes (and caches) the frame path of f.
func (r *recorder) framePath(f *recFrame, depth int) (string, bool) {
	if f.pathOK {
		return f.path, true
	}
	if depth > 12 || f.parent == "" {
		return "", false
	}
	prefix := ""
	if pf := r.frames[f.parent]; pf != nil {
		p, ok := r.framePath(pf, depth+1)
		if !ok {
			return "", false
		}
		prefix = p
	} else if f.parent != f.tab {
		return "", false // parent not known yet
	}
	sel := r.ownerSelector(f)
	if sel == "" {
		return "", false
	}
	if prefix != "" {
		f.path = prefix + framePathSep + sel
	} else {
		f.path = sel
	}
	f.pathOK = true
	return f.path, true
}

// ownerSelector finds f's <iframe> element in the parent document and
// builds its selector there (in the capture world).
func (r *recorder) ownerSelector(f *recFrame) string {
	ctx, err := recWorldCtx(r.conn, f.parentSid, f.parent)
	if err != nil {
		return ""
	}
	obj := ""
	if raw, err := r.call(f.parentSid, "DOM.getFrameOwner", map[string]any{"frameId": f.id}); err == nil {
		var o struct {
			BackendNodeID int `json:"backendNodeId"`
		}
		_ = json.Unmarshal(raw, &o)
		if o.BackendNodeID != 0 {
			if raw, err := r.call(f.parentSid, "DOM.resolveNode", map[string]any{"backendNodeId": o.BackendNodeID, "executionContextId": ctx}); err == nil {
				var ro struct {
					Object struct {
						ObjectID string `json:"objectId"`
					} `json:"object"`
				}
				_ = json.Unmarshal(raw, &ro)
				obj = ro.Object.ObjectID
			}
		}
	}
	if obj == "" {
		obj = r.scanOwner(f, ctx)
	}
	if obj == "" {
		return ""
	}
	defer func() { _, _ = r.call(f.parentSid, "Runtime.releaseObject", map[string]any{"objectId": obj}) }()
	raw, err := r.call(f.parentSid, "Runtime.callFunctionOn", map[string]any{"objectId": obj, "functionDeclaration": recFrameSelFn, "returnByValue": true})
	if err != nil {
		return ""
	}
	var v string
	if decodeEvalResult(raw, &v) != nil {
		return ""
	}
	return v
}

// scanOwner is the fallback when DOM.getFrameOwner cannot see the frame:
// describe each iframe element of the parent and match its frameId.
func (r *recorder) scanOwner(f *recFrame, ctx int) string {
	raw, err := r.call(f.parentSid, "Runtime.evaluate", map[string]any{"expression": "Array.from(document.querySelectorAll('iframe,frame'))", "contextId": ctx})
	if err != nil {
		return ""
	}
	var arr struct {
		Result struct {
			ObjectID string `json:"objectId"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &arr)
	if arr.Result.ObjectID == "" {
		return ""
	}
	defer func() { _, _ = r.call(f.parentSid, "Runtime.releaseObject", map[string]any{"objectId": arr.Result.ObjectID}) }()
	raw, err = r.call(f.parentSid, "Runtime.getProperties", map[string]any{"objectId": arr.Result.ObjectID, "ownProperties": true})
	if err != nil {
		return ""
	}
	var props struct {
		Result []struct {
			Name  string `json:"name"`
			Value struct {
				ObjectID string `json:"objectId"`
			} `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &props)
	for _, p := range props.Result {
		if p.Value.ObjectID == "" || p.Name == "__proto__" {
			continue
		}
		d, err := r.call(f.parentSid, "DOM.describeNode", map[string]any{"objectId": p.Value.ObjectID})
		if err != nil {
			continue
		}
		var dn struct {
			Node struct {
				FrameID string `json:"frameId"`
			} `json:"node"`
		}
		_ = json.Unmarshal(d, &dn)
		if dn.Node.FrameID == f.id {
			return p.Value.ObjectID
		}
	}
	return ""
}

// drainFrames drains every known iframe of tab tc.
func (r *recorder) drainFrames(tc *recTabConn) {
	for sid := range r.treeDirty {
		if r.tabOfSession(sid) == tc.id {
			delete(r.treeDirty, sid)
			r.refreshTree(sid)
		}
	}
	for _, f := range r.frames {
		if f.tab == tc.id {
			r.drainFrame(tc, f)
		}
	}
}

func (r *recorder) drainFrame(tc *recTabConn, f *recFrame) {
	if tc.dialog || time.Now().Before(f.retryAt) {
		return
	}
	if f.ctx == 0 {
		id, err := recWorldCtx(r.conn, f.sid, f.id)
		if err != nil {
			f.retryAt = time.Now().Add(400 * time.Millisecond)
			return
		}
		f.ctx = id
	}
	path, ok := r.framePath(f, 0)
	if !ok {
		f.retryAt = time.Now().Add(300 * time.Millisecond)
		return
	}
	r.mu.Lock()
	n := countSteps(r.core.steps)
	paused, o := r.paused, r.core.opts
	mode := ""
	if tc.id == r.core.cur && f.pickSeq != r.pickSeq {
		mode = r.pickMode
		if mode == "" {
			mode = "none"
		}
		f.pickSeq = r.pickSeq
	}
	r.mu.Unlock()
	raw, err := r.call(f.sid, "Runtime.evaluate", map[string]any{"expression": recFrameDrainExpr(n, paused, o.Overlay, o.Scroll, mode), "contextId": f.ctx, "returnByValue": true})
	var v string
	if err == nil {
		err = decodeEvalResult(raw, &v)
	}
	if err == nil && v == "\x00" {
		// First visit (or a new document): install the script, then drain.
		raw, err = r.call(f.sid, "Runtime.evaluate", map[string]any{"expression": recDrainExpr(n, paused, o.Overlay, o.Scroll, mode), "contextId": f.ctx, "returnByValue": true})
		v = ""
		if err == nil {
			err = decodeEvalResult(raw, &v)
		}
	}
	if err != nil {
		f.ctx = 0
		f.retryAt = time.Now().Add(200 * time.Millisecond)
		return
	}
	if v == "" || v == "\x00" {
		return
	}
	var evs []recEvent
	if json.Unmarshal([]byte(v), &evs) != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ev := range evs {
		ev.Frame = path
		r.applyEvent(tc, ev, &f.pickSeq)
	}
}

// applyEvent handles one drained event (r.mu held). seq is the source
// frame's pick sequence, updated so a mode it reported is not echoed back.
func (r *recorder) applyEvent(tc *recTabConn, ev recEvent, seq *int) {
	switch ev.T {
	case "ctl":
		switch ev.A {
		case "pause":
			r.setPaused(true)
		case "resume":
			r.setPaused(false)
		case "stop":
			r.stoppedBy = "overlay"
		}
		r.dirty = true
		return
	case "mode":
		r.pickMode = ev.A
		r.pickSeq++
		*seq = r.pickSeq
		return
	case "extract", "assert":
		// The pick is done: every frame leaves pick mode.
		r.pickMode = ""
		r.pickSeq++
		*seq = r.pickSeq
	}
	if r.paused {
		return
	}
	r.core.event(tc.id, ev)
	r.dirty = true
}
