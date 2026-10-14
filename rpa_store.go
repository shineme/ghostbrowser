package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RPA workflows live in workflows.json next to data.json, and recent run
// records in rpa_runs.json, so the main data file stays small and other
// features' writes never race with workflow edits.

type rpaStore struct {
	mu        sync.Mutex
	dir       string
	workflows map[string]*Workflow
	runs      []*RpaRun // newest first
}

const rpaKeepRuns = 60

func newRpaStore(dir string) *rpaStore {
	r := &rpaStore{dir: dir, workflows: map[string]*Workflow{}}
	r.load()
	return r
}

func (r *rpaStore) wfFile() string   { return filepath.Join(r.dir, "workflows.json") }
func (r *rpaStore) runsFile() string { return filepath.Join(r.dir, "rpa_runs.json") }

func (r *rpaStore) load() {
	var d struct {
		Workflows []*Workflow `json:"workflows"`
	}
	if b, err := os.ReadFile(r.wfFile()); err == nil && json.Unmarshal(b, &d) == nil {
		for _, w := range d.Workflows {
			if w != nil && w.ID != "" {
				w.Steps = normaliseSteps(w.Steps, false)
				r.workflows[w.ID] = w
			}
		}
	}
	if b, err := os.ReadFile(r.runsFile()); err == nil {
		var runs []*RpaRun
		if json.Unmarshal(b, &runs) == nil {
			for _, run := range runs {
				// A run that was in flight when the app quit can never finish.
				if run.Status == "running" {
					run.Status = "cancelled"
					for _, t := range run.Targets {
						if t.Status == "running" || t.Status == "pending" {
							t.Status, t.Error = "cancelled", "程序退出时中断"
						}
					}
				}
			}
			r.runs = runs
		}
	}
}

func writeJSONAtomic(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (r *rpaStore) saveWorkflowsLocked() error {
	list := make([]*Workflow, 0, len(r.workflows))
	for _, w := range r.workflows {
		list = append(list, w)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt < list[j].CreatedAt })
	return writeJSONAtomic(r.wfFile(), map[string]any{"workflows": list})
}

func (r *rpaStore) saveRunsLocked() error {
	if len(r.runs) > rpaKeepRuns {
		r.runs = r.runs[:rpaKeepRuns]
	}
	return writeJSONAtomic(r.runsFile(), r.runs)
}

func (r *rpaStore) List() []Workflow {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Workflow, 0, len(r.workflows))
	for _, w := range r.workflows {
		out = append(out, *w)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdatedAt > out[j].UpdatedAt })
	return out
}

func (r *rpaStore) Get(id string) (Workflow, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.workflows[id]
	if !ok {
		return Workflow{}, false
	}
	return *w, true
}

func (r *rpaStore) Put(w Workflow) (Workflow, error) {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return Workflow{}, fmt.Errorf("请填写流程名称")
	}
	w.Steps = normaliseSteps(w.Steps, false)
	if w.Vars == nil {
		w.Vars = map[string]string{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().Unix()
	if old, ok := r.workflows[w.ID]; ok && w.ID != "" {
		// Run statistics belong to the store, not the editor.
		w.CreatedAt, w.LastRunAt, w.LastStatus, w.LastRunID, w.RunCount = old.CreatedAt, old.LastRunAt, old.LastStatus, old.LastRunID, old.RunCount
	} else {
		if w.ID == "" {
			w.ID = newID()
		}
		w.CreatedAt = now
	}
	w.UpdatedAt = now
	cp := w
	r.workflows[w.ID] = &cp
	return w, r.saveWorkflowsLocked()
}

func (r *rpaStore) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workflows, id)
	return r.saveWorkflowsLocked()
}

func (r *rpaStore) markRun(wfID, runID, status string, at int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.workflows[wfID]
	if !ok {
		return
	}
	if status == "running" {
		w.RunCount++
	}
	w.LastRunAt, w.LastRunID, w.LastStatus = at, runID, status
	_ = r.saveWorkflowsLocked()
}

// ---------- runs ----------

func (r *rpaStore) addRun(run *RpaRun) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs = append([]*RpaRun{run}, r.runs...)
	_ = r.saveRunsLocked()
}

// updateRun applies fn under the lock; persist writes the file (used at
// target/run boundaries rather than on every step).
func (r *rpaStore) updateRun(id string, persist bool, fn func(*RpaRun)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, run := range r.runs {
		if run.ID == id {
			fn(run)
			if persist {
				_ = r.saveRunsLocked()
			}
			return
		}
	}
}

func (r *rpaStore) Run(id string) (RpaRun, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, run := range r.runs {
		if run.ID == id {
			return copyRun(run), true
		}
	}
	return RpaRun{}, false
}

// Runs returns run summaries (without step detail) for one workflow, or all.
func (r *rpaStore) Runs(wfID string) []RpaRun {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []RpaRun{}
	for _, run := range r.runs {
		if wfID == "" || run.WorkflowID == wfID {
			c := copyRun(run)
			for i := range c.Targets {
				c.Targets[i].Steps = nil
			}
			out = append(out, c)
		}
	}
	return out
}

func copyRun(run *RpaRun) RpaRun {
	b, _ := json.Marshal(run)
	var c RpaRun
	_ = json.Unmarshal(b, &c)
	return c
}
