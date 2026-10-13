package undetectable

import (
	"bufio"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fakeAuthProxy is a tiny HTTP proxy that requires user:pass and supports
// CONNECT and absolute-form GET.
func fakeAuthProxy(t *testing.T) (string, *int) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	hits := new(int)
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("al@ice:p:w"))
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				br := bufio.NewReader(c)
				req, err := http.ReadRequest(br)
				if err != nil {
					return
				}
				if req.Header.Get("Proxy-Authorization") != want {
					io.WriteString(c, "HTTP/1.1 407 Proxy Authentication Required\r\nContent-Length: 0\r\n\r\n")
					return
				}
				*hits++
				if req.Method == "CONNECT" {
					up, err := net.Dial("tcp", req.Host)
					if err != nil {
						return
					}
					io.WriteString(c, "HTTP/1.1 200 OK\r\n\r\n")
					go io.Copy(up, br)
					io.Copy(c, up)
					return
				}
				req.RequestURI = ""
				resp, err := http.DefaultTransport.RoundTrip(req)
				if err != nil {
					return
				}
				resp.Write(c)
			}(c)
		}
	}()
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), hits
}

func TestRelayHTTPUpstream(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "hello "+r.URL.Path) }))
	defer origin.Close()
	tls := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "secure") }))
	defer tls.Close()
	addr, hits := fakeAuthProxy(t)
	up := &url.URL{Scheme: "http", Host: addr, User: url.UserPassword("al@ice", "p:w")}
	if !NeedsRelay(up.String()) {
		t.Fatal("needs relay")
	}
	r, err := StartRelay(up.String())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	pu, _ := url.Parse(r.URL())
	tr := tls.Client().Transport.(*http.Transport).Clone()
	tr.Proxy = http.ProxyURL(pu)
	cl := &http.Client{Transport: tr}
	for _, c := range []struct{ u, want string }{{origin.URL + "/a", "hello /a"}, {origin.URL + "/b", "hello /b"}, {tls.URL, "secure"}} {
		resp, err := cl.Get(c.u)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(b) != c.want {
			t.Fatalf("%s: %q", c.u, b)
		}
	}
	if *hits < 2 {
		t.Fatalf("upstream hits %d", *hits)
	}
	// wrong password -> 502 from the relay, never a silent direct connection
	bad, _ := StartRelay(strings.Replace(up.String(), "p%3Aw", "nope", 1))
	defer bad.Close()
	bu, _ := url.Parse(bad.URL())
	tr2 := tls.Client().Transport.(*http.Transport).Clone()
	tr2.Proxy = http.ProxyURL(bu)
	if _, err := (&http.Client{Transport: tr2}).Get(tls.URL); err == nil {
		t.Fatal("expected failure with bad credentials")
	}
}
