package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func useTempCore(t *testing.T) {
	t.Helper()
	old := coreMgr
	coreMgr = &coreManager{dir: t.TempDir()}
	t.Cleanup(func() { coreMgr.StopAll(); coreMgr = old })
}

func TestBuildCoreConfig(t *testing.T) {
	nodes := map[string]Proxy{
		"b": {ID: "b", Name: "🇯🇵 B", RawNode: `{"name":"🇯🇵 B","type":"ss","server":"198.51.100.1","port":8388,"cipher":"aes-128-gcm","password":"x"}`},
		"a": {ID: "a", Name: "A", RawNode: `{"name":"A","type":"vless","server":"198.51.100.2","port":443,"uuid":"u"}`},
	}
	b, err := buildCoreConfig("127.0.0.1:9", "sec", nodes, map[string]int{"a": 20001, "b": 20002})
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["allow-lan"] != false || cfg["secret"] != "sec" || cfg["mixed-port"] != nil || cfg["tun"] != nil || cfg["dns"] != nil {
		t.Fatalf("unsafe config: %s", b)
	}
	ls := cfg["listeners"].([]any)
	px := cfg["proxies"].([]any)
	if len(ls) != 2 || len(px) != 2 {
		t.Fatal("listeners / proxies")
	}
	l0 := ls[0].(map[string]any)
	if l0["proxy"] != "gb-a" || l0["listen"] != "127.0.0.1" || l0["type"] != "mixed" || l0["port"].(float64) != 20001 {
		t.Fatalf("listener %v", l0)
	}
	if px[1].(map[string]any)["name"] != "gb-b" {
		t.Fatal("node names must be replaced")
	}
	if _, err := buildCoreConfig("x", "y", map[string]Proxy{"c": {ID: "c", RawNode: "{bad"}}, map[string]int{"c": 1}); err == nil {
		t.Fatal("broken raw node accepted")
	}
}

func TestPickCoreAsset(t *testing.T) {
	rel := ghRelease{Tag: "v1.19.0"}
	for _, n := range []string{"mihomo-windows-amd64-v1.19.0.zip", "mihomo-windows-amd64-compatible-v1.19.0.zip", "mihomo-windows-amd64-v1-go120-v1.19.0.zip",
		"mihomo-linux-amd64-compatible-v1.19.0.gz", "mihomo-linux-arm64-v1.19.0.gz", "mihomo-linux-amd64-v1.19.0.deb"} {
		rel.Assets = append(rel.Assets, struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
			Size int64  `json:"size"`
		}{n, "https://example.com/" + n, 1})
	}
	for _, c := range [][3]string{{"windows", "amd64", "mihomo-windows-amd64-compatible-v1.19.0.zip"}, {"linux", "amd64", "mihomo-linux-amd64-compatible-v1.19.0.gz"}, {"linux", "arm64", "mihomo-linux-arm64-v1.19.0.gz"}, {"darwin", "arm64", ""}} {
		if n, _, _ := pickCoreAsset(rel, c[0], c[1]); n != c[2] {
			t.Fatalf("%s/%s -> %q", c[0], c[1], n)
		}
	}
}

func TestExtractCore(t *testing.T) {
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	w, _ := zw.Create("mihomo-windows-amd64-compatible.exe")
	_, _ = w.Write([]byte("EXE"))
	_ = zw.Close()
	if b, err := extractCore("x.zip", zb.Bytes()); err != nil || string(b) != "EXE" {
		t.Fatalf("zip %q %v", b, err)
	}
	var gb bytes.Buffer
	gw := gzip.NewWriter(&gb)
	_, _ = gw.Write([]byte("ELF"))
	_ = gw.Close()
	if b, err := extractCore("x.gz", gb.Bytes()); err != nil || string(b) != "ELF" {
		t.Fatalf("gz %q %v", b, err)
	}
}

func TestCoreMissing(t *testing.T) {
	useTempCore(t)
	t.Setenv("GHOST_MIHOMO", "")
	node := Proxy{ID: "n", Name: "V", Type: "vless", Protocol: "vless", Host: "198.51.100.9", Port: 443, RawNode: `{"type":"vless","server":"198.51.100.9","port":443,"uuid":"u"}`}
	r := CheckProxy(node)
	if r.Status != "unknown" || r.Error != "需要内核" {
		t.Fatalf("check without core: %+v", r)
	}
	if _, err := resolveLaunchProxy("profile:x", &node); err == nil || !strings.Contains(err.Error(), "设置 → 代理内核") {
		t.Fatalf("launch without core: %v", err)
	}
	direct := Proxy{ID: "d", Type: "http", Protocol: "http", Host: "192.0.2.1", Port: 80, RawNode: `{"type":"http"}`}
	if got, err := resolveLaunchProxy("profile:y", &direct); err != nil || got.Host != "192.0.2.1" {
		t.Fatal("direct nodes must not need the core")
	}
	if st := coreMgr.Status(); st.Installed || st.Running {
		t.Fatalf("status %+v", st)
	}
}

// ---- live: GHOST_MIHOMO=/path/to/mihomo go test -run TestCoreLive ----

func get(t *testing.T, px Proxy, target string) (string, error) {
	t.Helper()
	cl := px.client(10 * time.Second)
	resp, err := cl.Get(target)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b), nil
}

// forwardProxy is a tiny plain-HTTP forward proxy (absolute-URI GET only).
func forwardProxy(hits *int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*hits++
		if r.Method == http.MethodConnect {
			dst, err := net.Dial("tcp", r.Host)
			if err != nil {
				w.WriteHeader(502)
				return
			}
			w.WriteHeader(200)
			hj, _ := w.(http.Hijacker)
			c, _, _ := hj.Hijack()
			go func() { _, _ = io.Copy(dst, c) }()
			_, _ = io.Copy(c, dst)
			c.Close()
			dst.Close()
			return
		}
		out, err := http.DefaultTransport.RoundTrip(&http.Request{Method: r.Method, URL: r.URL, Header: r.Header, Host: r.Host})
		if err != nil {
			w.WriteHeader(502)
			return
		}
		defer out.Body.Close()
		w.Header().Set("X-Via", "fwd")
		w.WriteHeader(out.StatusCode)
		_, _ = io.Copy(w, out.Body)
	}))
}

func TestCoreLive(t *testing.T) {
	bin := os.Getenv("GHOST_MIHOMO")
	if bin == "" {
		t.Skip("set GHOST_MIHOMO=/path/to/mihomo")
	}
	if _, err := coreVersion(bin); err != nil {
		t.Fatal(err)
	}
	useTempCore(t)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hello-core")) }))
	defer target.Close()

	// A second mihomo acts as a Shadowsocks server, so the client side runs a
	// real ss node with no outside dependency.
	ssPort := freePort()
	srvDir := t.TempDir()
	srvCfg := fmt.Sprintf(`{"allow-lan":false,"log-level":"warning","mode":"rule","listeners":[{"name":"ss-in","type":"shadowsocks","listen":"127.0.0.1","port":%d,"cipher":"aes-128-gcm","password":"gbtest-pass","udp":true}],"rules":["MATCH,DIRECT"]}`, ssPort)
	_ = os.WriteFile(filepath.Join(srvDir, "config.yaml"), []byte(srvCfg), 0o600)
	srv := exec.Command(bin, "-d", srvDir, "-f", filepath.Join(srvDir, "config.yaml"))
	srv.Stdout, srv.Stderr = io.Discard, io.Discard
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Process.Kill(); _ = srv.Wait() }()
	for i := 0; i < 100; i++ {
		if c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", ssPort)); err == nil {
			c.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	ssNode := Proxy{ID: "ss1", Name: "🇺🇸 SS live", Type: "ss", Protocol: "ss", Host: "127.0.0.1", Port: ssPort,
		RawNode: fmt.Sprintf(`{"name":"🇺🇸 SS live","type":"ss","server":"127.0.0.1","port":%d,"cipher":"aes-128-gcm","password":"gbtest-pass","udp":true}`, ssPort)}
	local, err := resolveLaunchProxy("profile:p1", &ssNode)
	if err != nil {
		t.Fatal(err)
	}
	if local.Type != "socks5" || local.Host != "127.0.0.1" || local.CoreOf != "ss1" || local.Name != ssNode.Name {
		t.Fatalf("local %+v", local)
	}
	if body, err := get(t, *local, target.URL); err != nil || body != "hello-core" {
		t.Fatalf("through ss node: %q %v", body, err)
	}
	httpLocal := *local
	httpLocal.Type = "http" // mixed listener speaks both
	if body, err := get(t, httpLocal, target.URL); err != nil || body != "hello-core" {
		t.Fatalf("mixed listener as http: %q %v", body, err)
	}
	st := coreMgr.Status()
	if !st.Running || st.Listeners != 1 || !st.Installed {
		t.Fatalf("status %+v", st)
	}

	// Second node (an http upstream run through the core) → reload; the first
	// listener keeps its port and keeps working.
	hits := 0
	fwd := forwardProxy(&hits)
	defer fwd.Close()
	fu, _ := url.Parse(fwd.URL)
	_, fport, _ := net.SplitHostPort(fu.Host)
	httpNode := Proxy{ID: "h1", Name: "HTTP via core", Type: "https", Protocol: "test-http", Host: "127.0.0.1",
		RawNode: fmt.Sprintf(`{"name":"x","type":"http","server":"127.0.0.1","port":%s}`, fport)}
	local2, err := resolveLaunchProxy("profile:p2", &httpNode)
	if err != nil {
		t.Fatal(err)
	}
	if body, err := get(t, *local2, target.URL); err != nil || body != "hello-core" || hits == 0 {
		t.Fatalf("through http node: %q %v hits=%d", body, err, hits)
	}
	if body, err := get(t, *local, target.URL); err != nil || body != "hello-core" {
		t.Fatalf("first listener after reload: %q %v", body, err)
	}
	pid := coreMgr.Status().PID

	// A node the core rejects fails alone; the running set is untouched.
	bad := Proxy{ID: "bad", Name: "bad", Type: "ss", Protocol: "ss", RawNode: `{"name":"bad","type":"ss","server":"127.0.0.1","port":1,"cipher":"no-such-cipher","password":"x"}`}
	if _, err := resolveLaunchProxy("profile:p3", &bad); err == nil {
		t.Fatal("bad node accepted")
	} else {
		t.Logf("bad node: %v", err)
	}
	if body, err := get(t, *local, target.URL); err != nil || body != "hello-core" {
		t.Fatalf("first listener after rejected node: %q %v", body, err)
	}
	if coreMgr.Status().PID != pid {
		t.Log("core restarted after the rejected node (allowed, ports kept)")
	}

	// Batch: CheckProxies-style lease of both, then per-node probes reuse it.
	release := coreBatch([]Proxy{ssNode, httpNode})
	if st := coreMgr.Status(); st.Leases != 3 {
		t.Fatalf("leases %+v", st)
	}
	release()

	coreMgr.Release("profile:p1")
	if !coreMgr.Status().Running {
		t.Fatal("core must keep running while p2 uses it")
	}
	if body, err := get(t, *local2, target.URL); err != nil || body != "hello-core" {
		t.Fatalf("p2 after p1 left: %q %v", body, err)
	}
	coreMgr.Release("profile:p2")
	if st := coreMgr.Status(); st.Running || st.Listeners != 0 {
		t.Fatalf("core must stop when unused: %+v", st)
	}
	// runReg.remove releases the profile lease
	if _, err := resolveLaunchProxy("profile:p9", &ssNode); err != nil {
		t.Fatal(err)
	}
	runReg.add("p9", &RunState{ProfileID: "p9"}, nil)
	runReg.remove("p9")
	if coreMgr.Status().Running {
		t.Fatal("profile exit must release its core lease")
	}
}
