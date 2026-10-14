package main

import (
	"net/url"
	"sync"
)

// UA override guard.
//
// Chromium keeps an Emulation.setUserAgentOverride per DevTools session, and
// verified on Chromium 155: once a second client (RPA runner, recorder,
// syncer) is attached to a page without its own override, the next document
// in that page reads the real navigator.platform ("Linux x86_64" under a
// Windows fingerprint) even though the fingerprint emulator's session still
// holds the override. Every helper session we keep attached to a page or an
// iframe target therefore re-applies the emulator's exact override.

var emuUA sync.Map // CDP host:port → map[string]any (UA override params)

func wsHostKey(ws string) string {
	u, err := url.Parse(ws)
	if err != nil {
		return ws
	}
	return u.Host
}

// registerEmuUA records the override the fingerprint emulator uses for the
// browser at ws (nil removes it).
func registerEmuUA(ws string, ua map[string]any) {
	if ua == nil {
		emuUA.Delete(wsHostKey(ws))
		return
	}
	emuUA.Store(wsHostKey(ws), ua)
}

// guardUA applies the emulator's UA override on our own session sid, if the
// browser at ws is fingerprinted by this process. Errors are ignored: the
// session may belong to a target type without the Emulation domain.
func guardUA(c *cdpConn, ws, sid string) {
	v, ok := emuUA.Load(wsHostKey(ws))
	if !ok || c == nil || sid == "" {
		return
	}
	ua := v.(map[string]any)
	_, _ = c.call(sid, "Emulation.setUserAgentOverride", ua)
	_, _ = c.call(sid, "Network.setUserAgentOverride", ua)
}
