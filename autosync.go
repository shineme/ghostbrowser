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
	Phase   string      `json:"phase"` // start | item | prompt | prompt-closed | done | error
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name,omitempty"`
	Action  string      `json:"action,omitempty"` // pull | push | skip
	Error   string      `json:"error,omitempty"`
	Pulled  int         `json:"pulled"`
	Pushed  int         `json:"pushed"`
	Failed  int         `json:"failed"`
	Message string      `json:"message,omitempty"`
	Prompt  *SyncPrompt `json:"prompt,omitempty"`
}

var globalAuto *autoSyncer

type autoSyncer struct {
	s    *Store
	emit func(SyncEvent)
	mu   sync.Mutex // one reconcile / push at a time per profile set
	busy map[string]bool
	bmu  sync.Mutex

	inflight sync.WaitGroup // pushes after a browser closed

	pmu    sync.Mutex // start-up prompt state
	prompt *SyncPrompt
	decide chan syncDecision
	hold   chan struct{}
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

// SyncItem is one profile the start-up comparison wants to move.
type SyncItem struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Action       string `json:"action"` // pull | push
	Reason       string `json:"reason"` // cloud_newer | missing_local | local_only | local_changed | keep_local
	Conflict     bool   `json:"conflict"`
	CloudVersion int64  `json:"cloudVersion"`
	LocalVersion int64  `json:"localVersion"`
	CloudUpdated int64  `json:"cloudUpdatedAt"`
	CloudDevice  string `json:"cloudDevice"`
	LocalRun     int64  `json:"localLastRun"`
	Size         int64  `json:"size"`
}

// SyncPrompt asks whether cloud copies should overwrite local ones.
type SyncPrompt struct {
	Items    []SyncItem `json:"items"`
	Seconds  int        `json:"seconds"`
	Action   string     `json:"action"`   // what happens when the countdown ends: pull | keep
	Deadline int64      `json:"deadline"` // unix ms
	Held     bool       `json:"held"`     // the user touched the dialog: no auto action
}

type syncDecision struct{ pull map[string]bool }

func (a *autoSyncer) plan() (prompt, auto []SyncItem, cloud map[string]cloudMeta, err error) {
	list, err := a.s.ListCloud()
	if err != nil {
		return nil, nil, nil, err
	}
	cloud = map[string]cloudMeta{}
	for _, c := range list {
		cloud[c.ID] = c
	}
	local := map[string]Profile{}
	for _, p := range a.s.ListProfiles() {
		local[p.ID] = p
	}
	for _, c := range list {
		p, ok := local[c.ID]
		it := SyncItem{ID: c.ID, Name: c.Name, Action: "pull", CloudVersion: c.Version, CloudUpdated: c.Updated, CloudDevice: c.Device, Size: c.Size}
		switch {
		case !ok:
			it.Reason = "missing_local" // nothing local to overwrite
			auto = append(auto, it)
		case p.CloudVer < c.Version && p.KeepLocal != c.Version:
			it.Name, it.LocalVersion, it.LocalRun = p.Name, p.CloudVer, p.LastRun
			it.Reason, it.Conflict = "cloud_newer", localDirty(p)
			prompt = append(prompt, it)
		}
	}
	for _, p := range local {
		c, ok := cloud[p.ID]
		it := SyncItem{ID: p.ID, Name: p.Name, Action: "push", LocalVersion: p.CloudVer, LocalRun: p.LastRun, CloudVersion: c.Version}
		switch {
		case !ok && p.CloudVer == 0:
			it.Reason = "local_only"
		case ok && p.CloudVer == c.Version && localDirty(p):
			it.Reason = "local_changed"
		case ok && p.KeepLocal == c.Version && p.CloudVer < c.Version:
			it.Reason = "keep_local"
		default:
			continue
		}
		auto = append(auto, it)
	}
	return prompt, auto, cloud, nil
}

// pushKeepingLocal uploads a profile; when the user chose to keep the local
// copy over a newer cloud version, the upload is based on that version so the
// local copy replaces it instead of failing with a conflict.
func (a *autoSyncer) push(id string) error {
	if p, ok := a.s.GetProfile(id); ok && p.KeepLocal > p.CloudVer {
		p.CloudVer = p.KeepLocal
		a.s.PutProfile(*p)
	}
	return a.s.PushProfile(id)
}

// Reconcile runs the start-up comparison. Pushes and downloads of profiles
// missing locally happen at once; when the cloud holds newer copies of local
// profiles the user is asked (with a countdown) whether to overwrite.
func (a *autoSyncer) Reconcile() SyncEvent {
	sum := SyncEvent{Phase: "done"}
	if !a.enabled() {
		sum.Message = "auto sync not configured"
		return sum
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.emit(SyncEvent{Phase: "start", Message: "正在对比云端与本地环境"})
	prompt, auto, _, err := a.plan()
	if err != nil {
		ev := SyncEvent{Phase: "error", Error: err.Error(), Message: "无法连接同步服务器，使用本地数据"}
		a.emit(ev)
		return ev
	}
	step := func(it SyncItem) {
		if _, running := runReg.get(it.ID); running || !a.claim(it.ID) {
			return
		}
		defer a.release(it.ID)
		f := a.s.PullProfile
		if it.Action == "push" {
			f = a.push
		}
		e := retry(3, f, it.ID)
		ev := SyncEvent{Phase: "item", ID: it.ID, Name: it.Name, Action: it.Action}
		if e != nil {
			ev.Error = e.Error()
			sum.Failed++
		} else if it.Action == "pull" {
			sum.Pulled++
		} else {
			sum.Pushed++
		}
		a.emit(ev)
	}
	for _, it := range auto {
		step(it)
	}
	if len(prompt) > 0 {
		pull := a.ask(prompt)
		for _, it := range prompt {
			if pull[it.ID] {
				step(it)
				continue
			}
			// Keep local: remember the choice and upload local over that cloud version.
			if p, ok := a.s.GetProfile(it.ID); ok {
				p.KeepLocal = it.CloudVersion
				a.s.PutProfile(*p)
				it.Action = "push"
				step(it)
			}
		}
	}
	sum.Message = fmt.Sprintf("同步完成：下载 %d，上传 %d，失败 %d", sum.Pulled, sum.Pushed, sum.Failed)
	a.emit(sum)
	return sum
}

// ask shows the start-up prompt and waits for the user, or for the countdown.
func (a *autoSyncer) ask(items []SyncItem) map[string]bool {
	st := a.s.GetSettings()
	all := func(on bool) map[string]bool {
		m := map[string]bool{}
		for _, it := range items {
			m[it.ID] = on
		}
		return m
	}
	def := st.SyncPromptAction != "keep"
	if st.SyncPromptSecs <= 0 || a.emit == nil {
		return all(def)
	}
	ch := make(chan syncDecision, 1)
	a.pmu.Lock()
	a.prompt = &SyncPrompt{Items: items, Seconds: st.SyncPromptSecs, Action: map[bool]string{true: "pull", false: "keep"}[def],
		Deadline: time.Now().Add(time.Duration(st.SyncPromptSecs) * time.Second).UnixMilli()}
	a.decide = ch
	pr := *a.prompt
	a.pmu.Unlock()
	a.emit(SyncEvent{Phase: "prompt", Message: fmt.Sprintf("云端有 %d 个环境比本地新", len(items)), Prompt: &pr})
	defer func() {
		a.pmu.Lock()
		a.prompt, a.decide = nil, nil
		a.pmu.Unlock()
		a.emit(SyncEvent{Phase: "prompt-closed"})
	}()
	for {
		a.pmu.Lock()
		held, deadline := a.prompt.Held, a.prompt.Deadline
		a.pmu.Unlock()
		wait := time.Until(time.UnixMilli(deadline))
		if held {
			wait = 30 * time.Minute // the user is looking at it; they decide
		}
		if wait < 0 {
			wait = 0
		}
		select {
		case d := <-ch:
			return d.pull
		case <-time.After(wait):
			a.pmu.Lock()
			held = a.prompt.Held
			a.pmu.Unlock()
			if !held || wait >= 30*time.Minute {
				return all(def)
			}
		case <-a.holdCh():
		}
	}
}

func (a *autoSyncer) holdCh() chan struct{} {
	a.pmu.Lock()
	defer a.pmu.Unlock()
	if a.hold == nil {
		a.hold = make(chan struct{}, 1)
	}
	return a.hold
}

// Pending returns the open start-up prompt (nil when none).
func (a *autoSyncer) Pending() *SyncPrompt {
	a.pmu.Lock()
	defer a.pmu.Unlock()
	if a.prompt == nil {
		return nil
	}
	p := *a.prompt
	return &p
}

// Hold stops the countdown: the user is choosing.
func (a *autoSyncer) Hold() {
	a.pmu.Lock()
	if a.prompt != nil {
		a.prompt.Held = true
	}
	a.pmu.Unlock()
	select {
	case a.holdCh() <- struct{}{}:
	default:
	}
}

// Decide answers the prompt: ids in pull are overwritten from the cloud, the
// rest keep their local copy.
func (a *autoSyncer) Decide(pull []string) bool {
	a.pmu.Lock()
	ch := a.decide
	a.pmu.Unlock()
	if ch == nil {
		return false
	}
	m := map[string]bool{}
	for _, id := range pull {
		m[id] = true
	}
	select {
	case ch <- syncDecision{pull: m}:
		return true
	default:
		return false
	}
}

// BeforeLaunch pulls a profile when the server has a newer version, unless
// the local copy has unsynced changes or the user chose to keep it.
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
		if c.ID != id || c.Version <= p.CloudVer || c.Version == p.KeepLocal {
			continue
		}
		if localDirty(*p) {
			a.emit(SyncEvent{Phase: "item", ID: id, Name: p.Name, Action: "skip", Message: "云端有更新版本，但本地也有未上传的改动，已使用本地。下次启动软件时会询问如何处理。"})
			continue
		}
		if a.claim(id) {
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
	a.inflight.Add(1)
	go func() {
		defer a.inflight.Done()
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
		e := retry(3, a.push, id)
		ev := SyncEvent{Phase: "item", ID: id, Name: name, Action: "push"}
		if e != nil {
			ev.Error = e.Error()
			if isConflict(e) {
				ev.Message = "云端已有更新的版本（可能在另一台电脑上用过），本次未上传。下次启动软件时会询问保留哪一份。"
			}
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

// Drain waits (bounded) for uploads started by closed browsers, so quitting
// the app right after closing a browser still saves its login state.
func (a *autoSyncer) Drain(max time.Duration) {
	if a == nil {
		return
	}
	done := make(chan struct{})
	go func() { a.inflight.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(max):
	}
}
