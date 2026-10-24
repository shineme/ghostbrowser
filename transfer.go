package main

import (
	"io"
	"sort"
	"sync"
	"time"
)

// TransferEvent is a live progress report for one profile upload or download,
// emitted to the UI as "sync:progress".
type TransferEvent struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Dir   string  `json:"dir"`   // up | down
	Phase string  `json:"phase"` // scan | pack | upload | commit | download | write | done | error
	Done  int64   `json:"done"`
	Total int64   `json:"total"` // 0 = unknown (indeterminate)
	Speed float64 `json:"speed"` // bytes per second, recent window
	ETA   int     `json:"eta"`   // seconds, -1 unknown
	Saved int64   `json:"saved"` // bytes skipped because the other side already had them (跳过，未变化)
	// Sent is what actually crossed the network (compressed + encrypted
	// chunks, the manifest, or the legacy archive); Done/Total of the upload
	// phase count the plain bytes of the changed pieces.
	Sent    int64  `json:"sent"`
	Started int64  `json:"started"`
	Error   string `json:"error,omitempty"`
	Final   bool   `json:"final"`
}

type speedSample struct {
	at time.Time
	n  int64
}

type transfer struct {
	mu      sync.Mutex
	s       *Store
	ev      TransferEvent
	moved   int64 // bytes moved across all phases, for the speed window
	samples []speedSample
	last    time.Time
}

var transfers = struct {
	sync.Mutex
	m map[string]*transfer
}{m: map[string]*transfer{}}

// newTransfer starts tracking; dir is "up" or "down".
func (s *Store) newTransfer(id, dir string) *transfer {
	name := id
	if p, ok := s.GetProfile(id); ok && p.Name != "" {
		name = p.Name
	}
	t := &transfer{s: s, ev: TransferEvent{ID: id, Name: name, Dir: dir, Phase: "scan", ETA: -1, Started: time.Now().UnixMilli()}}
	if dir == "down" {
		t.ev.Phase = "download"
	}
	transfers.Lock()
	transfers.m[id] = t
	transfers.Unlock()
	t.emit(true)
	return t
}

// ActiveTransfers lists transfers in progress (for a UI that opened late).
func ActiveTransfers() []TransferEvent {
	transfers.Lock()
	defer transfers.Unlock()
	out := make([]TransferEvent, 0, len(transfers.m))
	for _, t := range transfers.m {
		t.mu.Lock()
		out = append(out, t.ev)
		t.mu.Unlock()
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started < out[j].Started })
	return out
}

func (t *transfer) phase(p string, total int64) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.ev.Phase, t.ev.Done, t.ev.Total, t.ev.ETA = p, 0, total, -1
	t.mu.Unlock()
	t.emit(true)
}

func (t *transfer) saved(n int64) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.ev.Saved += n
	t.mu.Unlock()
}

// wire records n bytes that crossed the network.
func (t *transfer) wire(n int64) {
	if t == nil || n <= 0 {
		return
	}
	t.mu.Lock()
	t.ev.Sent += n
	t.mu.Unlock()
}

// add records n more bytes in the current phase.
func (t *transfer) add(n int64) {
	if t == nil || n == 0 {
		return
	}
	t.mu.Lock()
	t.ev.Done += n
	if t.ev.Total > 0 && t.ev.Done > t.ev.Total {
		t.ev.Done = t.ev.Total
	}
	t.moved += n
	t.mu.Unlock()
	t.emit(false)
}

// set moves the current phase to an absolute position.
func (t *transfer) set(done int64) {
	if t == nil {
		return
	}
	t.mu.Lock()
	d := done - t.ev.Done
	t.mu.Unlock()
	if d > 0 {
		t.add(d)
	}
}

func (t *transfer) finish(err error) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.ev.Final, t.ev.ETA = true, 0
	if err != nil {
		t.ev.Phase, t.ev.Error = "error", err.Error()
	} else {
		t.ev.Phase = "done"
		if t.ev.Total > 0 {
			t.ev.Done = t.ev.Total
		}
		if el := time.Since(time.UnixMilli(t.ev.Started)).Seconds(); el > 0.2 {
			t.ev.Speed = float64(t.moved) / el
		}
	}
	t.mu.Unlock()
	transfers.Lock()
	if transfers.m[t.ev.ID] == t {
		delete(transfers.m, t.ev.ID)
	}
	transfers.Unlock()
	t.emit(true)
}

// emit sends at most ~6 updates a second unless forced.
func (t *transfer) emit(force bool) {
	t.mu.Lock()
	now := time.Now()
	if !force && now.Sub(t.last) < 160*time.Millisecond {
		t.mu.Unlock()
		return
	}
	t.last = now
	t.samples = append(t.samples, speedSample{now, t.moved})
	cut := 0
	for cut < len(t.samples)-1 && now.Sub(t.samples[cut].at) > 4*time.Second {
		cut++
	}
	t.samples = t.samples[cut:]
	if f := t.samples[0]; !t.ev.Final && now.Sub(f.at) > 300*time.Millisecond {
		t.ev.Speed = float64(t.moved-f.n) / now.Sub(f.at).Seconds()
	}
	if !t.ev.Final {
		t.ev.ETA = -1
		if t.ev.Total > 0 && t.ev.Speed > 1 && (t.ev.Phase == "upload" || t.ev.Phase == "download") {
			t.ev.ETA = int(float64(t.ev.Total-t.ev.Done)/t.ev.Speed + 0.5)
		}
	}
	ev := t.ev
	fn := t.s.onTransfer
	t.mu.Unlock()
	if fn != nil {
		fn(ev)
	}
}

// countingReader reports bytes as they are read (request bodies, downloads).
type countingReader struct {
	r  io.Reader
	fn func(n int64)
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, e := c.r.Read(p)
	if n > 0 && c.fn != nil {
		c.fn(int64(n))
	}
	return n, e
}
