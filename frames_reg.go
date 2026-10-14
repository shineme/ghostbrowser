package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// frameReg tracks the iframes of a set of CDP page sessions (same-process
// child frames from Page.getFrameTree, out-of-process ones registered when
// their "iframe" target attaches) and computes each frame's path: the
// selectors of its ancestor <iframe> elements from the top document. The
// recorder and the multi-window syncer share it. Not safe for concurrent
// use: each owner drives it from one goroutine.
type frameReg struct {
	conn      *cdpConn
	world     string                  // isolated world used in parent documents
	tabOf     func(sid string) string // session → tab (page target id)
	frames    map[string]*cdpFrame    // frame id → frame
	fsess     map[string]string       // iframe target session → tab
	treeDirty map[string]bool         // sessions whose frame tree changed
}

func newFrameReg(c *cdpConn, world string, tabOf func(string) string) *frameReg {
	return &frameReg{conn: c, world: world, tabOf: tabOf, frames: map[string]*cdpFrame{}, fsess: map[string]string{}, treeDirty: map[string]bool{}}
}

// recWorldCtxNamed returns (creating if needed) the context of the named
// isolated world in a frame.
func recWorldCtxNamed(c *cdpConn, sid, frameID, world string) (int, error) {
	raw, err := c.call(sid, "Page.createIsolatedWorld", map[string]any{"frameId": frameID, "worldName": world})
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

type cdpFrame struct {
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

// cdpFrameSelFn builds a robust selector for an iframe element (bound to
// this), unique in its document: id, name, title, a stable part of src, then
// a structural path.
const cdpFrameSelFn = `function() {
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

// cdpFrameDrainExpr drains an iframe's queue; it returns "\x00" when the
// world has no live script yet (the caller then sends the full script).
// frameGone forgets a frame and its descendants.
func (r *frameReg) frameGone(id string) {
	delete(r.frames, id)
	for k, f := range r.frames {
		if f.parent == id {
			r.frameGone(k)
		}
	}
}

// sessionGone forgets an iframe target session.
func (r *frameReg) sessionGone(sid string) {
	delete(r.fsess, sid)
	delete(r.treeDirty, sid)
	for k, f := range r.frames {
		if f.sid == sid || f.parentSid == sid {
			delete(r.frames, k)
		}
	}
}

// refreshTree registers a session's same-process child frames.
func (r *frameReg) refreshTree(sid string) {
	tab := r.tabOf(sid)
	if tab == "" {
		return
	}
	raw, err := r.conn.call(sid, "Page.getFrameTree", nil)
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
				r.frames[n.Frame.ID] = &cdpFrame{id: n.Frame.ID, sid: sid, parent: n.Frame.ParentID, parentSid: sid, tab: tab}
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
func (r *frameReg) framePath(f *cdpFrame, depth int) (string, bool) {
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
func (r *frameReg) ownerSelector(f *cdpFrame) string {
	ctx, err := recWorldCtxNamed(r.conn, f.parentSid, f.parent, r.world)
	if err != nil {
		return ""
	}
	obj := ""
	if raw, err := r.conn.call(f.parentSid, "DOM.getFrameOwner", map[string]any{"frameId": f.id}); err == nil {
		var o struct {
			BackendNodeID int `json:"backendNodeId"`
		}
		_ = json.Unmarshal(raw, &o)
		if o.BackendNodeID != 0 {
			if raw, err := r.conn.call(f.parentSid, "DOM.resolveNode", map[string]any{"backendNodeId": o.BackendNodeID, "executionContextId": ctx}); err == nil {
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
	defer func() { _, _ = r.conn.call(f.parentSid, "Runtime.releaseObject", map[string]any{"objectId": obj}) }()
	raw, err := r.conn.call(f.parentSid, "Runtime.callFunctionOn", map[string]any{"objectId": obj, "functionDeclaration": cdpFrameSelFn, "returnByValue": true})
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
func (r *frameReg) scanOwner(f *cdpFrame, ctx int) string {
	raw, err := r.conn.call(f.parentSid, "Runtime.evaluate", map[string]any{"expression": "Array.from(document.querySelectorAll('iframe,frame'))", "contextId": ctx})
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
	defer func() {
		_, _ = r.conn.call(f.parentSid, "Runtime.releaseObject", map[string]any{"objectId": arr.Result.ObjectID})
	}()
	raw, err = r.conn.call(f.parentSid, "Runtime.getProperties", map[string]any{"objectId": arr.Result.ObjectID, "ownProperties": true})
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
		d, err := r.conn.call(f.parentSid, "DOM.describeNode", map[string]any{"objectId": p.Value.ObjectID})
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
