package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func init() { newKernelClient = func() *http.Client { return &http.Client{Transport: &http.Transport{}} } }

func fakeKernelZip(t *testing.T, extra map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"chrome-win/chrome.exe":              "MZ fake chrome",
		"chrome-win/150.0.7800.3.manifest":   "<assembly/>",
		"chrome-win/150.0.7800.3/chrome.dll": strings.Repeat("x", 300_000),
		"chrome-win/locales/zh-CN.pak":       "pak",
	}
	for k, v := range extra {
		files[k] = v
	}
	for name, body := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// snapshotServer serves LAST_CHANGE + <rev>/chrome-win.zip with Range support.
func snapshotServer(t *testing.T, data []byte, gate chan struct{}) (*httptest.Server, *int32) {
	var ranged int32
	mux := http.NewServeMux()
	mux.HandleFunc("/snap/LAST_CHANGE", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "1700001\n") })
	mux.HandleFunc("/snap/1700001/chrome-win.zip", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "" {
			atomic.AddInt32(&ranged, 1)
		}
		if gate != nil {
			// Stream slowly so cancel can land mid-download.
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			w.WriteHeader(200)
			w.Write(data[:len(data)/4])
			w.(http.Flusher).Flush()
			select {
			case <-gate:
			case <-r.Context().Done():
				return
			}
			w.Write(data[len(data)/4:])
			return
		}
		http.ServeContent(w, r, "chrome-win.zip", time.Time{}, bytes.NewReader(data))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &ranged
}

func waitKernel(t *testing.T, k *kernelInstaller) KernelProgress {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if s := k.snapshot(); !s.Running {
			return s
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("install did not finish")
	return KernelProgress{}
}

func TestKernelInstallProgressAndResult(t *testing.T) {
	data := fakeKernelZip(t, nil)
	srv, _ := snapshotServer(t, data, nil)
	root := t.TempDir()
	k := &kernelInstaller{st: KernelProgress{Phase: "idle"}}
	var mu sync.Mutex
	phases := map[string]bool{}
	var maxDone int64
	k.emit = func(p KernelProgress) {
		mu.Lock()
		phases[p.Phase] = true
		if p.Phase == "download" && p.Done > maxDone {
			maxDone = p.Done
		}
		mu.Unlock()
	}
	var got string
	if _, err := k.Start(root, "custom", srv.URL+"/snap/", func(p string) { got = p }); err != nil {
		t.Fatal(err)
	}
	s := waitKernel(t, k)
	if s.Phase != "done" {
		t.Fatalf("phase %s: %s %s", s.Phase, s.Message, s.Error)
	}
	want := filepath.Join(root, "chromium-150.0.7800.3", "chrome.exe")
	if got != want || s.Path != want || s.Version != "150.0.7800.3" || s.Revision != "1700001" {
		t.Fatalf("got path=%q state=%+v", got, s)
	}
	mu.Lock()
	for _, p := range []string{"resolve", "download", "extract", "verify", "done"} {
		if !phases[p] {
			t.Errorf("missing phase %s in events %v", p, phases)
		}
	}
	if maxDone != int64(len(data)) {
		t.Errorf("download progress reached %d of %d", maxDone, len(data))
	}
	mu.Unlock()
	if ents, _ := os.ReadDir(filepath.Join(root, ".tmp")); len(ents) != 0 {
		t.Errorf("temp not cleaned: %v", ents)
	}
	ks := listKernels(root, want)
	if len(ks) != 1 || !ks[0].Active || ks[0].Version != "150.0.7800.3" || ks[0].Size == 0 {
		t.Fatalf("list: %+v", ks)
	}
	if err := deleteKernel(root, filepath.Join(root, "..")); err == nil {
		t.Fatal("delete outside kernels dir must fail")
	}
	if err := deleteKernel(root, ks[0].Dir); err != nil {
		t.Fatal(err)
	}
	if len(listKernels(root, "")) != 0 {
		t.Fatal("kernel not deleted")
	}
}

func TestKernelInstallCancelThenResume(t *testing.T) {
	data := fakeKernelZip(t, nil)
	gate := make(chan struct{})
	srv, _ := snapshotServer(t, data, gate)
	root := t.TempDir()
	k := &kernelInstaller{st: KernelProgress{Phase: "idle"}}
	if _, err := k.Start(root, "custom", srv.URL+"/snap", nil); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500 && k.snapshot().Done == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := k.Start(root, "custom", srv.URL+"/snap", nil); err == nil {
		t.Fatal("second concurrent install must be refused")
	}
	k.Cancel()
	s := waitKernel(t, k)
	close(gate)
	if s.Phase != "cancelled" {
		t.Fatalf("phase %s %s", s.Phase, s.Error)
	}
	part := filepath.Join(root, ".tmp", "chromium-1700001.zip.part")
	fi, err := os.Stat(part)
	if err != nil || fi.Size() == 0 || fi.Size() >= int64(len(data)) {
		t.Fatalf("partial download should be kept: %v %v", fi, err)
	}

	// Retry against a Range-capable server resumes from the .part file.
	srv2, ranged := snapshotServer(t, data, nil)
	k2 := &kernelInstaller{st: KernelProgress{Phase: "idle"}}
	k2.Start(root, "custom", srv2.URL+"/snap/1700001/chrome-win.zip", nil)
	s = waitKernel(t, k2)
	if s.Phase != "done" || !s.Resumed || atomic.LoadInt32(ranged) == 0 {
		t.Fatalf("resume: %+v ranged=%d", s, atomic.LoadInt32(ranged))
	}
}

func TestKernelInstallRejectsZipSlip(t *testing.T) {
	for _, evil := range []string{"../evil.txt", "chrome-win/../../evil.txt", "/abs/evil.txt", "C:/evil.txt"} {
		data := fakeKernelZip(t, map[string]string{evil: "pwned"})
		srv, _ := snapshotServer(t, data, nil)
		root := t.TempDir()
		k := &kernelInstaller{st: KernelProgress{Phase: "idle"}}
		k.Start(root, "custom", srv.URL+"/snap/", nil)
		s := waitKernel(t, k)
		if s.Phase != "error" || !strings.Contains(s.Message, "不安全的路径") {
			t.Fatalf("%s: expected zip-slip rejection, got %+v", evil, s)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(root), "evil.txt")); err == nil {
			t.Fatalf("%s escaped the destination", evil)
		}
		if len(listKernels(root, "")) != 0 {
			t.Fatalf("%s: nothing should be installed", evil)
		}
	}
}

func TestKernelResolveMirrorListing(t *testing.T) {
	mux := http.NewServeMux()
	var base string
	mux.HandleFunc("/m/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/m/":
			fmt.Fprint(w, `[{"name":"990/","type":"dir"},{"name":"1000/","type":"dir"},{"name":"1001/","type":"dir"}]`)
		case "/m/1001/": // newest folder still uploading
			fmt.Fprint(w, `[]`)
		case "/m/1000/":
			fmt.Fprintf(w, `[{"name":"chrome-win.zip","size":123,"url":"%s/m/1000/chrome-win.zip"}]`, base)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	base = srv.URL
	u, rev, size, err := resolveSnapshotBase(context.Background(), srv.Client(), srv.URL+"/m/")
	if err != nil || rev != "1000" || size != 123 || u != srv.URL+"/m/1000/chrome-win.zip" {
		t.Fatalf("got %q %q %d %v", u, rev, size, err)
	}
}

func TestKernelErrHints(t *testing.T) {
	_, h := kernelErrHint(&httpErr{code: 403, what: "x"})
	if !strings.Contains(h, "国内镜像") {
		t.Fatal(h)
	}
	_, h = kernelErrHint(&netErr{errors.New("dial tcp: i/o timeout")})
	if !strings.Contains(h, "国内镜像") {
		t.Fatal(h)
	}
	m, _ := kernelErrHint(&diskErr{need: 2 << 30, free: 1 << 30})
	if !strings.Contains(m, "磁盘空间不足") {
		t.Fatal(m)
	}
}
