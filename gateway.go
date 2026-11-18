package main

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

// Residential gateways (711Proxy, Cliproxy and the many providers sharing
// their syntax) choose the exit from parameters inside the username:
//
//	711Proxy  user-zone-custom-region-US-st-California-session-12345678-sessTime-30
//	Cliproxy  user-region-US-st-Louisiana-sid-e6CXdJDm-t-15
//
// The same session value keeps the same exit IP; a new one gets a new IP. So
// for these proxies "change IP" needs no provider URL: roll the session value.
var gwSession = regexp.MustCompile(`-(session|sid)-([A-Za-z0-9]+)`)

// gatewaySessionKind reports which session key a gateway username uses, or "".
func gatewaySessionKind(user string) string {
	if m := gwSession.FindStringSubmatch(user); m != nil {
		return m[1]
	}
	return ""
}

func randString(n int, alphabet string) string {
	b := make([]byte, n)
	for i := range b {
		k, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[k.Int64()]
	}
	return string(b)
}

// newGatewaySession makes a session value in the form the provider documents:
// 711Proxy wants 8 digits, Cliproxy shows 8 letters and digits.
func newGatewaySession(kind string) string {
	if kind == "session" {
		return randString(8, "0123456789")
	}
	return randString(8, "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789")
}

// rerollGatewaySession returns the username with a fresh session value.
func rerollGatewaySession(user string) (string, bool) {
	kind := gatewaySessionKind(user)
	if kind == "" {
		return user, false
	}
	done := false
	out := gwSession.ReplaceAllStringFunc(user, func(string) string {
		if done {
			return ""
		}
		done = true
		return "-" + kind + "-" + newGatewaySession(kind)
	})
	return out, true
}

// canRotate: a change-IP URL, or a gateway username with a session value.
func (p Proxy) canRotate() bool {
	return p.ChangeIP != "" || gatewaySessionKind(p.User) != ""
}
