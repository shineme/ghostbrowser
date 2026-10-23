package main

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A front-end proxy (nginx client_max_body_size) rejects big bodies with an
// HTML 413 that ping's maxBody cannot predict; uploads must split themselves.
func TestChunkUploadSplitsOn413(t *testing.T) {
	const cap = 6 << 20
	got := map[string]bool{}
	var maxSeen int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("r")
		if path == "" {
			path = r.URL.Path
		}
		switch {
		case filepath.Base(path) == "ping":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "protocol": 2, "features": []string{"chunks"}, "maxBody": 64 << 20})
		case filepath.Base(path) == "missing":
			var in struct{ IDs []string }
			_ = json.NewDecoder(r.Body).Decode(&in)
			_ = json.NewEncoder(w).Encode(map[string]any{"missing": in.IDs})
		case r.Method == "PUT":
			b, _ := io.ReadAll(r.Body)
			if len(b) > cap {
				w.WriteHeader(413)
				_, _ = io.WriteString(w, "<html><head><title>413 Request Entity Too Large</title></head><body><center><h1>413 Request Entity Too Large</h1></center><hr><center>nginx</center></body></html>")
				return
			}
			if len(b) > maxSeen {
				maxSeen = len(b)
			}
			for len(b) >= 68 {
				id := string(b[:64])
				n := int(b[64])<<24 | int(b[65])<<16 | int(b[66])<<8 | int(b[67])
				got[id] = true
				b = b[68+n:]
			}
			_, _ = io.WriteString(w, `{"ok":true}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	d := t.TempDir()
	s := &Store{dir: d, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ProfilesDir: filepath.Join(d, "profiles"), SyncServer: srv.URL, SyncToken: "tok-tok-tok-tok-tok-tok", SyncPassword: "pw", DeviceID: "dev"}}
	p := s.PutProfile(Profile{Name: "x", Fingerprint: GenerateFingerprint("windows", "141")})
	dir := s.ProfileDir(p.ID)
	big := make([]byte, 14<<20) // 4 incompressible chunks, ~14 MB per 12 MB batch
	_, _ = rand.Read(big)
	_ = os.MkdirAll(dir, 0700)
	_ = os.WriteFile(filepath.Join(dir, "big.bin"), big, 0600)
	cp := *s.Profiles[p.ID]
	var evs []TransferEvent
	s.onTransfer = func(e TransferEvent) { evs = append(evs, e) }
	tr := s.newTransfer(p.ID, "up")
	cu, e := s.buildChunked(dir, s.Settings.ProfilesDir, &cp, "pw", tr, false)
	tr.finish(e)
	if e != nil {
		t.Fatal(e)
	}
	ids := cu.refs
	for _, id := range ids {
		if !got[id] {
			t.Fatalf("chunk %s never uploaded", id)
		}
	}
	if maxSeen > cap || s.bodyCap.Load() == 0 {
		t.Fatalf("maxSeen=%d cap learned=%d", maxSeen, s.bodyCap.Load())
	}
	var prev int64
	ups := 0
	for _, ev := range evs {
		if ev.Phase == "upload" {
			ups++
			if ev.Done < prev || ev.Total != 14<<20 {
				t.Fatalf("bad progress %+v", ev)
			}
			prev = ev.Done
		}
	}
	if last := evs[len(evs)-1]; ups < 1 || last.Phase != "done" || last.Done != 14<<20 || len(ActiveTransfers()) != 0 {
		t.Fatalf("ups=%d last=%+v", ups, last)
	}
	if !isTooLarge(syncStatus(&http.Response{StatusCode: 413, Body: io.NopCloser(strings.NewReader("<html>nginx</html>"))})) {
		t.Fatal("413 not classified")
	}
}
