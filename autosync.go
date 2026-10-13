package main

import (
	"fmt"
	"sync"
	"time"
)

// Automatic sync policy:
//   - On app start, when sync is configured, compare the VPS list with local
//     profiles. The server wins: anything newer (or missing) locally is pulled.
//     Local-only profiles and local changes on an up-to-date version are pushed.
//   - Before a profile starts, pull it first if the server has a newer version.
//   - Every time a browser closes, push that profile to the server.

type SyncEvent struct {
	Phase   string `json:"phase"` // start | item | done | error
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Action  string `json:"action,omitempty"` // pull | push | skip
	Error   string `json:"error,omitempty"`
	Pulled  int    `json:"pulled"`
	Pushed  int    `json:"pushed"`
	Failed  int    `json:"failed"`
	Message string `json:"message,omitempty"`
}

var globalAuto *autoSyncer

type autoSyncer struct {
	s    *Store
	emit func(SyncEvent)
	mu   sync.Mutex // one reconcile / push at a time per profile set
	busy map[string]bool
	bmu  sync.Mutex
}

func (a *autoSyncer) enabled() bool {
	st := a.s.GetSettings()
	return !st.AutoSyncOff && syncReady(st, true) == nil
}

func (a *autoSyncer) claim(id string) bool {
	a.bmu.Lock()
	defer a.bmu.Unlock()
	if a.busy[id] {
		return false
	}
	a.busy[id] = true
	return true
}

func (a *autoSyncer) release(id string) {
	a.bmu.Lock()
	delete(a.busy, id)
	a.bmu.Unlock()
}

// localDirty: the profile ran after its last successful sync.
func localDirty(p Profile) bool { return p.LastRun > p.SyncedAt }

// Reconcile runs the start-up comparison. Server versions take priority.
func (a *autoSyncer) Reconcile() SyncEvent {
	sum := SyncEvent{Phase: "done"}
	if !a.enabled() {
		sum.Message = "auto sync not configured"
		return sum
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.emit(SyncEvent{Phase: "start", Message: "正在对比云端与本地环境"})
	cloud, err := a.s.ListCloud()
	if err != nil {
		ev := SyncEvent{Phase: "error", Error: err.Error(), Message: "无法连接同步服务器，使用本地数据"}
		a.emit(ev)
		return ev
	}
	inCloud := map[string]cloudMeta{}
	for _, c := range cloud {
		inCloud[c.ID] = c
	}
	local := map[string]Profile{}
	for _, p := range a.s.ListProfiles() {
		local[p.ID] = p
	}
	step := func(id, name, action string, f func(string) error) {
		if _, running := runReg.get(id); running || !a.claim(id) {
			return
		}
		defer a.release(id)
		e := retry(3, f, id)
		ev := SyncEvent{Phase: "item", ID: id, Name: name, Action: action}
		if e != nil {
			ev.Error = e.Error()
			sum.Failed++
		} else if action == "pull" {
			sum.Pulled++
		} else {
			sum.Pushed++
		}
		a.emit(ev)
	}
	// 1. Server first: pull everything missing or newer locally.
	for _, c := range cloud {
		p, ok := local[c.ID]
		if !ok || p.CloudVer < c.Version {
			step(c.ID, c.Name, "pull", a.s.PullProfile)
		}
	}
	// 2. Then upload what only exists locally, or changed on the current version.
	for _, p := range local {
		c, ok := inCloud[p.ID]
		switch {
		case !ok && p.CloudVer == 0:
			step(p.ID, p.Name, "push", a.s.PushProfile)
		case ok && p.CloudVer == c.Version && localDirty(p):
			step(p.ID, p.Name, "push", a.s.PushProfile)
		}
	}
	sum.Message = fmt.Sprintf("同步完成：下载 %d，上传 %d，失败 %d", sum.Pulled, sum.Pushed, sum.Failed)
	a.emit(sum)
	return sum
}

// BeforeLaunch pulls a profile when the server has a newer version.
// Network problems never block the launch; the local copy is used.
func (a *autoSyncer) BeforeLaunch(id string) {
	if !a.enabled() {
		return
	}
	p, ok := a.s.GetProfile(id)
	if !ok {
		return
	}
	cloud, err := a.s.ListCloud()
	if err != nil {
		return
	}
	for _, c := range cloud {
		if c.ID == id && c.Version > p.CloudVer && a.claim(id) {
			e := a.s.PullProfile(id)
			a.release(id)
			ev := SyncEvent{Phase: "item", ID: id, Name: p.Name, Action: "pull"}
			if e != nil {
				ev.Error = e.Error()
			}
			a.emit(ev)
		}
	}
}

// AfterExit pushes a profile as soon as its browser has closed.
func (a *autoSyncer) AfterExit(id string) {
	if !a.enabled() {
		return
	}
	go func() {
		time.Sleep(1500 * time.Millisecond) // let Chrome release its file locks
		if !a.claim(id) {
			return
		}
		defer a.release(id)
		name := ""
		if p, ok := a.s.GetProfile(id); ok {
			name = p.Name
		} else {
			return // deleted
		}
		e := retry(3, a.s.PushProfile, id)
		ev := SyncEvent{Phase: "item", ID: id, Name: name, Action: "push"}
		if e != nil {
			ev.Error = e.Error()
		}
		a.emit(ev)
	}()
}

func retry(n int, f func(string) error, id string) error {
	var e error
	for i := 0; i < n; i++ {
		if e = f(id); e == nil {
			return nil
		}
		if isConflict(e) {
			return e // retrying cannot fix a version conflict
		}
		time.Sleep(time.Duration(i+1) * 2 * time.Second)
	}
	return e
}

func isConflict(e error) bool {
	return e != nil && len(e.Error()) >= 9 && e.Error()[:9] == "conflict:"
}

// syncStamp marks a profile as clean: never earlier than its last run, so a
// clock difference between devices cannot leave it looking dirty forever.
func syncStamp(lastRun int64) int64 {
	if now := time.Now().Unix(); now > lastRun {
		return now
	}
	return lastRun
}
