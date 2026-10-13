package undetectable

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	xproxy "golang.org/x/net/proxy"
)

// Chromium's --proxy-server cannot carry a username/password, and the native
// kernel has no extension to answer auth challenges. A Relay is a local,
// credential-free HTTP proxy on 127.0.0.1 that forwards every connection to the
// real upstream proxy with its credentials. Hostnames are passed through to the
// upstream (no local DNS lookups, so no DNS leak).
type Relay struct {
	Addr string
	ln   net.Listener
	dial func(ctx context.Context, addr string) (net.Conn, error)
	wg   sync.WaitGroup
	mu   sync.Mutex
	open map[net.Conn]struct{}
}

// NeedsRelay reports whether a proxy URL carries credentials.
func NeedsRelay(proxyURL string) bool {
	u, err := url.Parse(proxyURL)
	return err == nil && u.User != nil && u.User.Username() != ""
}

// StartRelay listens on 127.0.0.1 (random port) and forwards to upstream,
// e.g. "http://user:pass@1.2.3.4:8080" or "socks5://user:pass@1.2.3.4:1080".
func StartRelay(upstream string) (*Relay, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}
	r := &Relay{open: map[net.Conn]struct{}{}}
	switch strings.ToLower(u.Scheme) {
	case "socks5", "socks5h", "socks":
		var auth *xproxy.Auth
		if u.User != nil {
			pw, _ := u.User.Password()
			auth = &xproxy.Auth{User: u.User.Username(), Password: pw}
		}
		d, err := xproxy.SOCKS5("tcp", u.Host, auth, &net.Dialer{Timeout: 15 * time.Second})
		if err != nil {
			return nil, err
		}
		cd := d.(xproxy.ContextDialer)
		r.dial = func(ctx context.Context, addr string) (net.Conn, error) { return cd.DialContext(ctx, "tcp", addr) }
	case "http", "https", "":
		r.dial = func(ctx context.Context, addr string) (net.Conn, error) { return httpConnect(ctx, u, addr) }
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q", u.Scheme)
	}
	r.ln, err = net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	r.Addr = r.ln.Addr().String()
	r.wg.Add(1)
	go r.serve()
	return r, nil
}

// URL is what to pass to --proxy-server.
func (r *Relay) URL() string { return "http://" + r.Addr }

func (r *Relay) Close() {
	if r == nil {
		return
	}
	r.ln.Close()
	r.mu.Lock()
	for c := range r.open {
		c.Close()
	}
	r.mu.Unlock()
}

func (r *Relay) track(c net.Conn, on bool) {
	r.mu.Lock()
	if on {
		r.open[c] = struct{}{}
	} else {
		delete(r.open, c)
	}
	r.mu.Unlock()
}

func (r *Relay) serve() {
	defer r.wg.Done()
	for {
		c, err := r.ln.Accept()
		if err != nil {
			return
		}
		go r.handle(c)
	}
}

func (r *Relay) handle(c net.Conn) {
	r.track(c, true)
	defer func() { r.track(c, false); c.Close() }()
	br := bufio.NewReader(c)
	var tr *http.Transport
	for {
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		if req.Method == http.MethodConnect {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			up, err := r.dial(ctx, req.Host)
			cancel()
			if err != nil {
				io.WriteString(c, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
				return
			}
			r.track(up, true)
			io.WriteString(c, "HTTP/1.1 200 Connection Established\r\n\r\n")
			// Anything the client already pipelined sits in br.
			if n := br.Buffered(); n > 0 {
				b, _ := br.Peek(n)
				up.Write(b)
				br.Discard(n)
			}
			pipe(c, up)
			r.track(up, false)
			up.Close()
			return
		}
		// Plain http:// request in absolute form.
		if tr == nil {
			tr = &http.Transport{DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) { return r.dial(ctx, addr) }, DisableCompression: true, Proxy: nil}
			defer tr.CloseIdleConnections()
		}
		if req.URL.Host == "" {
			req.URL.Host = req.Host
		}
		if req.URL.Scheme == "" {
			req.URL.Scheme = "http"
		}
		req.RequestURI = ""
		req.Header.Del("Proxy-Connection")
		req.Header.Del("Proxy-Authorization")
		resp, err := tr.RoundTrip(req)
		if err != nil {
			io.WriteString(c, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
			return
		}
		err = resp.Write(c)
		resp.Body.Close()
		if err != nil || req.Close || resp.Close {
			return
		}
	}
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		io.Copy(dst, src)
		if tc, ok := dst.(interface{ CloseWrite() error }); ok {
			tc.CloseWrite()
		}
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
	// give the other direction a moment to drain, then tear down
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

// httpConnect opens a tunnel to addr through an HTTP proxy with Basic auth.
func httpConnect(ctx context.Context, u *url.URL, addr string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 15 * time.Second}
	c, err := d.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return nil, err
	}
	if dl, ok := ctx.Deadline(); ok {
		c.SetDeadline(dl)
	}
	req := "CONNECT " + addr + " HTTP/1.1\r\nHost: " + addr + "\r\n"
	if u.User != nil {
		pw, _ := u.User.Password()
		req += "Proxy-Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte(u.User.Username()+":"+pw)) + "\r\n"
	}
	req += "\r\n"
	if _, err = io.WriteString(c, req); err != nil {
		c.Close()
		return nil, err
	}
	br := bufio.NewReader(c)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		c.Close()
		return nil, err
	}
	if resp.StatusCode != 200 {
		c.Close()
		if resp.StatusCode == 407 {
			return nil, fmt.Errorf("upstream proxy: authentication failed (407)")
		}
		return nil, fmt.Errorf("upstream proxy: %s", resp.Status)
	}
	c.SetDeadline(time.Time{})
	if br.Buffered() > 0 { // unlikely, but keep any bytes already read
		return &bufConn{Conn: c, r: br}, nil
	}
	return c, nil
}

type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufConn) Read(p []byte) (int, error) { return b.r.Read(p) }
