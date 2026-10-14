package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// Syncer iframe support.
//
// Master: the capture script also runs in iframes (the new-document script
// with a world name covers a tab's same-process child frames; OOPIFs attach
// as "iframe" targets through Target.setAutoAttach on each master session).
// Events drained from a frame carry its frame path (frameReg) plus the
// click/wheel point as a fraction of the master's top-level viewport.
//
// Followers: the same frame path is resolved in the follower tab with the
// RPA frame resolver (isolated worlds only); the element (or the frame-local
// point) is translated into top-level coordinates and dispatched as real
// input on the page session, which Chromium routes into the frame, OOPIFs
// included. If the follower has no such frame, the top-level fraction is used.

// initFrames creates the master's frame registry. It is owned by the poll
// goroutine: CDP event handlers hand it changes through frameOp, so they
// never wait on a drain and master events keep their order.
func (s *syncer) initFrames() {
	s.fr = newFrameReg(s.mconn, syncWorld, s.tabIDOfSession)
}

// frameOp queues a registry change for the poll goroutine.
func (s *syncer) frameOp(fn func(fr *frameReg)) {
	if s.fops == nil {
		return
	}
	select {
	case s.fops <- fn:
	default: // registry falls behind; the next tree refresh catches up
	}
}

// runFrameOps applies queued registry changes (poll goroutine).
func (s *syncer) runFrameOps() {
	for {
		select {
		case fn := <-s.fops:
			if s.fr != nil {
				fn(s.fr)
			}
		default:
			return
		}
	}
}

// tabIDOfSession maps a master session (tab or iframe target) to its tab
// (poll goroutine).
func (s *syncer) tabIDOfSession(sid string) string {
	if mt := s.tabOfSession(sid); mt != nil {
		return mt.tid
	}
	return s.fr.fsess[sid]
}

// prepareMasterSession makes a master session report its OOPIFs and keeps
// the fingerprint's UA override intact while we are attached.
func (s *syncer) prepareMasterSession(sid string) {
	autoAttachIframes(s.mconn, sid, false)
	guardUA(s.mconn, s.mws, sid)
	s.frameOp(func(fr *frameReg) { fr.treeDirty[sid] = true })
}

// onMasterFrameEvent handles the master's iframe-related CDP events (on the
// connection's event goroutine). It reports whether the event is fully
// handled; a tab's own navigation is left to the main handler.
func (s *syncer) onMasterFrameEvent(m cdpMsg) bool {
	switch m.Method {
	case "Target.attachedToTarget":
		var p struct {
			SessionID  string `json:"sessionId"`
			TargetInfo struct {
				TargetID string `json:"targetId"`
				Type     string `json:"type"`
				Parent   string `json:"parentFrameId"`
			} `json:"targetInfo"`
			Waiting bool `json:"waitingForDebugger"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.TargetInfo.Type != "iframe" || m.Session == "" {
			return true
		}
		c := s.mconn
		_, _ = c.call(p.SessionID, "Page.enable", nil)
		_, _ = c.call(p.SessionID, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": syncCaptureScript(), "worldName": syncWorld, "runImmediately": true})
		s.prepareMasterSession(p.SessionID)
		if p.Waiting {
			_, _ = c.call(p.SessionID, "Runtime.runIfWaitingForDebugger", nil)
		}
		parentSid, ti := m.Session, p.TargetInfo
		s.frameOp(func(fr *frameReg) {
			if tab := s.tabIDOfSession(parentSid); tab != "" {
				fr.fsess[p.SessionID] = tab
				fr.frames[ti.TargetID] = &cdpFrame{id: ti.TargetID, sid: p.SessionID, parent: ti.Parent, parentSid: parentSid, tab: tab, oopif: true}
			}
		})
		return true
	case "Target.detachedFromTarget":
		var p struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(m.Params, &p)
		s.frameOp(func(fr *frameReg) {
			if fr.fsess[p.SessionID] != "" {
				fr.sessionGone(p.SessionID)
			}
		})
		return true
	case "Page.frameAttached":
		sid := m.Session
		s.frameOp(func(fr *frameReg) { fr.treeDirty[sid] = true })
		return true
	case "Page.frameDetached":
		var p struct {
			FrameID string `json:"frameId"`
			Reason  string `json:"reason"`
		}
		_ = json.Unmarshal(m.Params, &p)
		sid := m.Session
		s.frameOp(func(fr *frameReg) {
			if f := fr.frames[p.FrameID]; f != nil && (p.Reason != "swap" || !f.oopif) {
				fr.frameGone(p.FrameID)
			}
			fr.treeDirty[sid] = true
		})
		return true
	case "Page.frameNavigated":
		var p struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
			} `json:"frame"`
		}
		_ = json.Unmarshal(m.Params, &p)
		sid := m.Session
		s.frameOp(func(fr *frameReg) {
			if f := fr.frames[p.Frame.ID]; f != nil {
				f.ctx = 0
			}
			fr.treeDirty[sid] = true
		})
		// A master tab's own navigation goes on to the main handler.
		return p.Frame.ParentID != "" || s.tabOfSession(sid) == nil
	}
	return false
}

// drainFrames drains the capture queues of a master tab's iframes.
func (s *syncer) drainFrames(mt *masterTab) {
	if s.fr == nil {
		return
	}
	for sid := range s.fr.treeDirty {
		if s.tabIDOfSession(sid) == mt.tid {
			delete(s.fr.treeDirty, sid)
			s.fr.refreshTree(sid)
		}
	}
	for _, f := range s.fr.frames {
		if f.tab == mt.tid {
			s.drainFrame(mt, f)
		}
	}
}

func (s *syncer) drainFrame(mt *masterTab, f *cdpFrame) {
	if time.Now().Before(f.retryAt) {
		return
	}
	if f.ctx == 0 {
		id, err := recWorldCtxNamed(s.mconn, f.sid, f.id, syncWorld)
		if err != nil {
			f.retryAt = time.Now().Add(400 * time.Millisecond)
			return
		}
		f.ctx = id
	}
	path, ok := s.fr.framePath(f, 0)
	if !ok {
		f.retryAt = time.Now().Add(300 * time.Millisecond)
		return
	}
	raw, err := s.mconn.call(f.sid, "Runtime.evaluate", map[string]any{"expression": syncDrainExpr(), "contextId": f.ctx, "returnByValue": true})
	var v string
	if err == nil {
		err = decodeEvalResult(raw, &v)
	}
	if err != nil {
		f.ctx = 0
		f.retryAt = time.Now().Add(200 * time.Millisecond)
		return
	}
	if len(v) < 3 {
		return
	}
	var evs []syncEvent
	if json.Unmarshal([]byte(v[2:]), &evs) != nil {
		return
	}
	var top *frameCtx
	var tw, th float64
	for _, ev := range evs {
		if ev.T == "" || ev.T == "nav" {
			continue
		}
		ev.tab, ev.Frame = mt.tid, path
		if ev.T == "click" || ev.T == "scroll" {
			// Top-level fallback point for followers without this frame.
			if top == nil {
				d := &cdpDriver{conn: s.mconn, sid: mt.sid, target: mt.tid, ws: s.mws, isoTop: true}
				top, _ = d.resolveFrame(parseFramePath(path), false)
				if top != nil {
					var wh []float64
					if raw, err := s.mconn.call(mt.sid, "Runtime.evaluate", map[string]any{"expression": "[innerWidth, innerHeight]", "contextId": top0World(d, mt.tid), "returnByValue": true}); err == nil && decodeEvalResult(raw, &wh) == nil && len(wh) == 2 {
						tw, th = wh[0], wh[1]
					}
				}
			}
			if top != nil && tw > 0 && th > 0 {
				x, y := ev.X, ev.Y
				if ev.T == "scroll" {
					x, y = ev.VX*ev.FW, ev.VY*ev.FH // frame-local wheel point
				}
				ev.TVX, ev.TVY = (top.ox+x*top.scale)/tw, (top.oy+y*top.scale)/th
			}
		}
		s.dispatch(ev)
	}
}

// top0World is the isolated world d uses in the top document.
func top0World(d *cdpDriver, tid string) int {
	w, _ := d.isolatedWorld(d.sid, tid)
	return w
}

// ---------- follower side ----------

// frameDriver returns the follower tab's frame resolver.
func (f *syncFollower) frameDriver() *cdpDriver {
	if f.cur.drv == nil || f.cur.drv.sid != f.cur.sid {
		f.cur.drv = &cdpDriver{conn: f.conn, sid: f.cur.sid, target: f.cur.tid, ws: f.ws, isoTop: true, implicit: 2 * time.Second}
	}
	return f.cur.drv
}

// topPoint turns a top-level viewport fraction into follower coordinates.
func (f *syncFollower) topPoint(vx, vy float64) (float64, float64, bool) {
	if vx <= 0 || vy <= 0 {
		return 0, 0, false
	}
	v, err := f.eval(`[innerWidth, innerHeight]`)
	if err != nil {
		return 0, 0, false
	}
	var p []float64
	if json.Unmarshal(v, &p) != nil || len(p) != 2 {
		return 0, 0, false
	}
	return vx * p[0], vy * p[1], true
}

// framePoint resolves an event inside a follower iframe to top-level
// coordinates: the element by selector (relative position kept), else the
// frame-local viewport fraction. ok=false: the frame is not there.
func (f *syncFollower) framePoint(ev syncEvent, useSel bool) (x, y float64, fc *frameCtx, ok bool) {
	d := f.frameDriver()
	path := parseFramePath(ev.Frame)
	var err error
	if fc, err = d.resolveFrame(path, true); err != nil {
		// A slow follower may still be loading the frame: wait briefly.
		deadline := time.Now().Add(1500 * time.Millisecond)
		for err != nil && time.Now().Before(deadline) {
			time.Sleep(150 * time.Millisecond)
			fc, err = d.resolveFrame(path, true)
		}
		if err != nil {
			return 0, 0, nil, false
		}
	}
	var p []float64
	if useSel && ev.Sel != "" {
		expr := fmt.Sprintf(`(() => { const e = document.querySelector(%s); if (!e) return null; e.scrollIntoView({block: 'nearest', inline: 'nearest'}); const r = e.getBoundingClientRect(); return [r.left + r.width * %g, r.top + r.height * %g]; })()`, jsStr(ev.Sel), ev.RX, ev.RY)
		if raw, err := d.evalInFrame(fc, expr, false); err == nil {
			_ = decodeEvalResult(raw, &p)
		}
		if len(p) == 2 {
			// The element may have scrolled the frame: measure the frame again.
			if fc2, err := d.resolveFrame(path, false); err == nil {
				fc = fc2
			}
		}
	}
	if len(p) != 2 {
		var wh []float64
		if raw, err := d.evalInFrame(fc, `[innerWidth, innerHeight]`, false); err == nil {
			_ = decodeEvalResult(raw, &wh)
		}
		if len(wh) != 2 || ev.VX <= 0 || ev.VY <= 0 {
			return 0, 0, fc, false
		}
		p = []float64{ev.VX * wh[0], ev.VY * wh[1]}
	}
	return fc.ox + p[0]*fc.scale, fc.oy + p[1]*fc.scale, fc, true
}

// applyFrame replays an event captured inside a master iframe. handled=false
// leaves the event to the top-level code (keys go to the focused frame).
func (f *syncFollower) applyFrame(ev syncEvent) (handled bool, err error) {
	switch ev.T {
	case "click":
		x, y, _, ok := f.framePoint(ev, true)
		if !ok {
			if x, y, ok = f.topPoint(ev.TVX, ev.TVY); !ok {
				return true, nil
			}
		}
		return true, f.clickAt(x, y, ev)
	case "scroll":
		x, y, _, ok := f.framePoint(ev, false)
		if !ok {
			if x, y, ok = f.topPoint(ev.TVX, ev.TVY); !ok {
				return true, nil
			}
		}
		_, err := f.call("Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": x, "y": y, "deltaX": ev.DX, "deltaY": ev.DY})
		return true, soft(err)
	case "select":
		d := f.frameDriver()
		fc, err := d.resolveFrame(parseFramePath(ev.Frame), false)
		if err != nil {
			return true, nil
		}
		_, err = d.evalInFrame(fc, fmt.Sprintf(`(() => { const e = document.querySelector(%s); if (!e) return false; e.value = %s; e.dispatchEvent(new Event('input', {bubbles: true})); e.dispatchEvent(new Event('change', {bubbles: true})); return true; })()`, jsStr(ev.Sel), jsStr(ev.Value)), false)
		return true, soft(err)
	case "text":
		if ev.Key != "" && ev.Key == ev.S {
			return false, nil // a key press: the page session routes it to the focused frame
		}
		// Inserted text (IME, paste) goes to the session owning the frame.
		d := f.frameDriver()
		fc, err := d.resolveFrame(parseFramePath(ev.Frame), false)
		if err != nil {
			return false, nil
		}
		_, err = f.conn.call(fc.sid, "Input.insertText", map[string]any{"text": ev.S})
		return true, soft(err)
	}
	return false, nil
}

// clickAt dispatches a real click at top-level coordinates.
func (f *syncFollower) clickAt(x, y float64, ev syncEvent) error {
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
}
