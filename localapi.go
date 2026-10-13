package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

func newAPIKey() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return "gb_" + hex.EncodeToString(b)
}

// guard protects the API from web pages: a page in any browser can send
// requests to 127.0.0.1, so we require a loopback Host (blocks DNS rebinding),
// refuse requests a browser marks as cross-site (Origin / Sec-Fetch-Site), and
// require the API key unless the user turned that off.
func (a *localAPI) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(strings.ToLower(host), "[]")
		if host != "127.0.0.1" && host != "localhost" && host != "::1" {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" {
			http.Error(w, "browser pages may not call the local API", http.StatusForbidden)
			return
		}
		if sf := r.Header.Get("Sec-Fetch-Site"); sf != "" && sf != "none" && sf != "same-origin" {
			http.Error(w, "browser pages may not call the local API", http.StatusForbidden)
			return
		}
		st := a.store.GetSettings()
		if !st.APIKeyOff && r.URL.Path != "/status" {
			k := r.Header.Get("X-API-Key")
			if k == "" {
				if b := r.Header.Get("Authorization"); strings.HasPrefix(b, "Bearer ") {
					k = strings.TrimSpace(b[7:])
				}
			}
			if k == "" {
				k = r.URL.Query().Get("key")
			}
			if st.APIKey == "" || subtle.ConstantTimeCompare([]byte(k), []byte(st.APIKey)) != 1 {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"status":"error","error":"missing or wrong API key (X-API-Key header or ?key=)"}`))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Local automation API, Undetectable-style: list/create/launch/close profiles,
// so Puppeteer / Playwright / Selenium can attach to the CDP endpoint.

type localAPI struct {
	store *Store
	srv   *http.Server
	mu    sync.Mutex
}

var api = &localAPI{}

func (a *localAPI) start(store *Store) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.store = store
	st := store.GetSettings()
	if !st.APIEnabled {
		return
	}
	if a.srv != nil {
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/list", a.handleList)
	mux.HandleFunc("/profile/start", a.handleStart)
	mux.HandleFunc("/profile/stop", a.handleStop)
	mux.HandleFunc("/fingerprints", func(w http.ResponseWriter, r *http.Request) {
		lib, err := (&App{store: a.store}).NativePresets()
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		writeJSON(w, lib)
	})
	mux.HandleFunc("/profile/json", func(w http.ResponseWriter, r *http.Request) {
		raw, err := (&App{store: a.store}).ExportNativeJSON(r.URL.Query().Get("profile"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	})
	mux.HandleFunc("/profile/launch-json", a.handleLaunchJSON)
	mux.HandleFunc("/profile/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", 405)
			return
		}
		raw, err := (&App{store: a.store}).NativeSnapshot(r.URL.Query().Get("profile"))
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(raw))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "running": runReg.list()})
	})
	a.srv = &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", st.APIPort), Handler: a.guard(mux)}
	go func() { _ = a.srv.ListenAndServe() }()
}

func (a *localAPI) handleLaunchJSON(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}
	if a.store.GetSettings().KernelMode != "undetectable" {
		http.Error(w, "enable Undetectable kernel mode in settings first", 409)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	p, err := (&App{store: a.store}).ImportNativeJSON(string(raw))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if globalAuto != nil {
		globalAuto.BeforeLaunch(p.ID)
	}
	rs, err := a.store.Launch(p.ID)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "profileId": p.ID, "data": rs})
}

func (a *localAPI) restart(store *Store) {
	a.mu.Lock()
	if a.srv != nil {
		_ = a.srv.Close()
		a.srv = nil
	}
	a.mu.Unlock()
	a.start(store)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (a *localAPI) handleList(w http.ResponseWriter, r *http.Request) {
	type row struct {
		ID, Name, Status string
	}
	out := []row{}
	for _, p := range a.store.ListProfiles() {
		s := "stopped"
		if _, ok := runReg.get(p.ID); ok {
			s = "running"
		}
		out = append(out, row{p.ID, p.Name, s})
	}
	writeJSON(w, map[string]any{"data": out})
}

func (a *localAPI) handleStart(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("profile")
	if id == "" {
		id = r.URL.Query().Get("uuid")
	}
	if globalAuto != nil {
		globalAuto.BeforeLaunch(id)
	}
	rs, err := a.store.Launch(id)
	if err != nil {
		writeJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"status": "ok", "data": map[string]any{
		"port": rs.DebugPort, "websocket_link": rs.WSURL, "pid": rs.PID,
	}})
}

func (a *localAPI) handleStop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("profile")
	if id == "" {
		id = r.URL.Query().Get("uuid")
	}
	runReg.stop(id)
	writeJSON(w, map[string]any{"status": "ok"})
}
