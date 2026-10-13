package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// writeFPExtension generates an MV3 extension that applies the fingerprint in
// the page's MAIN world at document_start (stock-kernel mode). JS-level
// overrides are detectable by advanced anti-fraud; the custom-kernel mode is
// the production path. This mode exists so the whole app works end-to-end now.
func writeFPExtension(dir string, fp Fingerprint, px *Proxy, injectFP bool) error {
	_ = os.MkdirAll(dir, 0o755)
	manifest := map[string]any{
		"manifest_version": 3,
		"name":             "GB Core",
		"version":          "1.0.0",
		"permissions":      []string{"webRequest", "webRequestAuthProvider", "declarativeNetRequest"},
		"host_permissions": []string{"<all_urls>"},
		"background":       map[string]any{"service_worker": "bg.js"},
		"content_scripts": []map[string]any{{
			"matches":    []string{"<all_urls>"},
			"js":         []string{"fp.js"},
			"run_at":     "document_start",
			"all_frames": true,
			"world":      "MAIN",
		}},
	}
	mb, _ := json.MarshalIndent(manifest, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), mb, 0o644); err != nil {
		return err
	}
	cfg, _ := json.Marshal(fp)
	js := strings.Replace(fpScript, "__CFG__", string(cfg), 1)
	if !injectFP {
		js = "// fingerprint applied natively by the custom kernel\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "fp.js"), []byte(js), 0o644); err != nil {
		return err
	}
	user, pass := "", ""
	if px != nil {
		user, pass = px.User, px.Pass
	}
	ub, _ := json.Marshal(user)
	pb, _ := json.Marshal(pass)
	bg := strings.NewReplacer("__LANG__", strings.Join(fp.Languages, ","), "__USER__", string(ub), "__PASS__", string(pb)).Replace(bgScript)
	return os.WriteFile(filepath.Join(dir, "bg.js"), []byte(bg), 0o644)
}

// bg.js: proxy auth + Accept-Language matching navigator.languages.
const bgScript = `
const PU = __USER__, PP = __PASS__;
if (PU) chrome.webRequest.onAuthRequired.addListener(
  (d, cb) => cb(d.isProxy ? { authCredentials: { username: PU, password: PP } } : {}),
  { urls: ["<all_urls>"] }, ["asyncBlocking"]);
const langs = "__LANG__".split(",");
const al = langs.map((l,i)=> i===0 ? l : l+";q="+(1-i*0.1).toFixed(1)).join(",");
chrome.declarativeNetRequest.updateDynamicRules({
  removeRuleIds:[1],
  addRules:[{id:1,priority:1,action:{type:"modifyHeaders",requestHeaders:[{header:"Accept-Language",operation:"set",value:al}]},
    condition:{resourceTypes:["main_frame","sub_frame","xmlhttprequest","script","image","stylesheet","font","media","other","websocket","ping"]}}]
});
`

const fpScript = `(() => {
const C = __CFG__;
// deterministic PRNG from the profile seed
let s = (C.noiseSeed % 2147483647) || 1;
const rnd = () => (s = s * 16807 % 2147483647) / 2147483647;
const hide = new WeakMap();
const nativeToString = Function.prototype.toString;
const patchedToString = function () { return hide.has(this) ? hide.get(this) : nativeToString.call(this); };
hide.set(patchedToString, "function toString() { [native code] }");
Function.prototype.toString = patchedToString;
const def = (obj, prop, getter) => {
  const d = Object.getOwnPropertyDescriptor(obj, prop) || {};
  const g = { get [prop]() { return getter(); } };
  const fn = Object.getOwnPropertyDescriptor(g, prop).get;
  hide.set(fn, "function get " + prop + "() { [native code] }");
  Object.defineProperty(obj, prop, { get: fn, configurable: true, enumerable: d.enumerable !== false });
};
const wrap = (obj, name, impl) => {
  const orig = obj[name]; if (!orig) return;
  const w = { [name](...a) { return impl.call(this, orig, a); } }[name];
  hide.set(w, "function " + name + "() { [native code] }");
  obj[name] = w;
};

// navigator
const N = Navigator.prototype;
def(N, "userAgent", () => C.userAgent);
def(N, "appVersion", () => C.userAgent.replace(/^Mozilla\//, ""));
def(N, "platform", () => C.platform);
def(N, "hardwareConcurrency", () => C.hardwareConcurrency);
def(N, "deviceMemory", () => C.deviceMemory);
def(N, "maxTouchPoints", () => C.maxTouchPoints);
def(N, "language", () => C.languages[0]);
def(N, "languages", () => Object.freeze([...C.languages]));
def(N, "webdriver", () => false);
if (C.doNotTrack) def(N, "doNotTrack", () => "1");

// screen
const S = Screen.prototype;
def(S, "width", () => C.screenWidth);
def(S, "height", () => C.screenHeight);
def(S, "availWidth", () => C.screenWidth);
def(S, "availHeight", () => C.screenHeight - (C.os === "windows" ? 40 : 25));
def(S, "colorDepth", () => C.colorDepth);
def(S, "pixelDepth", () => C.colorDepth);
def(window, "devicePixelRatio", () => C.pixelRatio);

// WebGL
for (const P of [self.WebGLRenderingContext, self.WebGL2RenderingContext]) {
  if (!P) continue;
  wrap(P.prototype, "getParameter", function (orig, a) {
    if (a[0] === 37445) return C.webglVendor;
    if (a[0] === 37446) return C.webglRenderer;
    return orig.apply(this, a);
  });
  if (C.webglNoise) wrap(P.prototype, "readPixels", function (orig, a) {
    const r = orig.apply(this, a); const px = a[6];
    if (px && px.length) for (let i = 0; i < px.length; i += 97) px[i] = px[i] ^ (rnd() > .5 ? 1 : 0);
    return r;
  });
}

// Canvas noise (stable per profile)
if (C.canvasNoise) {
  const noisify = (ctx, w, h) => {
    try {
      const img = ctx.getImageData(0, 0, w, h);
      for (let i = 0; i < img.data.length; i += 4 * 53) img.data[i] = img.data[i] ^ 1;
      ctx.putImageData(img, 0, 0);
    } catch (e) {}
  };
  wrap(HTMLCanvasElement.prototype, "toDataURL", function (orig, a) {
    const ctx = this.getContext("2d"); if (ctx && this.width && this.height) noisify(ctx, this.width, this.height);
    return orig.apply(this, a);
  });
  wrap(HTMLCanvasElement.prototype, "toBlob", function (orig, a) {
    const ctx = this.getContext("2d"); if (ctx && this.width && this.height) noisify(ctx, this.width, this.height);
    return orig.apply(this, a);
  });
}

// Audio noise
if (C.audioNoise && self.AudioBuffer) {
  const done = new WeakSet();
  wrap(AudioBuffer.prototype, "getChannelData", function (orig, a) {
    const d = orig.apply(this, a);
    if (!done.has(d)) { done.add(d); for (let i = 0; i < d.length; i += 100) d[i] += (rnd() - .5) * 1e-7; }
    return d;
  });
}

// ClientRects noise
if (C.clientRectNoise) {
  const off = (rnd() - .5) * 0.0002;
  wrap(Element.prototype, "getBoundingClientRect", function (orig, a) {
    const r = orig.apply(this, a);
    return new DOMRect(r.x + off, r.y + off, r.width + off, r.height + off);
  });
}

// Timezone
if (C.timezone) {
  const DTF = Intl.DateTimeFormat;
  wrap(Intl.DateTimeFormat.prototype, "resolvedOptions", function (orig, a) {
    const o = orig.apply(this, a); o.timeZone = C.timezone; return o;
  });
  const P = function (loc, opt) { return new DTF(loc, Object.assign({ timeZone: C.timezone }, opt || {})); };
  P.prototype = DTF.prototype; P.supportedLocalesOf = DTF.supportedLocalesOf;
  hide.set(P, "function DateTimeFormat() { [native code] }");
  Intl.DateTimeFormat = P;
  const tzOffset = (d) => {
    const f = new DTF("en-US", { timeZone: C.timezone, hourCycle: "h23", year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit", second: "2-digit" });
    const p = Object.fromEntries(f.formatToParts(d).map(x => [x.type, x.value]));
    const asUTC = Date.UTC(+p.year, +p.month - 1, +p.day, +p.hour, +p.minute, +p.second);
    return Math.round((d.getTime() - asUTC) / 60000);
  };
  wrap(Date.prototype, "getTimezoneOffset", function (orig, a) { return tzOffset(this); });
}

// Geolocation
if (C.latitude || C.longitude) {
  const pos = { coords: { latitude: C.latitude, longitude: C.longitude, accuracy: 30 + rnd() * 50, altitude: null, altitudeAccuracy: null, heading: null, speed: null }, timestamp: Date.now() };
  if (navigator.geolocation) {
    wrap(Geolocation.prototype, "getCurrentPosition", function (orig, a) { setTimeout(() => a[0] && a[0](pos), 50); });
    wrap(Geolocation.prototype, "watchPosition", function (orig, a) { setTimeout(() => a[0] && a[0](pos), 50); return 1; });
  }
}

// WebRTC
if (C.webrtc === "disabled") {
  for (const k of ["RTCPeerConnection", "webkitRTCPeerConnection", "RTCDataChannel"]) { try { delete self[k]; } catch (e) {} }
}

// Media devices
if (C.hideMediaDevices && navigator.mediaDevices) {
  wrap(MediaDevices.prototype, "enumerateDevices", function () { return Promise.resolve([]); });
}

// Fonts: only report the profile's font list as available
if (C.fonts && C.fonts.length && document.fonts) {
  const set = new Set(C.fonts.map(f => f.toLowerCase()));
  wrap(FontFaceSet.prototype, "check", function (orig, a) {
    const m = /(?:\d+px\s+)?["']?([^"',]+)/.exec(a[0] || ""); const fam = m ? m[1].trim().toLowerCase() : "";
    if (["serif", "sans-serif", "monospace"].includes(fam)) return orig.apply(this, a);
    return set.has(fam);
  });
}
})();`
