package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// All fixtures are synthetic (documentation ranges / example.com); never put
// real subscription URLs, tokens or node data here.

const clashFixture = `mixed-port: 7890
allow-lan: true
mode: rule
proxies:
  - {name: "HTTP 01", type: http, server: 192.0.2.10, port: 8080, username: u1, password: p1}
  - {name: "HTTPS 01", type: http, server: 192.0.2.11, port: 443, tls: true, username: u2, password: p2}
  - {name: "SOCKS 01", type: socks5, server: 192.0.2.12, port: 1080, username: u3, password: p3}
  - name: "🇯🇵 VLESS Reality"
    type: vless
    server: node1.example.com
    port: 443
    uuid: 11111111-2222-3333-4444-555555555555
    tls: true
    servername: www.example.org
    flow: xtls-rprx-vision
    network: tcp
    client-fingerprint: chrome
    reality-opts: {public-key: TESTPUBLICKEY, short-id: abcd}
  - name: "VLESS WS"
    type: vless
    server: node2.example.com
    port: 443
    uuid: 22222222-2222-3333-4444-555555555555
    tls: true
    network: ws
    ws-opts: {path: /ws, headers: {Host: cdn.example.com}}
  - {name: "SS 01", type: ss, server: 198.51.100.1, port: 8388, cipher: aes-256-gcm, password: sspass}
  - {name: "HY2 01", type: hysteria2, server: 198.51.100.2, port: 35000, password: hypass, sni: www.example.org, skip-cert-verify: true}
  - {name: "Trojan 01", type: trojan, server: 198.51.100.3, port: 443, password: trpass, sni: t.example.org}
  - {name: "AnyTLS 01", type: anytls, server: 198.51.100.4, port: 19000, password: atpass}
  - {name: "VMess 01", type: vmess, server: 198.51.100.5, port: 443, uuid: 33333333-2222-3333-4444-555555555555, alterId: 0, cipher: auto}
  - {name: "TUIC 01", type: tuic, server: 198.51.100.6, port: 443, uuid: 44444444-2222-3333-4444-555555555555, password: tupass}
  - {name: "Direct", type: direct}
  - {name: "Broken", type: vless, port: 443}
proxy-groups:
  - {name: PROXY, type: select, proxies: ["HTTP 01"]}
rules:
  - MATCH,PROXY
`

func vmessLink() string {
	j, _ := json.Marshal(map[string]any{"v": "2", "ps": "VMess WS", "add": "198.51.100.20", "port": "443", "id": "55555555-2222-3333-4444-555555555555",
		"aid": "0", "scy": "auto", "net": "ws", "type": "none", "host": "v.example.com", "path": "/vm", "tls": "tls", "sni": "v.example.com"})
	return "vmess://" + base64.StdEncoding.EncodeToString(j)
}

func uriFixture() string {
	ssCred := base64.RawURLEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:sspw"))
	legacy := base64.StdEncoding.EncodeToString([]byte("aes-128-gcm:legacypw@198.51.100.31:8389"))
	return strings.Join([]string{
		"http://hu:hp@192.0.2.40:3128#HTTP%20URI",
		"https://su:sp@192.0.2.41:443#HTTPS%20URI",
		"socks5://ku:kp@192.0.2.42:1080#SOCKS%20URI",
		"ss://" + ssCred + "@198.51.100.30:8388#SS%20SIP002",
		"ss://" + legacy + "#SS%20legacy",
		"ss://" + ssCred + "@198.51.100.32:8390/?plugin=obfs-local%3Bobfs%3Dhttp%3Bobfs-host%3Dexample.com#SS%20obfs",
		"ssr://c29tZXRoaW5n",
		vmessLink(),
		"vless://66666666-2222-3333-4444-555555555555@198.51.100.40:443?security=reality&sni=www.example.org&fp=chrome&pbk=PBK&sid=ab&type=tcp&flow=xtls-rprx-vision#VLESS%20Reality",
		"vless://77777777-2222-3333-4444-555555555555@198.51.100.41:443?security=tls&type=ws&host=w.example.com&path=%2Fws#VLESS%20WS",
		"vless://88888888-2222-3333-4444-555555555555@198.51.100.42:443?security=tls&type=grpc&serviceName=svc#VLESS%20gRPC",
		"trojan://trpw@198.51.100.50:443?sni=t.example.org&allowInsecure=1#Trojan%20URI",
		"hysteria2://hy2pw@198.51.100.60:35000?sni=www.example.org&insecure=1&obfs=salamander&obfs-password=ob#HY2%20URI",
		"hy2://hy2pw2@198.51.100.61:35001?auth=x#HY2%20short",
		"tuic://99999999-2222-3333-4444-555555555555:tpw@198.51.100.70:443?congestion_control=bbr&alpn=h3&sni=t.example.org#TUIC%20URI",
		"anytls://atpw@198.51.100.80:19000?insecure=1&sni=a.example.org#AnyTLS%20URI",
		"# comment",
		"garbage line",
	}, "\n")
}

func protoCount(nodes []Proxy) map[string]int {
	m := map[string]int{}
	for _, n := range nodes {
		m[n.Protocol]++
	}
	return m
}

func TestParseClashSubscription(t *testing.T) {
	maps, format, skipped, err := parseSubscription([]byte(clashFixture))
	if err != nil || format != "clash" {
		t.Fatalf("parse: %v %s", err, format)
	}
	if skipped != 2 { // direct + broken
		t.Fatalf("skipped = %d", skipped)
	}
	nodes, protos, _ := buildSubNodes(maps, &Subscription{ID: "sub1", Name: "Test"})
	want := map[string]int{"http": 1, "https": 1, "socks5": 1, "vless": 2, "ss": 1, "hysteria2": 1, "trojan": 1, "anytls": 1, "vmess": 1, "tuic": 1}
	if fmt.Sprint(protos) != fmt.Sprint(want) {
		t.Fatalf("protocols = %v", protos)
	}
	for _, n := range nodes {
		if n.SubID != "sub1" || n.SubName != "Test" || n.NodeKey == "" || n.RawNode == "" {
			t.Fatalf("missing sub fields: %+v", n)
		}
		switch n.Protocol {
		case "http", "socks5":
			if n.needsCore() || n.User == "" || n.Pass == "" {
				t.Fatalf("direct node wrong: %+v", n)
			}
		default:
			if !n.needsCore() {
				t.Fatalf("%s should need the core", n.Protocol)
			}
		}
	}
	var vl map[string]any
	for _, n := range nodes {
		if n.Name == "VLESS WS" {
			_ = json.Unmarshal([]byte(n.RawNode), &vl)
		}
	}
	if vl["network"] != "ws" || vl["ws-opts"].(map[string]any)["headers"].(map[string]any)["Host"] != "cdn.example.com" {
		t.Fatalf("raw node lost nested options: %v", vl)
	}
}

func checkURINodes(t *testing.T, maps []map[string]any) {
	t.Helper()
	nodes, protos, _ := buildSubNodes(maps, &Subscription{ID: "u"})
	want := map[string]int{"http": 1, "https": 1, "socks5": 1, "ss": 3, "vmess": 1, "vless": 3, "trojan": 1, "hysteria2": 2, "tuic": 1, "anytls": 1}
	if fmt.Sprint(protos) != fmt.Sprint(want) {
		t.Fatalf("protocols = %v", protos)
	}
	byName := map[string]map[string]any{}
	for _, n := range nodes {
		var m map[string]any
		_ = json.Unmarshal([]byte(n.RawNode), &m)
		byName[n.Name] = m
	}
	chk := func(name, key string, want any) {
		t.Helper()
		m := byName[name]
		if m == nil {
			t.Fatalf("node %q missing (have %v)", name, keys(byName))
		}
		if fmt.Sprint(m[key]) != fmt.Sprint(want) {
			t.Fatalf("%s.%s = %v, want %v", name, key, m[key], want)
		}
	}
	chk("SS SIP002", "cipher", "chacha20-ietf-poly1305")
	chk("SS SIP002", "password", "sspw")
	chk("SS legacy", "server", "198.51.100.31")
	chk("SS legacy", "password", "legacypw")
	chk("SS obfs", "plugin", "obfs")
	chk("VMess WS", "network", "ws")
	chk("VMess WS", "tls", true)
	chk("VMess WS", "servername", "v.example.com")
	chk("VLESS Reality", "flow", "xtls-rprx-vision")
	chk("VLESS Reality", "servername", "www.example.org")
	if byName["VLESS Reality"]["reality-opts"].(map[string]any)["public-key"] != "PBK" {
		t.Fatal("reality public key")
	}
	chk("VLESS WS", "network", "ws")
	chk("VLESS gRPC", "network", "grpc")
	chk("Trojan URI", "skip-cert-verify", true)
	chk("HY2 URI", "obfs", "salamander")
	chk("HY2 URI", "password", "hy2pw")
	chk("HY2 short", "type", "hysteria2")
	chk("TUIC URI", "uuid", "99999999-2222-3333-4444-555555555555")
	chk("TUIC URI", "password", "tpw")
	chk("TUIC URI", "congestion-controller", "bbr")
	chk("AnyTLS URI", "password", "atpw")
	chk("HTTPS URI", "tls", true)
	chk("SOCKS URI", "username", "ku")
}

func keys(m map[string]map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestParseURIListPlainAndBase64(t *testing.T) {
	plain := uriFixture()
	maps, format, skipped, err := parseSubscription([]byte(plain))
	if err != nil || format != "uri" {
		t.Fatalf("plain: %v %s", err, format)
	}
	if skipped != 2 { // ssr + garbage
		t.Fatalf("skipped = %d", skipped)
	}
	checkURINodes(t, maps)
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawURLEncoding} {
		b64 := enc.EncodeToString([]byte(plain))
		// subscriptions often wrap base64 at 76 columns
		var wrapped strings.Builder
		for i := 0; i < len(b64); i += 76 {
			end := i + 76
			if end > len(b64) {
				end = len(b64)
			}
			wrapped.WriteString(b64[i:end] + "\r\n")
		}
		maps2, format2, _, err := parseSubscription([]byte(wrapped.String()))
		if err != nil || format2 != "uri" {
			t.Fatalf("base64: %v %s", err, format2)
		}
		checkURINodes(t, maps2)
	}
}

func TestParseSubscriptionRejects(t *testing.T) {
	if _, _, _, err := parseSubscription([]byte(`{"outbounds":[{"type":"vless"}]}`)); err == nil || !strings.Contains(err.Error(), "sing-box") {
		t.Fatalf("sing-box: %v", err)
	}
	if _, _, _, err := parseSubscription([]byte("   ")); err == nil {
		t.Fatal("empty accepted")
	}
	if _, _, _, err := parseSubscription([]byte("<html>login</html>")); err == nil {
		t.Fatal("html accepted")
	}
}

func TestSubHeaders(t *testing.T) {
	up, down, total, exp := parseUserInfo("upload=2019575151; download=10619960003; total=107374182400; expire=1798761600")
	if up != 2019575151 || down != 10619960003 || total != 107374182400 || exp != 1798761600 {
		t.Fatalf("userinfo %d %d %d %d", up, down, total, exp)
	}
	if n := nameFromDisposition(`inline; filename*=utf-8''My%20Airport%EF%BC%88%E7%B2%BE%E9%80%89%EF%BC%89.yaml`); n != "My Airport（精选）" {
		t.Fatalf("filename* = %q", n)
	}
	if n := nameFromDisposition(`attachment; filename*=utf-8''demo+%EF%BC%88+x%EF%BC%89.yaml`); n != "demo （ x）" {
		t.Fatalf("plus = %q", n)
	}
	if n := nameFromDisposition(`attachment; filename="Plain Name.txt"`); n != "Plain Name" {
		t.Fatalf("filename = %q", n)
	}
	h := http.Header{}
	if n := subNameFor("https://sub.example.com/x?token=1", h); n != "sub.example.com" {
		t.Fatalf("host fallback = %q", n)
	}
}

func TestNodeKeyStable(t *testing.T) {
	a := map[string]any{"type": "vless", "server": "Node.Example.com", "port": 443, "uuid": "x", "name": "A"}
	b := map[string]any{"type": "vless", "server": "node.example.com", "port": "443", "uuid": "x", "name": "B renamed", "network": "ws"}
	if nodeKey(a) != nodeKey(b) {
		t.Fatal("rename / transport change must keep the key")
	}
	c := map[string]any{"type": "vless", "server": "node.example.com", "port": 443, "uuid": "y"}
	if nodeKey(a) == nodeKey(c) {
		t.Fatal("different credential must change the key")
	}
	// duplicates in one source get #2
	nodes, _, _ := buildSubNodes([]map[string]any{a, b}, &Subscription{ID: "s"})
	if nodes[0].NodeKey == nodes[1].NodeKey || !strings.HasSuffix(nodes[1].NodeKey, "#2") {
		t.Fatalf("dup keys %s %s", nodes[0].NodeKey, nodes[1].NodeKey)
	}
}

func clashBody(nodes ...string) string {
	return "proxies:\n" + strings.Join(nodes, "\n") + "\n"
}

func TestSubscriptionUpsertAndServer(t *testing.T) {
	s := testStore(t)
	var body atomic.Value
	body.Store(clashBody(
		`  - {name: "A", type: vless, server: 198.51.100.1, port: 443, uuid: aaaa}`,
		`  - {name: "B", type: ss, server: 198.51.100.2, port: 8388, cipher: aes-256-gcm, password: b}`,
		`  - {name: "C", type: http, server: 192.0.2.3, port: 8080}`,
	))
	var gotUA atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA.Store(r.UserAgent())
		w.Header().Set("Content-Disposition", `attachment; filename*=utf-8''Test%20Sub.yaml`)
		w.Header().Set("Profile-Update-Interval", "12")
		w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=30; expire=1800000000")
		_, _ = w.Write([]byte(body.Load().(string)))
	}))
	defer srv.Close()

	pv, err := s.PreviewSubscription(srv.URL+"/sub?token=t", "")
	if err != nil || pv.Count != 3 || pv.CoreCount != 2 || pv.Name != "Test Sub" || pv.Interval != 12 {
		t.Fatalf("preview %+v %v", pv, err)
	}
	sub, res, err := s.AddSubscription(srv.URL+"/sub?token=t", "")
	if err != nil {
		t.Fatal(err)
	}
	if gotUA.Load() != subDefaultUA {
		t.Fatalf("UA %v", gotUA.Load())
	}
	if res.Added != 3 || sub.NodeCount != 3 || sub.AutoUpdateHours != 12 || sub.Name != "Test Sub" || sub.Total != 30 || sub.Expire != 1800000000 {
		t.Fatalf("add %+v %+v", sub, res)
	}
	if _, _, err := s.AddSubscription(srv.URL+"/sub?token=t", ""); err == nil {
		t.Fatal("duplicate URL accepted")
	}
	ids := map[string]string{}
	for _, p := range s.ListProxies() {
		ids[p.Name] = p.ID
		if p.SubID != sub.ID || p.SubName != "Test Sub" {
			t.Fatalf("node not grouped: %+v", p)
		}
	}
	// a profile uses B; B and C then leave the source, A is renamed, D is new
	pr := s.PutProfile(Profile{Name: "uses B", ProxyID: ids["B"]})
	body.Store(clashBody(
		`  - {name: "A renamed", type: vless, server: 198.51.100.1, port: 443, uuid: aaaa, network: ws}`,
		`  - {name: "D", type: trojan, server: 198.51.100.4, port: 443, password: d}`,
	))
	_, res, err = s.UpdateSubscription(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.Updated != 1 || res.Gone != 1 || res.Removed != 1 {
		t.Fatalf("refresh %+v", res)
	}
	if p, ok := s.GetProxy(ids["A"]); !ok || p.Name != "A renamed" {
		t.Fatalf("A must keep its ID: %+v %v", p, ok)
	}
	if p, ok := s.GetProxy(ids["B"]); !ok || !p.Gone {
		t.Fatalf("B used by a profile must stay, marked gone: %+v %v", p, ok)
	}
	if _, ok := s.GetProxy(ids["C"]); ok {
		t.Fatal("unused C must be deleted")
	}
	if got, _ := s.GetProfile(pr.ID); got.ProxyID != ids["B"] {
		t.Fatal("profile binding changed")
	}
	// B comes back: no longer gone, same ID
	body.Store(clashBody(
		`  - {name: "A renamed", type: vless, server: 198.51.100.1, port: 443, uuid: aaaa, network: ws}`,
		`  - {name: "B", type: ss, server: 198.51.100.2, port: 8388, cipher: aes-256-gcm, password: b}`,
	))
	if _, _, err := s.UpdateSubscription(sub.ID); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.GetProxy(ids["B"]); p == nil || p.Gone {
		t.Fatalf("B should be back: %+v", p)
	}
	// deterministic IDs: a second store adding the same source gets the same IDs
	s2 := testStore(t)
	s2.mu.Lock()
	s2.Subs = map[string]*Subscription{sub.ID: {ID: sub.ID, Name: sub.Name, URL: sub.URL, Enabled: true}}
	s2.mu.Unlock()
	if _, _, err := s2.UpdateSubscription(sub.ID); err != nil {
		t.Fatal(err)
	}
	if p, ok := s2.GetProxy(ids["A"]); !ok || p.Name != "A renamed" {
		t.Fatal("node IDs must be deterministic across computers")
	}

	// errors are recorded, nodes kept
	body.Store("<html>expired</html>")
	if _, _, err := s.UpdateSubscription(sub.ID); err == nil {
		t.Fatal("bad body accepted")
	}
	cur, _ := s.getSub(sub.ID)
	if cur.LastError == "" || len(s.ListProxies()) != 2 {
		t.Fatalf("error not recorded or nodes dropped: %+v", cur)
	}

	// edit renames the group
	cur.Name, cur.AutoUpdateHours = "Renamed", 0
	if _, err := s.EditSubscription(cur); err != nil {
		t.Fatal(err)
	}
	for _, p := range s.ListProxies() {
		if p.SubName != "Renamed" {
			t.Fatal("group label not renamed")
		}
	}
	if len(s.dueSubscriptions(time.Now().Add(1000*time.Hour))) != 0 {
		t.Fatal("interval 0 must disable auto-update")
	}

	// delete keeping nodes: they become manual proxies
	if _, err := s.DeleteSubscription(sub.ID, false); err != nil {
		t.Fatal(err)
	}
	for _, p := range s.ListProxies() {
		if p.SubID != "" || p.RawNode == "" {
			t.Fatalf("kept node wrong: %+v", p)
		}
	}
}

func TestSubscriptionDeleteNodes(t *testing.T) {
	s := testStore(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString([]byte(uriFixture()))))
	}))
	defer srv.Close()
	sub, _, err := s.AddSubscription(srv.URL, "B64")
	if err != nil || sub.NodeCount != 15 || sub.Format != "uri" || sub.Skipped != 2 {
		t.Fatalf("%+v %v", sub, err)
	}
	px := s.ListProxies()[0]
	pr := s.PutProfile(Profile{Name: "x", ProxyID: px.ID})
	if _, err := s.DeleteSubscription(sub.ID, true); err != nil {
		t.Fatal(err)
	}
	if len(s.ListProxies()) != 0 {
		t.Fatal("nodes not deleted")
	}
	if got, _ := s.GetProfile(pr.ID); got.ProxyID != "" {
		t.Fatal("profile should fall back to direct")
	}
}

func TestSubscriptionFetchGuards(t *testing.T) {
	big := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := []byte(strings.Repeat("a", 1<<20))
		for i := 0; i < 21; i++ {
			_, _ = w.Write(chunk)
		}
	}))
	defer big.Close()
	if _, err := fetchSubOnce(big.URL, "", nil); err == nil || !strings.Contains(err.Error(), "20 MB") {
		t.Fatalf("size cap: %v", err)
	}
	notFound := httptest.NewServer(http.NotFoundHandler())
	defer notFound.Close()
	if _, err := fetchSub(notFound.URL, ""); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("404: %v", err)
	}
	if _, err := validSubURL("ftp://x"); err == nil {
		t.Fatal("ftp accepted")
	}
}

func TestGlobalSyncCarriesSubscriptions(t *testing.T) {
	s := testStore(t)
	h0 := globalHash(s.globalPayload())
	s.mu.Lock()
	s.Subs = map[string]*Subscription{"a": {ID: "a", Name: "n", URL: "https://sub.example.com/a", AutoUpdateHours: 24, Enabled: true}}
	s.mu.Unlock()
	g := s.globalPayload()
	if len(g.Subs) != 1 {
		t.Fatal("subs missing from payload")
	}
	h1 := globalHash(g)
	if h0 == h1 {
		t.Fatal("adding a subscription must dirty the global hash")
	}
	s.mu.Lock()
	s.Subs["a"].LastUpdated, s.Subs["a"].NodeCount, s.Subs["a"].Download = 99, 5, 7
	s.mu.Unlock()
	if globalHash(s.globalPayload()) != h1 {
		t.Fatal("refresh results must not dirty the global hash")
	}
	// payloads without subs keep their old hash shape
	empty := globalPayload{Settings: Settings{}, Proxies: []Proxy{{ID: "x", Host: "h"}}}
	withNil := empty
	withNil.Subs = []Subscription{}
	if globalHash(empty) != globalHash(withNil) {
		t.Fatal("empty subs must not change the hash")
	}
}
