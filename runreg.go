package main

import (
	"os/exec"
	"sync"
)

type runEntry struct {
	cleanup func()
	state   *RunState
	cmd     *exec.Cmd
}

type runRegistry struct {
	mu     sync.Mutex
	m      map[string]runEntry
	onExit func(id string)
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

func (r *runRegistry) stop(id string) {
	r.mu.Lock()
	e, ok := r.m[id]
	r.mu.Unlock()
	if ok && e.cleanup != nil {
		e.cleanup()
		return
	}
	if ok && e.cmd.Process != nil {
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
	for _, s := range r.list() {
		r.stop(s.ProfileID)
	}
}
