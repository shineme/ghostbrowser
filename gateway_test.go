package main

import "testing"

func TestRerollGatewaySession(t *testing.T) {
	for _, c := range []struct{ in, kind string }{
		{"abc123-zone-custom-region-US-st-California-session-12345678-sessTime-30", "session"},
		{"test123456-region-US-st-Louisiana-sid-e6CXdJDm-t-15", "sid"},
		{"plainuser", ""},
		{"abc-zone-custom-region-US", ""},
	} {
		if k := gatewaySessionKind(c.in); k != c.kind {
			t.Fatalf("%s: kind %q want %q", c.in, k, c.kind)
		}
		out, ok := rerollGatewaySession(c.in)
		if ok != (c.kind != "") {
			t.Fatalf("%s: ok=%v", c.in, ok)
		}
		if !ok {
			continue
		}
		if out == c.in || len(out) != len(c.in) || gatewaySessionKind(out) != c.kind {
			t.Fatalf("%s -> %s", c.in, out)
		}
	}
	u, _ := rerollGatewaySession("a-zone-custom-region-JP-session-11111111-sessTime-5")
	if m := gwSession.FindStringSubmatch(u); m == nil || len(m[2]) != 8 || m[2] == "11111111" {
		t.Fatal(u)
	}
	for _, ch := range gwSession.FindStringSubmatch(u)[2] {
		if ch < '0' || ch > '9' {
			t.Fatal("711 session must be digits:", u)
		}
	}
}
