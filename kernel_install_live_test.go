package main

import (
	"context"
	"net/http"
	"os"
	"testing"
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
