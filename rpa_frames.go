package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// iframe support for the RPA runner (and the helpers the recorder and the
// syncer share).
//
// A frame path lists the selectors of the <iframe> elements to walk through,
// starting at the top document: "iframe#pay >> iframe[name=card]". Each hop is
// resolved in its parent frame: the iframe element is found with the usual
// selector resolver, DOM.describeNode gives its frameId, and the frame is then
// either in the same renderer (contentDocument is present: evaluate in an
// isolated world created for that frameId on the parent's session) or out of
// process (an OOPIF: attach to the iframe target, whose id is the frameId).
//
// Coordinates: every hop adds the iframe's content-box origin (border and
// padding included, CSS scale taken from the bounding box) so a point inside
// the frame can be dispatched as real input on the top-level page session,
// which Chromium routes into the right frame, OOPIFs included.

// framePathSep separates the hops of a frame path.
const framePathSep = " >> "

// parseFramePath splits "a >> b" into its selectors. A leading "frame:" is
// accepted, empty hops are dropped, and "top"/"main"/"主页面" mean the top
// document (an empty path).
func parseFramePath(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(strings.TrimPrefix(s, "frame:"))
	switch strings.ToLower(s) {
	case "", "top", "main", "主页面":
		return nil
	}
	out := []string{}
	for _, p := range strings.Split(s, ">>") {
		p = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "frame:"))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinFramePath(p []string) string { return strings.Join(p, framePathSep) }

// frameLabel is the short form used in logs: "iframe#pay › iframe[name=card]".
func frameLabel(p []string) string { return strings.Join(p, " › ") }

// switchFramePath applies a switch_frame target to the current path:
// ""/top/main = top document, "parent" = one level up, anything else is a
// path relative to the current frame.
func switchFramePath(cur []string, target string) []string {
	t := strings.TrimSpace(target)
	switch strings.ToLower(t) {
	case "", "top", "main", "主页面", "frame:top":
		return nil
	case "parent", "..", "上一级":
		if len(cur) == 0 {
			return nil
		}
		return append([]string(nil), cur[:len(cur)-1]...)
	}
	return append(append([]string(nil), cur...), parseFramePath(t)...)
}

// frameCtx is one resolved frame.
type frameCtx struct {
	sid     string  // CDP session that owns the frame's document
	frameID string  // "" = the session's main frame (top document)
	world   int     // isolated world for DOM work (0 = main world, top only)
	oopif   bool    // frame is the main frame of an iframe target session
	doc     int     // same-process frame: backendNodeId of its document
	ox, oy  float64 // frame viewport origin in top-level viewport CSS px
	scale   float64 // CSS scale of the frame relative to the top document
}

// errFrameNotReady is retried until the step's timeout.
type errFrameNotReady struct{ msg string }

func (e errFrameNotReady) Error() string { return e.msg }

const rpaWorldName = "gb_frame_rpa"

// frameBox measures an iframe element's content-box origin in its own
// frame's viewport, plus the element's CSS scale. Bound to `this`.
const frameBoxFn = `function(reveal) {
  if (reveal) { try { this.scrollIntoView({block: 'nearest', inline: 'nearest'}); } catch (e) {} }
  const r = this.getBoundingClientRect(); const cs = getComputedStyle(this);
  const sx = this.offsetWidth ? r.width / this.offsetWidth : 1, sy = this.offsetHeight ? r.height / this.offsetHeight : 1;
  return [r.left + (this.clientLeft + parseFloat(cs.paddingLeft || 0)) * sx, r.top + (this.clientTop + parseFloat(cs.paddingTop || 0)) * sy, sx || 1, r.width, r.height];
}`

// UseFrame sets the frame later element operations run in (nil = top) and
// waits up to timeout for it to exist.
func (d *cdpDriver) UseFrame(path []string, timeout time.Duration) error {
	d.fpath = append([]string(nil), path...)
	if len(path) == 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = d.implicit
	}
	deadline := time.Now().Add(timeout)
	for {
		_, err := d.resolveFrame(path, false)
		if err == nil {
			return nil
		}
		if fatal(err) {
			return err
		}
		var nr errFrameNotReady
		if !errors.As(err, &nr) || time.Now().After(deadline) {
			return fmt.Errorf("找不到框架 %s（%v）", frameLabel(path), err)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

// resolveFrame walks path from the top document. reveal scrolls each iframe
// element into view in its parent first.
func (d *cdpDriver) resolveFrame(path []string, reveal bool) (*frameCtx, error) {
	cur := &frameCtx{sid: d.sid, scale: 1}
	if d.isoTop && len(path) > 0 {
		// Helpers outside the RPA runner (the syncer) never touch the top
		// page's own world: look up the first hop in an isolated world too.
		w, err := d.isolatedWorld(d.sid, d.target)
		if err != nil {
			return nil, err
		}
		cur.world, cur.frameID = w, d.target
	}
	for i, sel := range path {
		next, err := d.frameHop(cur, sel, reveal)
		if err != nil {
			if _, ok := err.(errFrameNotReady); ok && i > 0 {
				return nil, errFrameNotReady{fmt.Sprintf("第 %d 层框架 %s：%v", i+1, sel, err)}
			}
			return nil, err
		}
		cur = next
	}
	return cur, nil
}

func (d *cdpDriver) frameHop(cur *frameCtx, sel string, reveal bool) (*frameCtx, error) {
	expr := "(() => {" + rpaFind + "const el = __gbq(" + jsq(sel) + "); if (!el) return null; const t = el.tagName; if (t !== 'IFRAME' && t !== 'FRAME') throw new Error('不是 iframe 元素：' + " + jsq(sel) + "); return el; })()"
	raw, err := d.evalObj(cur, expr)
	if err != nil {
		return nil, err
	}
	if raw == "" {
		return nil, errFrameNotReady{"页面里还没有 iframe " + sel}
	}
	defer func() { _, _ = d.conn.call(cur.sid, "Runtime.releaseObject", map[string]any{"objectId": raw}) }()
	box, err := d.conn.call(cur.sid, "Runtime.callFunctionOn", map[string]any{"objectId": raw, "functionDeclaration": frameBoxFn, "arguments": []any{map[string]any{"value": reveal}}, "returnByValue": true})
	if err != nil {
		return nil, d.wrapErr("Runtime.callFunctionOn", err)
	}
	var bv struct {
		Result struct {
			Value []float64 `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(box, &bv)
	desc, err := d.conn.call(cur.sid, "DOM.describeNode", map[string]any{"objectId": raw})
	if err != nil {
		return nil, d.wrapErr("DOM.describeNode", err)
	}
	var dn struct {
		Node struct {
			FrameID         string `json:"frameId"`
			ContentDocument *struct {
				BackendNodeID int `json:"backendNodeId"`
			} `json:"contentDocument"`
		} `json:"node"`
	}
	_ = json.Unmarshal(desc, &dn)
	if dn.Node.FrameID == "" {
		return nil, errFrameNotReady{"iframe " + sel + " 还没有加载"}
	}
	next := &frameCtx{frameID: dn.Node.FrameID, scale: cur.scale}
	if len(bv.Result.Value) >= 3 {
		next.ox = cur.ox + bv.Result.Value[0]*cur.scale
		next.oy = cur.oy + bv.Result.Value[1]*cur.scale
		next.scale = cur.scale * bv.Result.Value[2]
	}
	if dn.Node.ContentDocument != nil {
		next.sid, next.doc = cur.sid, dn.Node.ContentDocument.BackendNodeID
	} else {
		sid, err := d.frameSession(dn.Node.FrameID)
		if err != nil {
			return nil, err
		}
		next.sid, next.oopif = sid, true
	}
	w, err := d.isolatedWorld(next.sid, next.frameID)
	if err != nil {
		return nil, err
	}
	next.world = w
	return next, nil
}

// frameSession attaches to an out-of-process iframe's target (target id =
// frame id) and caches the session.
func (d *cdpDriver) frameSession(frameID string) (string, error) {
	d.mu.Lock()
	sid := d.fsess[frameID]
	d.mu.Unlock()
	if sid != "" {
		return sid, nil
	}
	r, err := d.conn.call("", "Target.attachToTarget", map[string]any{"targetId": frameID, "flatten": true})
	if err != nil {
		if d.gone() {
			return "", errBrowserGone
		}
		// The frame is between processes (navigating): try again shortly.
		return "", errFrameNotReady{"跨域框架还在加载"}
	}
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	d.mu.Lock()
	if d.fsess == nil {
		d.fsess = map[string]string{}
	}
	d.fsess[frameID] = at.SessionID
	d.mu.Unlock()
	guardUA(d.conn, d.ws, at.SessionID)
	return at.SessionID, nil
}

func (d *cdpDriver) isolatedWorld(sid, frameID string) (int, error) {
	key := sid + "/" + frameID
	d.mu.Lock()
	w := d.worlds[key]
	d.mu.Unlock()
	if w != 0 {
		return w, nil
	}
	r, err := d.conn.call(sid, "Page.createIsolatedWorld", map[string]any{"frameId": frameID, "worldName": rpaWorldName})
	if err != nil {
		if d.gone() {
			return 0, errBrowserGone
		}
		d.dropFrameCaches(sid, frameID)
		return 0, errFrameNotReady{"框架还在加载"}
	}
	var cw struct {
		ExecutionContextID int `json:"executionContextId"`
	}
	_ = json.Unmarshal(r, &cw)
	d.mu.Lock()
	if d.worlds == nil {
		d.worlds = map[string]int{}
	}
	d.worlds[key] = cw.ExecutionContextID
	d.mu.Unlock()
	return cw.ExecutionContextID, nil
}

// dropFrameCaches forgets a stale world or OOPIF session.
func (d *cdpDriver) dropFrameCaches(sid, frameID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.worlds, sid+"/"+frameID)
	if d.fsess[frameID] == sid {
		delete(d.fsess, frameID)
		for k := range d.worlds {
			if strings.HasPrefix(k, sid+"/") {
				delete(d.worlds, k)
			}
		}
	}
}

// staleCtx reports CDP errors that mean the world or session went away
// (the frame navigated or was removed): resolve again.
func staleCtx(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	return strings.Contains(m, "Cannot find context") || strings.Contains(m, "Session with given id not found") ||
		strings.Contains(m, "Execution context was destroyed") || strings.Contains(m, "No frame") || strings.Contains(m, "Inspected target navigated or closed")
}

// evalParams builds Runtime.evaluate params for a frame context.
func evalParams(fc *frameCtx, expr string, byValue bool) map[string]any {
	p := map[string]any{"expression": expr, "returnByValue": byValue, "awaitPromise": true, "userGesture": true}
	if fc != nil && fc.world != 0 {
		p["contextId"] = fc.world
	}
	return p
}

// evalObj evaluates expr in fc (top when fc.frameID == "") and returns the
// result's objectId ("" for null/undefined).
func (d *cdpDriver) evalObj(fc *frameCtx, expr string) (string, error) {
	p := evalParams(fc, expr, false)
	delete(p, "awaitPromise")
	raw, err := d.conn.call(fc.sid, "Runtime.evaluate", p)
	if err != nil {
		if staleCtx(err) && fc.frameID != "" {
			d.dropFrameCaches(fc.sid, fc.frameID)
			return "", errFrameNotReady{"框架已刷新"}
		}
		return "", d.wrapErr("Runtime.evaluate", err)
	}
	var r struct {
		Result struct {
			ObjectID string `json:"objectId"`
			Subtype  string `json:"subtype"`
		} `json:"result"`
		Exc *json.RawMessage `json:"exceptionDetails"`
	}
	_ = json.Unmarshal(raw, &r)
	if r.Exc != nil {
		return "", decodeEvalErr(raw)
	}
	if r.Result.Subtype == "null" {
		return "", nil
	}
	return r.Result.ObjectID, nil
}

// decodeEvalResult turns a Runtime.evaluate / callFunctionOn reply into out
// or the page's error message.
func decodeEvalResult(raw json.RawMessage, out any) error {
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exc *json.RawMessage `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	if r.Exc != nil {
		return decodeEvalErr(raw)
	}
	if out != nil && len(r.Result.Value) > 0 {
		return json.Unmarshal(r.Result.Value, out)
	}
	return nil
}

func decodeEvalErr(raw json.RawMessage) error {
	var r struct {
		Exc *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	_ = json.Unmarshal(raw, &r)
	if r.Exc == nil {
		return nil
	}
	msg := r.Exc.Exception.Description
	if msg == "" {
		msg = r.Exc.Text
	}
	msg = strings.TrimPrefix(strings.SplitN(msg, "\n", 2)[0], "Error: ")
	return fmt.Errorf("%s", msg)
}

// frameEval runs expr in the current frame (d.fpath). main selects the
// frame's page world (execute_js needs the page's own globals); otherwise an
// isolated world is used. It returns the frame as measured just before the
// evaluation (nil when at the top).
func (d *cdpDriver) frameEval(expr string, out any, main bool) (*frameCtx, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		fc, err := d.resolveFrame(d.fpath, false)
		if err != nil {
			return nil, err
		}
		raw, err := d.evalInFrame(fc, expr, main)
		if err == nil {
			return fc, decodeEvalResult(raw, out)
		}
		if !staleCtx(err) {
			return fc, d.wrapErr("Runtime.evaluate", err)
		}
		d.dropFrameCaches(fc.sid, fc.frameID)
		lastErr = errFrameNotReady{"框架已刷新"}
	}
	return nil, lastErr
}

func (d *cdpDriver) evalInFrame(fc *frameCtx, expr string, main bool) (json.RawMessage, error) {
	if !main || fc.oopif {
		p := evalParams(fc, expr, true)
		if main {
			delete(p, "contextId") // the iframe target's own main world
		}
		return d.conn.call(fc.sid, "Runtime.evaluate", p)
	}
	// Same-process frame, page world: resolve the frame's document in its
	// main world and call a function bound to it (no Runtime.enable needed).
	r, err := d.conn.call(fc.sid, "DOM.resolveNode", map[string]any{"backendNodeId": fc.doc})
	if err != nil {
		return nil, err
	}
	var o struct {
		Object struct {
			ObjectID string `json:"objectId"`
		} `json:"object"`
	}
	_ = json.Unmarshal(r, &o)
	defer func() {
		_, _ = d.conn.call(fc.sid, "Runtime.releaseObject", map[string]any{"objectId": o.Object.ObjectID})
	}()
	return d.conn.call(fc.sid, "Runtime.callFunctionOn", map[string]any{"objectId": o.Object.ObjectID,
		"functionDeclaration": "function() { return (0, eval)(" + jsq(expr) + "); }", "returnByValue": true, "awaitPromise": true, "userGesture": true})
}

// frameObj returns an objectId for expr's value in the current frame
// (isolated world), with the session that owns it.
func (d *cdpDriver) frameObj(expr string) (sid, obj string, err error) {
	for attempt := 0; attempt < 3; attempt++ {
		fc, err := d.resolveFrame(d.fpath, false)
		if err != nil {
			return "", "", err
		}
		obj, err := d.evalObj(fc, expr)
		if _, nr := err.(errFrameNotReady); nr {
			continue
		}
		return fc.sid, obj, err
	}
	return "", "", errFrameNotReady{"框架已刷新"}
}

// elemShotJS returns an element's box in its frame's viewport after
// scrolling it into view.
const elemShotJS = `el.scrollIntoView({block: 'center', inline: 'center'}); await new Promise(r => setTimeout(r, 120)); const r = el.getBoundingClientRect(); return [r.left, r.top, r.width, r.height];`

// ScreenshotElement captures one element (inside the current frame, if any).
func (d *cdpDriver) ScreenshotElement(sel string) ([]byte, error) {
	if err := d.waitPresent(sel); err != nil {
		return nil, err
	}
	if len(d.fpath) > 0 {
		if _, err := d.resolveFrame(d.fpath, true); err != nil {
			return nil, err
		}
	}
	var b []float64
	if err := d.eval(onEl(sel, elemShotJS), &b); err != nil {
		return nil, err
	}
	fc, err := d.resolveFrame(d.fpath, false)
	if err != nil {
		return nil, err
	}
	if len(b) < 4 || b[2] <= 0 || b[3] <= 0 {
		return nil, fmt.Errorf("元素没有可见区域：%s", sel)
	}
	// The clip is in document coordinates of the top page: add its scroll.
	var px, py float64
	if raw, err := d.call("Page.getLayoutMetrics", nil); err == nil {
		var m struct {
			VV struct{ PageX, PageY float64 } `json:"cssVisualViewport"`
		}
		_ = json.Unmarshal(raw, &m)
		px, py = m.VV.PageX, m.VV.PageY
	}
	clip := map[string]any{"x": px + fc.ox + b[0]*fc.scale, "y": py + fc.oy + b[1]*fc.scale, "width": b[2] * fc.scale, "height": b[3] * fc.scale, "scale": 1}
	return d.capture(map[string]any{"format": "png", "clip": clip})
}
