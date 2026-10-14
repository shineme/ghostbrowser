package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// GHOST_KERNEL_LIVE=1 go test -run TestKernelResolveLive .
func TestKernelResolveLive(t *testing.T) {
	if os.Getenv("GHOST_KERNEL_LIVE") == "" {
		t.Skip("set GHOST_KERNEL_LIVE=1")
	}
	cl := kernelHTTPClient()
	u, rev, size, err := resolveKernel(context.Background(), cl, "mirror", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("mirror: rev=%s size=%d url=%s", rev, size, u)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("Range", "bytes=0-1023")
	resp, err := cl.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("range request: HTTP %d", resp.StatusCode)
	}
}

// GHOST_KERNEL_ZIP=/path/chrome-win.zip installs a real snapshot zip served locally.
func TestKernelInstallRealZipLive(t *testing.T) {
	zp := os.Getenv("GHOST_KERNEL_ZIP")
	if zp == "" {
		t.Skip("set GHOST_KERNEL_ZIP")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, zp) }))
	defer srv.Close()
	root := t.TempDir()
	k := &kernelInstaller{st: KernelProgress{Phase: "idle"}}
	k.Start(root, "custom", srv.URL+"/1710741/chrome-win.zip", nil)
	s := waitKernelLong(t, k)
	if s.Phase != "done" {
		t.Fatalf("%+v", s)
	}
	t.Logf("installed %s version %s", s.Path, s.Version)
	if m := chromeMajorFrom(filepath.Dir(s.Path)); m == "" {
		t.Fatal("major not detected")
	}
}

func waitKernelLong(t *testing.T, k *kernelInstaller) KernelProgress {
	for i := 0; i < 1100; i++ {
		if s := k.snapshot(); !s.Running {
			return s
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timeout")
	return KernelProgress{}
}
