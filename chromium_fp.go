package main

// Extra fingerprint + launch controls for the open-source kernel (chromium.go).
// Pure helpers live here so they can be unit tested without a browser:
// noise seed, launch flags, user start-up parameters and the CDP side of the
// port-scan protection / video blocking.

import (
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net"
	"net/url"
	"strings"
	"sync"
)

// effectiveSeed is the noise seed the page script uses. It mixes the profile ID
// into the stored NoiseSeed so two profiles never share noise even when their
// configs were copied, and stays stable for one profile across launches.
// random=true (randomFingerprintOnLaunch) gives a fresh seed every launch.
// The result is in [1, 2^31-1) so the JS side can hash it with 32-bit ints.
func effectiveSeed(profileID string, noiseSeed int64, random bool) int64 {
	var v uint64
	if random {
		var b [8]byte
		_, _ = crand.Read(b[:])
		v = binary.LittleEndian.Uint64(b[:])
	} else {
		h := fnv.New64a()
		h.Write([]byte(profileID))
		var b [8]byte
		binary.LittleEndian.PutUint64(b[:], uint64(noiseSeed))
		h.Write(b[:])
		v = h.Sum64()
	}
	s := int64(v % 2147483646)
	return s + 1
}

// fpMode resolves a tri-state mode (off | noise | real); "" follows the old bool.
func fpMode(mode string, old bool) string {
	switch mode {
	case "off", "noise", "real":
		return mode
	}
	if old {
		return "noise"
	}
	return "real"
}

// mediaMode resolves mediaDevicesMode; "" follows HideMedia (off) or real.
func mediaMode(fp Fingerprint) string {
	switch fp.MediaDevicesMode {
	case "real", "custom", "off":
		return fp.MediaDevicesMode
	}
	if fp.HideMedia {
		return "off"
	}
	return "real"
}

// launchFlags returns the command-line switches the fingerprint asks for:
// window geometry, GPU, image blocking, muting and WebGPU.
func launchFlags(fp Fingerprint) []string {
	var out []string
	switch fp.WindowMode {
	case "maximized":
		out = append(out, "--start-maximized")
	case "custom":
		w, h := fp.WindowWidth, fp.WindowHeight
		if w <= 0 || h <= 0 {
			win := windowFor(fp)
			if w <= 0 {
				w = win.w
			}
			if h <= 0 {
				h = win.h
			}
		}
		out = append(out, fmt.Sprintf("--window-size=%d,%d", w, h), fmt.Sprintf("--window-position=%d,%d", fp.WindowX, fp.WindowY))
	default:
		win := windowFor(fp)
		out = append(out, fmt.Sprintf("--window-size=%d,%d", win.w, win.h))
	}
	if fp.HardwareAcceleration != nil && !*fp.HardwareAcceleration {
		out = append(out, "--disable-gpu")
	}
	if fp.BlockImages {
		out = append(out, "--blink-settings=imagesEnabled=false")
	}
	if fp.BlockSound {
		out = append(out, "--mute-audio")
	}
	if fp.WebGPU == "off" {
		out = append(out, "--disable-blink-features=WebGPU")
	}
	return out
}

// listFlags take comma separated values and are merged instead of replaced.
var listFlags = map[string]bool{
	"--disable-features": true, "--enable-features": true,
	"--disable-blink-features": true, "--enable-blink-features": true,
	"--blink-settings": true,
}

// reservedFlags are owned by GhostBrowser; user parameters can't set them.
func reservedFlag(name string) bool {
	switch name {
	case "--user-data-dir", "--proxy-server", "--user-agent", "--proxy-pac-url", "--proxy-bypass-list":
		return true
	}
	return strings.HasPrefix(name, "--remote-debugging")
}

// splitArgs splits a command line, honouring "double" and 'single' quotes.
func splitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	in, quote := false, rune(0)
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote, in = r, true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if in {
				out = append(out, cur.String())
				cur.Reset()
				in = false
			}
		default:
			cur.WriteRune(r)
			in = true
		}
	}
	if in {
		out = append(out, cur.String())
	}
	return out
}

// parseStartupParams turns the user's free-form start-up parameters into
// switches. Only "--" switches are kept; reserved ones are returned as rejected.
func parseStartupParams(s string) (keep, rejected []string) {
	for _, a := range splitArgs(s) {
		a = strings.TrimSpace(a)
		if !strings.HasPrefix(a, "--") || len(a) < 3 {
			if a != "" {
				rejected = append(rejected, a)
			}
			continue
		}
		name, _, _ := strings.Cut(a, "=")
		if reservedFlag(strings.ToLower(name)) {
			rejected = append(rejected, a)
			continue
		}
		keep = append(keep, a)
	}
	return
}

// mergeFlags folds duplicate switches: comma-list switches (features, blink
// features, blink settings) are unioned, keeping their first position; any
// other repeated switch keeps its last value at its first position.
// --blink-settings entries are key=value, the later key wins.
func mergeFlags(args []string) []string {
	type slot struct {
		idx  int
		vals []string
	}
	pos := map[string]*slot{}
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			out = append(out, a)
			continue
		}
		name, val, _ := strings.Cut(a, "=")
		sl, seen := pos[name]
		if listFlags[name] {
			if !seen {
				sl = &slot{idx: len(out)}
				pos[name] = sl
				out = append(out, "")
			}
			for _, v := range strings.Split(val, ",") {
				if v = strings.TrimSpace(v); v == "" {
					continue
				}
				if name == "--blink-settings" {
					k, _, _ := strings.Cut(v, "=")
					for i, old := range sl.vals {
						if ok, _, _ := strings.Cut(old, "="); ok == k {
							sl.vals = append(sl.vals[:i], sl.vals[i+1:]...)
							break
						}
					}
					sl.vals = append(sl.vals, v)
					continue
				}
				dup := false
				for _, old := range sl.vals {
					dup = dup || old == v
				}
				if !dup {
					sl.vals = append(sl.vals, v)
				}
			}
			out[sl.idx] = name + "=" + strings.Join(sl.vals, ",")
			continue
		}
		if seen {
			out[sl.idx] = a
			continue
		}
		pos[name] = &slot{idx: len(out)}
		out = append(out, a)
	}
	// drop list switches that ended up empty
	res := out[:0]
	for _, a := range out {
		if n, v, _ := strings.Cut(a, "="); listFlags[n] && v == "" {
			continue
		}
		res = append(res, a)
	}
	return res
}

// ---------- CDP network guard: port-scan protection + video blocking ----------

type fpNet struct {
	scan      bool
	video     bool
	sound     bool
	whitelist map[string]bool // "port" or "host:port"
	mu        sync.Mutex
	topURL    map[string]string // session → main frame URL
	topFrame  map[string]string // session → main frame id
}

func newFpNet(fp Fingerprint) *fpNet {
	n := &fpNet{scan: fp.PortScanProtection, video: fp.BlockVideo, sound: fp.BlockSound,
		whitelist: parseWhitelist(fp.ScanWhitelist), topURL: map[string]string{}, topFrame: map[string]string{}}
	if !n.scan && !n.video {
		return nil
	}
	return n
}

func parseWhitelist(s string) map[string]bool {
	m := map[string]bool{}
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\r' || r == '\t' }) {
		f = strings.ToLower(strings.TrimSpace(f))
		f = strings.TrimPrefix(strings.TrimPrefix(f, "http://"), "https://")
		f = strings.TrimSuffix(f, "/")
		if f != "" {
			m[f] = true
		}
	}
	return m
}

func whitelistJS(s string) []string {
	out := []string{}
	for k := range parseWhitelist(s) {
		out = append(out, k)
	}
	return out
}

// isLocalHost: loopback, unspecified and localhost names.
func isLocalHost(h string) bool {
	h = strings.Trim(strings.ToLower(h), "[]")
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && (ip.IsLoopback() || ip.IsUnspecified())
}

func urlHostPort(raw string) (host, port string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", false
	}
	host, port = u.Hostname(), u.Port()
	if port == "" {
		switch u.Scheme {
		case "https", "wss":
			port = "443"
		default:
			port = "80"
		}
	}
	return strings.ToLower(host), port, true
}

func (n *fpNet) whitelisted(host, port string) bool {
	return n.whitelist[port] || n.whitelist[host+":"+port] || n.whitelist[host]
}

// allowLocal decides a request to a local address.
func (n *fpNet) allowLocal(sid, reqURL, resType, frameID string) bool {
	host, port, ok := urlHostPort(reqURL)
	if !ok || !isLocalHost(host) {
		return true
	}
	if n.whitelisted(host, port) {
		return true
	}
	n.mu.Lock()
	top, topFrame := n.topURL[sid], n.topFrame[sid]
	n.mu.Unlock()
	if resType == "Document" && (frameID == "" || frameID == topFrame) {
		return true // the user (or the page) opened a local page in the tab
	}
	if th, _, ok := urlHostPort(top); ok && isLocalHost(th) {
		return true // the page itself is local
	}
	return false
}

var audioExt = []string{".mp3", ".m4a", ".aac", ".ogg", ".oga", ".opus", ".wav", ".flac", ".weba"}

func (n *fpNet) patterns() []map[string]any {
	var p []map[string]any
	if n.scan {
		for _, h := range []string{"127.*", "localhost", "[::1]", "0.0.0.0", "*.localhost"} {
			p = append(p, map[string]any{"urlPattern": "*://" + h + "/*", "requestStage": "Request"},
				map[string]any{"urlPattern": "*://" + h + ":*", "requestStage": "Request"})
		}
	}
	if n.video {
		p = append(p, map[string]any{"urlPattern": "*", "resourceType": "Media", "requestStage": "Request"})
	}
	return p
}

// attach enables Fetch interception on a page / iframe session.
func (n *fpNet) attach(c *cdpConn, sid, targetID string, top bool) {
	if n == nil {
		return
	}
	n.mu.Lock()
	if top {
		n.topFrame[sid] = targetID // a page target's id is its main frame id
	}
	n.mu.Unlock()
	_, _ = c.call(sid, "Fetch.enable", map[string]any{"patterns": n.patterns()})
}

func (n *fpNet) onEvent(c *cdpConn, m cdpMsg) bool {
	if n == nil {
		return false
	}
	switch m.Method {
	case "Page.frameNavigated":
		var ev struct {
			Frame struct {
				ID, ParentID, URL string
			} `json:"frame"`
		}
		if json.Unmarshal(m.Params, &ev) == nil && ev.Frame.ParentID == "" {
			n.mu.Lock()
			n.topURL[m.Session] = ev.Frame.URL
			if n.topFrame[m.Session] == "" {
				n.topFrame[m.Session] = ev.Frame.ID
			}
			n.mu.Unlock()
		}
		return true
	case "Fetch.requestPaused":
		var ev struct {
			RequestID    string `json:"requestId"`
			ResourceType string `json:"resourceType"`
			FrameID      string `json:"frameId"`
			Request      struct {
				URL string `json:"url"`
			} `json:"request"`
		}
		if json.Unmarshal(m.Params, &ev) != nil {
			return true
		}
		allow := true
		if n.scan && !n.allowLocal(m.Session, ev.Request.URL, ev.ResourceType, ev.FrameID) {
			// The first navigation can commit before Page.enable, so the
			// frame URL may be unknown or stale: read it and decide again.
			n.refreshTop(c, m.Session)
			allow = n.allowLocal(m.Session, ev.Request.URL, ev.ResourceType, ev.FrameID)
		}
		if allow && n.video && ev.ResourceType == "Media" {
			// keep plain audio files unless sound is blocked as well
			allow = isAudioURL(ev.Request.URL) && !n.sound
		}
		if allow {
			_, _ = c.call(m.Session, "Fetch.continueRequest", map[string]any{"requestId": ev.RequestID})
		} else {
			_, _ = c.call(m.Session, "Fetch.failRequest", map[string]any{"requestId": ev.RequestID, "errorReason": "ConnectionRefused"})
		}
		return true
	}
	return false
}

func (n *fpNet) refreshTop(c *cdpConn, sid string) {
	raw, err := c.call(sid, "Page.getFrameTree", nil)
	if err != nil {
		return
	}
	var ft struct {
		FrameTree struct {
			Frame struct{ ID, URL string } `json:"frame"`
		} `json:"frameTree"`
	}
	if json.Unmarshal(raw, &ft) == nil && ft.FrameTree.Frame.URL != "" {
		n.mu.Lock()
		n.topURL[sid] = ft.FrameTree.Frame.URL
		if n.topFrame[sid] == "" {
			n.topFrame[sid] = ft.FrameTree.Frame.ID
		}
		n.mu.Unlock()
	}
}

func isAudioURL(u string) bool {
	p := strings.ToLower(u)
	if i := strings.IndexAny(p, "?#"); i >= 0 {
		p = p[:i]
	}
	for _, e := range audioExt {
		if strings.HasSuffix(p, e) {
			return true
		}
	}
	return false
}

// speechVoicesFor returns an OS-appropriate voice list for speechVoices=custom.
func speechVoicesFor(os string, langs []string) []map[string]any {
	v := func(name, lang string, local, def bool) map[string]any {
		uri := name
		return map[string]any{"name": name, "lang": lang, "voiceURI": uri, "localService": local, "default": def}
	}
	google := []map[string]any{
		v("Google Deutsch", "de-DE", false, false), v("Google US English", "en-US", false, false),
		v("Google UK English Female", "en-GB", false, false), v("Google UK English Male", "en-GB", false, false),
		v("Google español", "es-ES", false, false), v("Google español de Estados Unidos", "es-US", false, false),
		v("Google français", "fr-FR", false, false), v("Google हिन्दी", "hi-IN", false, false),
		v("Google Bahasa Indonesia", "id-ID", false, false), v("Google italiano", "it-IT", false, false),
		v("Google 日本語", "ja-JP", false, false), v("Google 한국의", "ko-KR", false, false),
		v("Google Nederlands", "nl-NL", false, false), v("Google polski", "pl-PL", false, false),
		v("Google português do Brasil", "pt-BR", false, false), v("Google русский", "ru-RU", false, false),
		v("Google 普通话（中国大陆）", "zh-CN", false, false), v("Google 粤語（香港）", "zh-HK", false, false),
		v("Google 國語（臺灣）", "zh-TW", false, false),
	}
	var local []map[string]any
	switch os {
	case "windows":
		local = []map[string]any{
			v("Microsoft David - English (United States)", "en-US", true, true),
			v("Microsoft Mark - English (United States)", "en-US", true, false),
			v("Microsoft Zira - English (United States)", "en-US", true, false),
		}
		lang := strings.ToLower(firstLang(langs))
		switch {
		case strings.HasPrefix(lang, "zh"):
			local = append(local, v("Microsoft Huihui - Chinese (Simplified, PRC)", "zh-CN", true, false), v("Microsoft Kangkang - Chinese (Simplified, PRC)", "zh-CN", true, false), v("Microsoft Yaoyao - Chinese (Simplified, PRC)", "zh-CN", true, false))
		case strings.HasPrefix(lang, "ja"):
			local = append(local, v("Microsoft Ayumi - Japanese (Japan)", "ja-JP", true, false), v("Microsoft Haruka - Japanese (Japan)", "ja-JP", true, false), v("Microsoft Ichiro - Japanese (Japan)", "ja-JP", true, false))
		case strings.HasPrefix(lang, "de"):
			local = append(local, v("Microsoft Hedda - German (Germany)", "de-DE", true, false), v("Microsoft Katja - German (Germany)", "de-DE", true, false), v("Microsoft Stefan - German (Germany)", "de-DE", true, false))
		case strings.HasPrefix(lang, "fr"):
			local = append(local, v("Microsoft Hortense - French (France)", "fr-FR", true, false), v("Microsoft Julie - French (France)", "fr-FR", true, false), v("Microsoft Paul - French (France)", "fr-FR", true, false))
		}
	case "macos":
		for _, n := range [][2]string{{"Samantha", "en-US"}, {"Alex", "en-US"}, {"Daniel", "en-GB"}, {"Karen", "en-AU"}, {"Moira", "en-IE"}, {"Rishi", "en-IN"}, {"Tessa", "en-ZA"}, {"Thomas", "fr-FR"}, {"Anna", "de-DE"}, {"Monica", "es-ES"}, {"Alice", "it-IT"}, {"Kyoko", "ja-JP"}, {"Yuna", "ko-KR"}, {"Tingting", "zh-CN"}, {"Sinji", "zh-HK"}, {"Meijia", "zh-TW"}, {"Milena", "ru-RU"}, {"Luciana", "pt-BR"}} {
			local = append(local, v(n[0], n[1], true, n[0] == "Samantha"))
		}
	case "android", "tizen", "webos":
		return []map[string]any{} // Android Chrome exposes no voices until TTS is queried
	case "chromeos":
		local = []map[string]any{v("Chrome OS US English", "en-US", true, true), v("Chrome OS UK English", "en-GB", true, false)}
	default: // linux: speech-dispatcher is usually absent
		return google
	}
	return append(local, google...)
}

// clampInt for user-entered device counts.
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
