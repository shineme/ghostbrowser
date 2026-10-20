package main

// 内置自检: a local page (127.0.0.1, random port, one unguessable token per
// launch) that shows the values the browser actually exposes next to the
// values the profile was launched with. The launchers record the effective
// fingerprint (after IP-follow and kernel alignment) with noteFpExpect.

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type fpExpectRec struct {
	Name   string      `json:"name"`
	Kernel string      `json:"kernel"`
	FP     Fingerprint `json:"fp"`
	ExitIP string      `json:"exitIp"` // the proxy's last checked exit IP ("" = unknown / direct)
	Proxy  string      `json:"proxy"`
	At     int64       `json:"at"`
}

var fpPage = struct {
	sync.Mutex
	srv    *http.Server
	port   int
	tokens map[string]string // token -> profile ID
	expect map[string]fpExpectRec
}{tokens: map[string]string{}, expect: map[string]fpExpectRec{}}

// noteFpExpect records what a launch configured (called by every launcher).
func (s *Store) noteFpExpect(p *Profile, fp Fingerprint, kernel string) {
	rec := fpExpectRec{Name: p.Name, Kernel: kernel, FP: fp, At: time.Now().Unix()}
	var px *Proxy
	if p.InlineProxy != nil {
		px = p.InlineProxy
	} else if p.ProxyID != "" {
		px, _ = s.GetProxy(p.ProxyID)
	}
	if px != nil && px.Type != "none" && px.Type != "" {
		rec.ExitIP, rec.Proxy = px.IP, px.Name
		if rec.Proxy == "" {
			rec.Proxy = px.Host
		}
	}
	fpPage.Lock()
	fpPage.expect[p.ID] = rec
	fpPage.Unlock()
}

// fpCheckBuiltinURL starts the page server on first use and returns a fresh
// page address for one profile.
func fpCheckBuiltinURL(profileID string) (string, error) {
	fpPage.Lock()
	defer fpPage.Unlock()
	if fpPage.srv == nil {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/fpcheck/", serveFpCheck)
		fpPage.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		fpPage.port = ln.Addr().(*net.TCPAddr).Port
		go func() { _ = fpPage.srv.Serve(ln) }()
	}
	if len(fpPage.tokens) > 500 { // a long session: drop the oldest batch
		fpPage.tokens = map[string]string{}
	}
	tok := newFpToken()
	fpPage.tokens[tok] = profileID
	return "http://127.0.0.1:" + strconv.Itoa(fpPage.port) + "/fpcheck/" + tok, nil
}

func serveFpCheck(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host != "127.0.0.1" && host != "localhost" {
		http.Error(w, "forbidden host", http.StatusForbidden)
		return
	}
	tok := strings.TrimPrefix(r.URL.Path, "/fpcheck/")
	fpPage.Lock()
	id, ok := fpPage.tokens[tok]
	rec, has := fpPage.expect[id]
	fpPage.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !ok || !has {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<!doctype html><meta charset="utf-8"><title>内置自检</title><body style="font:14px system-ui;background:#0e1016;color:#cfd3dc;padding:40px">自检链接已失效（GhostBrowser 重启过）。请在 GhostBrowser 里重新启动这个环境。</body>`))
		return
	}
	b, _ := json.Marshal(rec) // escapes < > & so it is safe inside <script>
	_, _ = w.Write([]byte(strings.Replace(fpCheckHTML, "/*EXPECT*/null", string(b), 1)))
}

const fpCheckHTML = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><title>GhostBrowser 内置自检</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
body{margin:0;background:#0e1016;color:#d6d9e0;font:14px/1.5 system-ui,-apple-system,"Segoe UI","Microsoft YaHei",sans-serif}
main{max-width:980px;margin:0 auto;padding:32px 24px 60px}
h1{font-size:20px;margin:0 0 4px}.sub{color:#8a90a0;font-size:12.5px}
.sum{display:flex;gap:10px;margin:18px 0}.sum div{flex:1;background:#161922;border:1px solid #232735;border-radius:10px;padding:12px 14px}
.sum b{display:block;font-size:22px}.ok{color:#5fd39a}.bad{color:#ff7a8a}.info{color:#8fb4ff}.warn{color:#ffcf6a}
table{width:100%;border-collapse:collapse;background:#161922;border:1px solid #232735;border-radius:10px;overflow:hidden}
th,td{text-align:left;padding:9px 12px;border-bottom:1px solid #20242f;vertical-align:top}th{color:#8a90a0;font-weight:500;font-size:12px;background:#12151d}
td.v{font-family:ui-monospace,Consolas,monospace;font-size:12.5px;word-break:break-all;max-width:340px}
.tag{display:inline-block;padding:1px 8px;border-radius:99px;font-size:12px;background:#1e2230;white-space:nowrap}
</style></head><body><main>
<h1>内置自检 · <span id="nm"></span></h1>
<div class="sub" id="meta"></div>
<div class="sum"><div><span class="sub">一致</span><b class="ok" id="nOk">–</b></div><div><span class="sub">不一致</span><b class="bad" id="nBad">–</b></div><div><span class="sub">仅供参考</span><b class="info" id="nInfo">–</b></div></div>
<table><thead><tr><th style="width:150px">项目</th><th>环境配置</th><th>浏览器实际</th><th style="width:96px">结果</th></tr></thead><tbody id="rows"></tbody></table>
<p class="sub">只对比 GhostBrowser 能确定的配置值。Canvas / 音频 / WebGL 图像噪声与字体请用 BrowserScan、CreepJS 等外部检测页查看。本页只在本机打开，不上传任何数据（出口 IP 通过 api.ipify.org 查询）。</p>
</main><script>
const E = /*EXPECT*/null;
const fp = E.fp || {};
const rows = [];
const K = {"": "定制 Chromium", chromium: "开源内核", firefox: "Firefox"};
document.getElementById('nm').textContent = E.name || '';
document.getElementById('meta').textContent = '内核：' + (K[E.kernel] || E.kernel) + ' · 启动于 ' + new Date(E.at * 1000).toLocaleString() + (E.proxy ? ' · 代理：' + E.proxy : ' · 未使用代理');
const esc = (s) => String(s).replace(/[&<>"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]));
function add(name, want, got, mode) {
  let st;
  const empty = want === undefined || want === null || want === '' || want === 0 || (Array.isArray(want) && !want.length);
  if (mode === 'info') st = 'info';
  else if (empty) st = 'info';
  else if (typeof mode === 'function') st = mode(want, got);
  else st = String(want) === String(got) ? 'ok' : 'bad';
  rows.push([name, empty && mode !== 'info' ? '（未设置）' : want, got, st]);
}
function gl() {
  try {
    const g = document.createElement('canvas').getContext('webgl');
    if (!g) return ['不可用', '不可用'];
    const d = g.getExtension('WEBGL_debug_renderer_info');
    return d ? [g.getParameter(d.UNMASKED_VENDOR_WEBGL), g.getParameter(d.UNMASKED_RENDERER_WEBGL)] : [g.getParameter(g.VENDOR), g.getParameter(g.RENDERER)];
  } catch (e) { return ['错误', String(e)]; }
}
function render() {
  const lab = {ok: '<span class="tag ok">一致</span>', bad: '<span class="tag bad">不一致</span>', info: '<span class="tag info">参考</span>', approx: '<span class="tag warn">≈ 版本随内核</span>'};
  document.getElementById('rows').innerHTML = rows.map(r => '<tr><td>' + esc(r[0]) + '</td><td class="v">' + esc(Array.isArray(r[1]) ? r[1].join(', ') : r[1]) + '</td><td class="v">' + esc(Array.isArray(r[2]) ? r[2].join(', ') : r[2]) + '</td><td>' + lab[r[3]] + '</td></tr>').join('');
  document.getElementById('nOk').textContent = rows.filter(r => r[3] === 'ok' || r[3] === 'approx').length;
  document.getElementById('nBad').textContent = rows.filter(r => r[3] === 'bad').length;
  document.getElementById('nInfo').textContent = rows.filter(r => r[3] === 'info').length;
  document.title = 'GhostBrowser 内置自检 · ' + rows.filter(r => r[3] === 'bad').length + ' 项不一致';
}
const N = navigator;
const noVer = (s) => String(s).replace(/\d+(\.\d+)*/g, '#');
add('User-Agent', fp.userAgent, N.userAgent, (w, g) => w === g ? 'ok' : noVer(w) === noVer(g) ? 'approx' : 'bad');
add('navigator.platform', fp.platform, N.platform);
add('语言 languages', fp.languages, (N.languages || []).join(','), (w, g) => w.join(',') === g ? 'ok' : 'bad');
add('时区', fp.timezone, Intl.DateTimeFormat().resolvedOptions().timeZone);
add('屏幕分辨率', fp.screenWidth ? fp.screenWidth + '×' + fp.screenHeight : '', screen.width + '×' + screen.height);
add('设备像素比', fp.pixelRatio, devicePixelRatio, (w, g) => Math.abs(w - g) < 0.01 ? 'ok' : 'bad');
add('色深', fp.colorDepth, screen.colorDepth);
add('CPU 核心数', fp.hardwareConcurrency, N.hardwareConcurrency);
add('内存 deviceMemory', fp.deviceMemory, N.deviceMemory === undefined ? '不支持' : N.deviceMemory, (w, g) => String(Math.min(w, 8)) === String(g) || (g === '不支持' && E.kernel === 'firefox') ? 'ok' : 'bad');
add('触控点数', fp.maxTouchPoints === 0 ? '0' : fp.maxTouchPoints, N.maxTouchPoints);
const [gv, gr] = gl();
add('WebGL 厂商', fp.webglVendor, gv);
add('WebGL 渲染器', fp.webglRenderer, gr);
add('navigator.webdriver', 'false', String(N.webdriver));
add('Do Not Track', fp.doNotTrack ? '1' : '', N.doNotTrack, (w, g) => String(g) === '1' ? 'ok' : 'bad');
if (N.userAgentData) add('UA-CH 平台', '', N.userAgentData.platform + (N.userAgentData.mobile ? '（移动）' : ''), 'info');
add('时区偏移（分钟）', '', -new Date().getTimezoneOffset(), 'info');
render();
const ipRow = rows.length; add('出口 IP', E.exitIp || '', '查询中…', 'info'); render();
const ctl = new AbortController(); setTimeout(() => ctl.abort(), 8000);
fetch('https://api.ipify.org?format=json', {signal: ctl.signal, cache: 'no-store'}).then(r => r.json()).then(j => {
  rows[ipRow][2] = j.ip; rows[ipRow][3] = E.exitIp ? (E.exitIp === j.ip ? 'ok' : 'bad') : 'info'; render();
}).catch(() => { rows[ipRow][2] = '查询失败'; render(); });
</script></body></html>`
