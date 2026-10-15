package undetectable

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Overrider keeps every page session aligned with the profile fingerprint.
//
// startInfo already carries the bulk of the fingerprint (CPU/memory, screen,
// devicePixelRatio, WebGL strings, fonts, timezone, languages, storage quota,
// JS heap limit). Two groups of values are not honoured on their own:
//
//   - the user agent version token, when the stored UA has an empty version
//   - navigator.userAgentData, which only exists on secure origins and is
//     empty when the startInfo record set predates UA-CH
//
// Emulation.setUserAgentOverride with userAgentMetadata fixes both. The
// override is per target, so the browser endpoint is put in auto-attach mode
// and every page session receives the override as soon as it is created. This
// is the same shape the Undetectable main app uses: Target.setAutoAttach plus
// per-target Emulation commands.
type Overrider struct {
	cfg  *Config
	exp  *Expected
	mu   sync.Mutex
	done map[string]bool
	log  []string
}

func NewOverrider(cfg *Config) *Overrider {
	return &Overrider{cfg: cfg, exp: cfg.Expected(), done: map[string]bool{}}
}

func (o *Overrider) note(s string) {
	o.mu.Lock()
	o.log = append(o.log, s)
	o.mu.Unlock()
}

func (o *Overrider) Applied() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, s := range o.log {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func (o *Overrider) userAgentParams() map[string]any {
	e := o.exp
	uad := o.cfg.UAD
	if uad == nil {
		// No client hint data is configured; keep only the user agent and the
		// basic platform so navigator.userAgentData still exists.
		p := map[string]any{"userAgent": e.UA, "platform": e.Platform}
		if e.LangHeader != "" {
			p["acceptLanguage"] = languageList(e.LangHeader)
		}
		return p
	}
	md := map[string]any{
		"brands":          uad.Brands,
		"fullVersionList": uad.FullVersionList,
		"fullVersion":     e.FullVersion,
		"platform":        e.UADPlatform,
		"platformVersion": uad.PlatformVersion,
		"architecture":    uad.Architecture,
		"model":           uad.Model,
		"mobile":          uad.Mobile,
		"bitness":         uad.Bitness,
		"wow64":           uad.Wow64,
	}
	if len(uad.FormFactors) > 0 {
		md["formFactors"] = uad.FormFactors
	}
	p := map[string]any{
		"userAgent":         e.UA,
		"platform":          e.Platform,
		"userAgentMetadata": md,
	}
	if e.LangHeader != "" {
		// The DevTools field is a plain comma separated language list; the
		// q-values used in the Accept-Language header would otherwise leak into
		// navigator.languages as a bogus tag such as "en;q=0.9".
		p["acceptLanguage"] = languageList(e.LangHeader)
	}
	return p
}

func languageList(header string) string {
	var out []string
	for _, part := range strings.Split(header, ",") {
		if i := strings.IndexByte(part, ';'); i >= 0 {
			part = part[:i]
		}
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return header
	}
	return strings.Join(out, ",")
}

// ApplySession pushes the overrides to one attached page session.
func (o *Overrider) ApplySession(c *CDP, session string) error {
	o.mu.Lock()
	if o.done[session] {
		o.mu.Unlock()
		return nil
	}
	o.done[session] = true
	o.mu.Unlock()

	_ = c.CallSession(session, "Page.enable", nil, nil)
	if err := c.CallSession(session, "Emulation.setUserAgentOverride",
		o.userAgentParams(), nil); err != nil {
		o.mu.Lock()
		delete(o.done, session)
		o.mu.Unlock()
		return err
	}
	o.note("Emulation.setUserAgentOverride")
	if langs := o.exp.Languages; len(langs) > 0 {
		if err := c.CallSession(session, "Emulation.setLocaleOverride",
			map[string]any{"locale": langs[0]}, nil); err == nil {
			o.note("Emulation.setLocaleOverride")
		}
	}
	return nil
}

// Start attaches to the browser endpoint and keeps applying overrides to every
// page target. The returned function stops the watcher.
func (o *Overrider) Start(port int) (func(), error) {
	ver, err := HTTPVersion(port)
	if err != nil {
		return nil, err
	}
	b, err := DialCDP(ver.WebSocketDebuggerURL, 15*time.Second)
	if err != nil {
		return nil, err
	}
	b.On("Target.attachedToTarget", func(m cdpMessage) {
		var p struct {
			SessionID  string `json:"sessionId"`
			TargetInfo struct {
				Type string `json:"type"`
				URL  string `json:"url"`
			} `json:"targetInfo"`
		}
		if json.Unmarshal(m.Params, &p) != nil {
			return
		}
		if p.TargetInfo.Type == "page" || p.TargetInfo.Type == "iframe" {
			_ = o.ApplySession(b, p.SessionID)
			// OOPIFs and workers of a target require recursive auto-attachment.
			_ = b.CallSession(p.SessionID, "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true}, nil)
		}
		_ = b.CallSession(p.SessionID, "Runtime.runIfWaitingForDebugger", nil, nil)
	})
	// A service worker stopped for being idle keeps its sessions when it
	// starts again (for the next navigation or fetch in its scope): no new
	// attachedToTarget, it just waits for the debugger on the old session and
	// reports that with Inspector.targetReloadedAfterCrash. Unanswered, the
	// page load that woke it hangs (DevTools disables the start timeouts).
	b.On("Inspector.targetReloadedAfterCrash", func(m cdpMessage) {
		if m.SessionID != "" {
			_ = b.CallSession(m.SessionID, "Runtime.runIfWaitingForDebugger", nil, nil)
		}
	})
	if err := b.Call("Target.setAutoAttach", map[string]any{
		"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true,
	}, nil); err != nil {
		b.Close()
		return nil, err
	}
	// Existing targets are attached as a side effect of setAutoAttach, but a
	// sweep makes startup deterministic.
	time.Sleep(300 * time.Millisecond)
	if targets, err := ListTargets(port); err == nil {
		for _, t := range targets {
			if t.Type != "page" {
				continue
			}
			var res struct {
				SessionID string `json:"sessionId"`
			}
			if err := b.Call("Target.attachToTarget",
				map[string]any{"targetId": t.ID, "flatten": true}, &res); err == nil && res.SessionID != "" {
				if err := o.ApplySession(b, res.SessionID); err != nil {
					b.Close()
					return nil, err
				}
				_ = b.CallSession(res.SessionID, "Runtime.runIfWaitingForDebugger", nil, nil)
			}
		}
	}
	return func() { b.Close() }, nil
}

// Fingerprint is the live snapshot read back over CDP.
type Fingerprint struct {
	UA           string   `json:"userAgent"`
	AppVersion   string   `json:"appVersion"`
	Platform     string   `json:"platform"`
	Cores        int      `json:"hardwareConcurrency"`
	Memory       float64  `json:"deviceMemory"`
	Languages    []string `json:"languages"`
	Language     string   `json:"language"`
	Timezone     string   `json:"timezone"`
	Screen       []int    `json:"screen"`
	DPR          float64  `json:"devicePixelRatio"`
	MaxTouch     int      `json:"maxTouchPoints"`
	WebDriver    bool     `json:"webdriver"`
	UAData       any      `json:"userAgentData"`
	HighEntropy  any      `json:"highEntropyValues"`
	WebGL        []string `json:"webgl"`
	SpeechVoices int      `json:"speechVoices"`
	Plugins      int      `json:"plugins"`
}

const fpProbe = `(async () => {
  const o = {};
  o.userAgent = navigator.userAgent;
  o.appVersion = navigator.appVersion;
  o.platform = navigator.platform;
  o.hardwareConcurrency = navigator.hardwareConcurrency;
  o.deviceMemory = navigator.deviceMemory;
  o.languages = navigator.languages;
  o.language = navigator.language;
  o.timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  o.screen = [screen.width, screen.height, screen.availWidth, screen.availHeight,
              screen.colorDepth, screen.pixelDepth];
  o.devicePixelRatio = devicePixelRatio;
  o.maxTouchPoints = navigator.maxTouchPoints;
  o.webdriver = navigator.webdriver;
  o.plugins = navigator.plugins.length;
  if (navigator.userAgentData) {
    o.userAgentData = {
      brands: navigator.userAgentData.brands,
      platform: navigator.userAgentData.platform,
      mobile: navigator.userAgentData.mobile,
      architecture: navigator.userAgentData.architecture,
      bitness: navigator.userAgentData.bitness,
      model: navigator.userAgentData.model,
      formFactors: navigator.userAgentData.formFactors,
    };
    if (navigator.userAgentData.getHighEntropyValues) {
      try {
        o.highEntropyValues = await navigator.userAgentData.getHighEntropyValues(
          ["architecture","bitness","model","platformVersion","uaFullVersion",
           "fullVersionList","formFactors","wow64"]);
      } catch (e) { o.highEntropyError = String(e); }
    }
  }
  try {
    const c = document.createElement("canvas");
    const g = c.getContext("webgl");
    const x = g.getExtension("WEBGL_debug_renderer_info");
    o.webgl = [g.getParameter(g.VENDOR), g.getParameter(g.RENDERER),
               x ? g.getParameter(x.UNMASKED_VENDOR_WEBGL) : null,
               x ? g.getParameter(x.UNMASKED_RENDERER_WEBGL) : null];
  } catch (e) { o.webglError = String(e); }
  try { o.speechVoices = speechSynthesis.getVoices().length; } catch (e) {}
  return JSON.stringify(o);
})()`

// ReadFingerprint reads the live values from the first page target. When url is
// non-empty the page is navigated there first: navigator.userAgentData is only
// exposed on secure origins, so a probe that matters has to run on https.
func ReadFingerprint(port int, url string) (*Fingerprint, error) {
	page, err := PageTarget(port)
	if err != nil {
		return nil, err
	}
	c, err := DialCDP(page.WebSocketDebuggerURL, 15*time.Second)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if url != "" {
		_ = c.Call("Page.enable", nil, nil)
		_ = c.Call("Page.navigate", map[string]any{"url": url}, nil)
		time.Sleep(3 * time.Second)
	}
	var raw string
	if err := c.EvaluateIn("", fpProbe, &raw); err != nil {
		return nil, err
	}
	var fp Fingerprint
	if err := json.Unmarshal([]byte(raw), &fp); err != nil {
		return nil, err
	}
	return &fp, nil
}

// compare returns the list of database expectations the live browser misses.
func Compare(e *Expected, fp *Fingerprint) []string {
	var bad []string
	if e.UA != "" && fp.UA != e.UA {
		bad = append(bad, fmt.Sprintf("userAgent: want %q got %q", e.UA, fp.UA))
	}
	if e.Cores != 0 && int64(fp.Cores) != e.Cores {
		bad = append(bad, fmt.Sprintf("hardwareConcurrency: want %d got %d", e.Cores, fp.Cores))
	}
	if e.Timezone != "" && fp.Timezone != e.Timezone {
		bad = append(bad, fmt.Sprintf("timezone: want %q got %q", e.Timezone, fp.Timezone))
	}
	if len(e.Languages) > 0 && len(fp.Languages) > 0 && fp.Languages[0] != e.Languages[0] {
		bad = append(bad, fmt.Sprintf("languages: want %v got %v", e.Languages, fp.Languages))
	}
	if len(fp.Screen) >= 4 {
		if e.Width != 0 && int64(fp.Screen[0]) != e.Width {
			bad = append(bad, fmt.Sprintf("screen.width: want %d got %d", e.Width, fp.Screen[0]))
		}
		if e.Height != 0 && int64(fp.Screen[1]) != e.Height {
			bad = append(bad, fmt.Sprintf("screen.height: want %d got %d", e.Height, fp.Screen[1]))
		}
	}
	if e.DPR != 0 && fp.DPR != e.DPR {
		bad = append(bad, fmt.Sprintf("devicePixelRatio: want %v got %v", e.DPR, fp.DPR))
	}
	if e.WebGLRender != "" && len(fp.WebGL) >= 4 && fp.WebGL[3] != e.WebGLRender {
		bad = append(bad, fmt.Sprintf("webgl renderer: want %q got %q", e.WebGLRender, fp.WebGL[3]))
	}
	// navigator.deviceMemory is undefined on non-secure origins, so only flag a
	// mismatch when the browser actually reported a value.
	if e.MemoryGB != 0 && fp.Memory > 0 && int64(fp.Memory) != e.MemoryGB {
		bad = append(bad, fmt.Sprintf("deviceMemory: want %d got %v", e.MemoryGB, fp.Memory))
	}
	return bad
}

// Summarize prints a compact diff table for the `verify` subcommand.
func Summarize(e *Expected, fp *Fingerprint) string {
	var b strings.Builder
	rows := [][3]string{
		{"userAgent", e.UA, fp.UA},
		{"hardwareConcurrency", strconv.FormatInt(e.Cores, 10), strconv.Itoa(fp.Cores)},
		{"deviceMemory", strconv.FormatInt(e.MemoryGB, 10), fmt.Sprintf("%v", fp.Memory)},
		{"timezone", e.Timezone, fp.Timezone},
		{"language", primaryLang(e.Languages), primaryLang(fp.Languages)},
	}
	if len(fp.Screen) >= 4 {
		rows = append(rows,
			[3]string{"screen", fmt.Sprintf("%dx%d", e.Width, e.Height),
				fmt.Sprintf("%dx%d", fp.Screen[0], fp.Screen[1])},
			[3]string{"avail", fmt.Sprintf("%dx%d", e.Width, e.Height-48),
				fmt.Sprintf("%dx%d", fp.Screen[2], fp.Screen[3])})
	}
	rows = append(rows,
		[3]string{"devicePixelRatio", fmt.Sprintf("%v", e.DPR), fmt.Sprintf("%v", fp.DPR)},
		[3]string{"platform", e.Platform, fp.Platform},
		[3]string{"uaPlatform", e.UADPlatform, uadPlatform(fp)})
	if len(fp.WebGL) >= 4 {
		rows = append(rows, [3]string{"webglRenderer", e.WebGLRender, fp.WebGL[3]})
	}
	for _, r := range rows {
		mark := "ok  "
		if r[1] != r[2] {
			mark = "DIFF"
		}
		fmt.Fprintf(&b, "  %s %-20s db=%-58s live=%s\n", mark, r[0], truncate(r[1], 58), truncate(r[2], 58))
	}
	return b.String()
}

func primaryLang(v []string) string {
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func uadPlatform(fp *Fingerprint) string {
	if m, ok := fp.UAData.(map[string]any); ok {
		if s, ok := m["platform"].(string); ok {
			return s
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func SortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
