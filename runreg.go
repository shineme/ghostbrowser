package main

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

type runEntry struct {
	cleanup func()
	state   *RunState
	cmd     *exec.Cmd
}

type runRegistry struct {
	mu         sync.Mutex
	m          map[string]runEntry
	onExit     func(id string)
	beforeStop func(id, ws string)
}

var runReg = &runRegistry{m: map[string]runEntry{}}

func (r *runRegistry) add(id string, s *RunState, c *exec.Cmd) {
	r.mu.Lock()
	r.m[id] = runEntry{state: s, cmd: c}
	r.mu.Unlock()
}

func (r *runRegistry) remove(id string) {
	r.mu.Lock()
	e, ok := r.m[id]
	delete(r.m, id)
	cb := r.onExit
	r.mu.Unlock()
	if ok && e.cleanup != nil {
		e.cleanup()
	}
	if ok && cb != nil {
		cb(id)
	}
}

func (r *runRegistry) list() []RunState {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []RunState{}
	for _, e := range r.m {
		out = append(out, *e.state)
	}
	return out
}

func (r *runRegistry) get(id string) (*RunState, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.m[id]
	if !ok {
		return nil, false
	}
	return e.state, true
}

// stop closes a browser gracefully so it flushes cookies and storage to disk
// (and so the sync after exit uploads the latest login state), then kills it
// if it has not exited after a few seconds.
func (r *runRegistry) stop(id string) {
	r.mu.Lock()
	e, ok := r.m[id]
	before := r.beforeStop
	r.mu.Unlock()
	if !ok {
		return
	}
	if before != nil && e.state != nil {
		before(id, e.state.WSURL)
	}
	if e.cleanup != nil {
		e.cleanup()
		return
	}
	if e.state != nil && strings.HasPrefix(e.state.WSURL, "ws") {
		if closeBrowser(e.state.WSURL) == nil {
			for i := 0; i < 40; i++ {
				if _, alive := r.get(id); !alive {
					return
				}
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
}

func (r *runRegistry) setCleanup(id string, fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.m[id]
	if ok {
		e.cleanup = fn
		r.m[id] = e
	}
}

func (r *runRegistry) stopAll() {
	var wg sync.WaitGroup
	for _, s := range r.list() {
		wg.Add(1)
		go func(id string) { defer wg.Done(); r.stop(id) }(s.ProfileID)
	}
	wg.Wait()
}
