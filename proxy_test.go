package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseProxyLine(t *testing.T) {
	cases := []struct {
		in, typ, host      string
		port               int
		user, pass, change string
	}{
		{"1.2.3.4:8080", "http", "1.2.3.4", 8080, "", "", ""},
		{"1.2.3.4:1080", "socks5", "1.2.3.4", 1080, "", "", ""},
		{"1.2.3.4:8080:alice:p:w", "http", "1.2.3.4", 8080, "alice", "p:w", ""},
		{"alice:pw:gate.smartproxy.com:7000", "http", "gate.smartproxy.com", 7000, "alice", "pw", ""},
		{"alice:pw@1.2.3.4:8080", "http", "1.2.3.4", 8080, "alice", "pw", ""},
		{"1.2.3.4:8080@alice:pw", "http", "1.2.3.4", 8080, "alice", "pw", ""},
		{"socks5://al%40ice:pw@1.2.3.4:9999", "socks5", "1.2.3.4", 9999, "al@ice", "pw", ""},
		{"socks5h://1.2.3.4:9999", "socks5", "1.2.3.4", 9999, "", "", ""},
		{"1.2.3.4 8080 alice pw", "http", "1.2.3.4", 8080, "alice", "pw", ""},
		{"socks5\t1.2.3.4\t5000\talice\tpw", "socks5", "1.2.3.4", 5000, "alice", "pw", ""},
		{"1.2.3.4:8080:alice:pw https://api.provider.com/changeip?key=x", "http", "1.2.3.4", 8080, "alice", "pw", "https://api.provider.com/changeip?key=x"},
		{"https,1.2.3.4,8443,alice,pw", "https", "1.2.3.4", 8443, "alice", "pw", ""},
		{"[2001:db8::1]:8080", "http", "2001:db8::1", 8080, "", "", ""},
		{"http://1.2.3.4:3128/", "http", "1.2.3.4", 3128, "", "", ""},
		{"1.2.3.4:8080 alice pw", "http", "1.2.3.4", 8080, "alice", "pw", ""},
	}
	for _, c := range cases {
		p, err := ParseProxyLine(c.in, "auto")
		if err != nil {
			t.Errorf("%q: %v", c.in, err)
			continue
		}
		if p.Type != c.typ || p.Host != c.host || p.Port != c.port || p.User != c.user || p.Pass != c.pass || p.ChangeIP != c.change {
			t.Errorf("%q -> %+v", c.in, p)
		}
	}
	for _, bad := range []string{"hello world", "1.2.3.4", "1.2.3.4:99999"} {
		if _, err := ParseProxyLine(bad, "auto"); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	pv := PreviewProxyText("type,host\n1.2.3.4:80\n# c\n1.2.3.4:80\nbad\n[{\"ip\":\"5.6.7.8\",\"port\":81}]", "http", []Proxy{{Host: "9.9.9.9", Port: 1}})
	if len(pv) != 4 || !pv[1].Dup || pv[2].OK {
		t.Errorf("preview: %+v", pv)
	}
	js := PreviewProxyText(`[{"ip":"5.6.7.8","port":81,"username":"u","password":"p","protocol":"socks5"},"1.1.1.1:2"]`, "http", nil)
	if len(js) != 2 || js[0].Proxy.Type != "socks5" || js[0].Proxy.User != "u" {
		t.Errorf("json: %+v", js)
	}
}

// A guessed protocol is corrected by the check: an HTTP proxy on port 1080.
func TestCheckFixesGuessedType(t *testing.T) {
	// fake HTTP proxy answering every request with an ip-api style body
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Host == "ip-api.com" {
			w.Write([]byte(`{"status":"success","query":"8.8.8.8","countryCode":"US","regionName":"California","city":"LA","timezone":"America/Los_Angeles","isp":"Comcast"}`))
			return
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	host, port, _ := net.SplitHostPort(u.Host)
	p, _ := ParseProxyLine(host+":"+port, "auto")
	p.Type, p.Guessed = "socks5", true
	r := CheckProxy(p)
	if r.Status != "ok" || r.Type != "http" || r.Country != "US" || r.ISP != "Comcast" || r.IPType != "residential" || r.Guessed {
		t.Fatalf("%+v", r)
	}
}
