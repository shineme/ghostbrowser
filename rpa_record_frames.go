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
// document by cdpFrameSelFn.

func cdpFrameDrainExpr(n int, paused, overlay, scroll bool, mode string) string {
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

// newRecFrameReg wires the shared frame registry to the recorder.
func (r *recorder) initFrames() {
	r.frameReg = newFrameReg(r.conn, recWorld, r.tabOfSession)
}

// setupFrameSession prepares an OOPIF target session like a page session.
func (r *recorder) setupFrameSession(parentSid, sid, targetID, parentFrame string, waiting bool) {
	tab := r.tabOfSession(parentSid)
	o := r.core.opts
	_, _ = r.call(sid, "Page.enable", nil)
	_, _ = r.call(sid, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": recCaptureScript(o.Overlay, o.Scroll), "worldName": recWorld, "runImmediately": true})
	autoAttachIframes(r.conn, sid, true)
	guardUA(r.conn, r.ws, sid)
	if waiting {
		_, _ = r.call(sid, "Runtime.runIfWaitingForDebugger", nil)
	}
	if tab == "" {
		return
	}
	r.fsess[sid] = tab
	r.frames[targetID] = &cdpFrame{id: targetID, sid: sid, parent: parentFrame, parentSid: parentSid, tab: tab, oopif: true}
	r.treeDirty[sid] = true
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

func (r *recorder) drainFrame(tc *recTabConn, f *cdpFrame) {
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
	raw, err := r.call(f.sid, "Runtime.evaluate", map[string]any{"expression": cdpFrameDrainExpr(n, paused, o.Overlay, o.Scroll, mode), "contextId": f.ctx, "returnByValue": true})
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
