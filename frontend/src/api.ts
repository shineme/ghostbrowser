// Thin wrapper over the Wails bindings. When the page runs outside Wails
// (plain `vite dev` in a browser) it falls back to an in-memory mock so the UI
// can be designed and previewed without the Go side.
import * as Go from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';

type Plain<T> = Omit<T, 'convertValues'>;
export type Fingerprint = Plain<main.Fingerprint>;
export type Proxy = Plain<main.Proxy>;
export type Profile = Omit<Plain<main.Profile>, 'fingerprint' | 'inlineProxy'> & { fingerprint: Fingerprint; inlineProxy?: Proxy };
export type Settings = Plain<main.Settings>;
export type RunState = Plain<main.RunState>;
export type NativePreset = Plain<main.NativePreset>;

const inWails = typeof (window as any).go !== 'undefined';

// ---------------- mock ----------------
const rid = () => Math.random().toString(16).slice(2, 18);
const mockFP = (os = 'windows'): any => ({
  os, osVersion: os === 'macos' ? '15.1' : '11', browserVersion: '141',
  userAgent: os === 'macos'
    ? 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36'
    : 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36',
  platform: os === 'macos' ? 'MacIntel' : 'Win32',
  screenWidth: os === 'macos' ? 1512 : 1920, screenHeight: os === 'macos' ? 982 : 1080, pixelRatio: os === 'macos' ? 2 : 1, colorDepth: 24,
  hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 0,
  webglVendor: os === 'macos' ? 'Google Inc. (Apple)' : 'Google Inc. (NVIDIA)',
  webglRenderer: os === 'macos' ? 'ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)' : 'ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002504) Direct3D11 vs_5_0 ps_5_0, D3D11)',
  languages: ['en-US', 'en'], languageFromIP: true, timezone: 'America/New_York', timezoneFromIP: true, geoFromIP: true,
  latitude: 0, longitude: 0, canvasNoise: true, audioNoise: true, webglNoise: true, clientRectNoise: true,
  noiseSeed: Math.floor(Math.random() * 1e12), webrtc: 'proxy', fonts: ['Arial', 'Calibri', 'Segoe UI', 'Tahoma', 'Verdana'], doNotTrack: false, hideMediaDevices: false,
});
const MOCK_DEV: any[] = [
  { id: 'phone-001', deviceType: 'phone', name: 'Samsung Galaxy S24 Ultra · 412x915', deviceName: 'Samsung Galaxy S24 Ultra', model: 'SM-S928B', os: 'android', osVersion: '14', webglVendor: 'Qualcomm', webglRenderer: 'Adreno (TM) 750', screenWidth: 412, screenHeight: 915, pixelRatio: 2.625, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 10 },
  { id: 'phone-009', deviceType: 'phone', name: 'Google Pixel 9 Pro · 410x914', deviceName: 'Google Pixel 9 Pro', model: 'Pixel 9 Pro', os: 'android', osVersion: '15', webglVendor: 'ARM', webglRenderer: 'Mali-G715', screenWidth: 410, screenHeight: 914, pixelRatio: 3.125, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 10 },
  { id: 'tablet-026', deviceType: 'tablet', name: 'Samsung Galaxy Tab S9 · 800x1280', deviceName: 'Samsung Galaxy Tab S9', model: 'SM-X710', os: 'android', osVersion: '14', webglVendor: 'Qualcomm', webglRenderer: 'Adreno (TM) 740', screenWidth: 800, screenHeight: 1280, pixelRatio: 2, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 10 },
  { id: 'foldable-034', deviceType: 'foldable', name: 'Samsung Galaxy Z Fold6 · 展开 · 707x823', deviceName: 'Samsung Galaxy Z Fold6 · 展开', model: 'SM-F956B', os: 'android', osVersion: '14', webglVendor: 'Qualcomm', webglRenderer: 'Adreno (TM) 750', screenWidth: 707, screenHeight: 823, pixelRatio: 2.625, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 10 },
  { id: 'chromebook-039', deviceType: 'chromebook', name: 'Acer Chromebook Plus 515 · 1536x864', deviceName: 'Acer Chromebook Plus 515', os: 'chromeos', osVersion: '16093.68.0', webglVendor: 'Google Inc. (Intel)', webglRenderer: 'ANGLE (Intel, Mesa Intel(R) UHD Graphics (ADL GT2), OpenGL ES 3.2)', screenWidth: 1536, screenHeight: 864, pixelRatio: 1.25, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 0 },
  { id: 'tv-044', deviceType: 'tv', name: 'Chromecast with Google TV · 960x540', deviceName: 'Chromecast with Google TV', model: 'Chromecast', os: 'android', osVersion: '12', webglVendor: 'ARM', webglRenderer: 'Mali-G31', screenWidth: 960, screenHeight: 540, pixelRatio: 2, hardwareConcurrency: 4, deviceMemory: 2, maxTouchPoints: 0 },
  { id: 'win-001', deviceType: 'desktop', name: 'Win11 · RTX 4060 · 1920x1080', os: 'windows', osVersion: '11', webglVendor: 'Google Inc. (NVIDIA)', webglRenderer: 'ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 (0x00002882) Direct3D11 vs_5_0 ps_5_0, D3D11)', screenWidth: 1920, screenHeight: 1080, pixelRatio: 1, hardwareConcurrency: 12, deviceMemory: 8, maxTouchPoints: 0 },
  { id: 'win-050', deviceType: 'laptop', name: 'Win11 · Iris Xe · 1920x1080', os: 'windows', osVersion: '11', webglVendor: 'Google Inc. (Intel)', webglRenderer: 'ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)', screenWidth: 1920, screenHeight: 1080, pixelRatio: 1.25, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 10 },
  { id: 'mac-001', deviceType: 'laptop', name: 'macOS 15.3 · Apple M3 Pro · 1512x982', os: 'macos', osVersion: '15.3', webglVendor: 'Google Inc. (Apple)', webglRenderer: 'ANGLE (Apple, ANGLE Metal Renderer: Apple M3 Pro, Unspecified Version)', screenWidth: 1512, screenHeight: 982, pixelRatio: 2, hardwareConcurrency: 12, deviceMemory: 8, maxTouchPoints: 0 },
];
const mockDevFP = (t: string, os: string): any => {
  t = t || 'desktop';
  const d = MOCK_DEV.find(x => x.deviceType === t && (!os || x.os === os)) || MOCK_DEV.find(x => x.deviceType === t);
  const base = mockFP(['windows', 'macos', 'linux'].includes(os) ? os : 'windows');
  if (!d) return { ...base, deviceType: t };
  const mobileTok = t === 'phone' || t === 'foldable' ? 'Mobile ' : '';
  const ua = d.os === 'android' ? `Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 ${mobileTok}Safari/537.36` : d.os === 'chromeos' ? 'Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36' : base.userAgent;
  return { ...base, ...d, id: undefined, name: undefined, model: undefined, os: d.os, deviceType: t, deviceModel: d.model || '', deviceName: d.deviceName || '', userAgent: ua, platform: d.os === 'android' ? 'Linux armv81' : d.os === 'chromeos' ? 'Linux x86_64' : base.platform, fonts: d.os === 'android' ? ['Roboto', 'Noto Sans', 'Noto Serif'] : base.fonts };
};
const mock = {
  profiles: [] as any[],
  proxies: [
    { id: 'px1', name: 'US Residential 01', type: 'socks5', host: '45.12.30.8', port: 1080, user: 'u', pass: 'p', changeIpUrl: '', ip: '45.12.30.8', country: 'US', city: 'Ashburn', timezone: 'America/New_York', lat: 39.04, lon: -77.49, latency: 182, status: 'ok', checkedAt: Date.now() / 1000 },
    { id: 'px2', name: 'DE Datacenter', type: 'http', host: '88.198.4.21', port: 3128, user: '', pass: '', changeIpUrl: '', ip: '88.198.4.21', country: 'DE', city: 'Falkenstein', timezone: 'Europe/Berlin', lat: 50.47, lon: 12.37, latency: 96, status: 'ok', checkedAt: Date.now() / 1000 },
    { id: 'px3', name: 'JP Mobile', type: 'http', host: '133.18.2.40', port: 8000, user: 'a', pass: 'b', changeIpUrl: 'https://example/rotate', ip: '', country: '', city: '', timezone: '', lat: 0, lon: 0, latency: 0, status: 'fail', checkedAt: 0 },
  ] as any[],
  running: [] as any[],
  settings: { kernelPath: '', kernelMode: 'undetectable', undetectableKit: 'undetectable-kit', profilesDir: 'C:\\Users\\me\\AppData\\Roaming\\GhostBrowser\\profiles', apiPort: 25325, apiEnabled: true, defaultOS: 'windows', syncServer: '', syncToken: '', syncPassword: '', deviceName: 'DESKTOP-01', deviceId: 'abc', theme: 'dark', syncPromptSecs: 3, syncPromptAction: 'pull' } as any,
};
['Shopify-Store-A', 'FB-Ads-01', 'FB-Ads-02', 'Amazon-Seller-EU', 'TikTok-Shop', 'Gmail-Warmup'].forEach((n, i) => {
  mock.profiles.push({ id: rid(), name: n, group: i < 3 ? '广告投放' : '电商', tags: i % 2 ? ['养号'] : ['主力'], notes: '', storage: i % 3 === 0 ? 'cloud' : 'local', proxyId: i % 2 ? 'px2' : 'px1', startUrls: [], cookies: '', fingerprint: mockFP(i === 3 ? 'macos' : 'windows'), createdAt: Date.now() / 1000 - i * 3600, lastRun: i < 4 ? Date.now() / 1000 - i * 5000 : 0, cloudVersion: i % 3 === 0 ? 3 : 0, syncedAt: 0 });
});
// ---- arrange + syncer ---- (mock helpers; preview starts with 4 running browsers so the run strip shows)
mock.profiles.slice(0, 4).forEach((p, i) => mock.running.push({ profileId: p.id, pid: 4100 + i, debugPort: 9300 + i, wsUrl: '', startedAt: Date.now() / 1000 - 600 - i * 1300 }));
const mockSync: { st: any; timer: any } = { st: { running: false, master: '', masterName: '', followers: [], events: 0, lastEvent: '', startedAt: 0, reason: '' }, timer: 0 };
// ---- syncer2 ---- (mock helpers)
const sync2Mock: { prefs: any } = { prefs: { click: true, input: true, key: true, scroll: true, tabs: true, delayMin: 0, delayMax: 120, master: '', followers: [] as string[], newTab: false } };
function sync2Norm(o: any) {
  const c = (v: any) => Math.max(0, Math.min(10000, Math.round(Number(v) || 0)));
  let a = c(o.delayMin), b = c(o.delayMax); if (a > b) [a, b] = [b, a];
  return { click: !!o.click, input: !!o.input, key: !!o.key, scroll: !!o.scroll, tabs: !!o.tabs, delayMin: a, delayMax: b };
}
function sync2URL(raw: string): string {
  let s = String(raw || '').trim();
  if (!s) throw new Error('请输入网址');
  if (/^(about:blank|chrome:\/\/|edge:\/\/)/i.test(s)) return s;
  if (/\s/.test(s)) throw new Error('网址里不能有空格');
  if (s.startsWith('//')) s = 'https:' + s; else if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(s)) s = 'https://' + s;
  let u: URL; try { u = new URL(s); } catch { throw new Error('网址格式不正确'); }
  if (!/^https?:$/.test(u.protocol)) throw new Error('只支持 http / https 网址');
  if (!u.hostname || (u.hostname !== 'localhost' && !u.hostname.includes('.'))) throw new Error('网址格式不正确');
  return s;
}
function mockArrange(n: number, mode: string) {
  const screen = { x: 0, y: 0, w: 1920, h: 1040 }; const cells: any[] = [];
  mode = mode === 'cascade' || mode === 'stack' ? mode : 'grid';
  if (n <= 0) return { mode, screen, cols: 0, rows: 0, cells };
  if (mode === 'cascade') {
    const w = Math.round(screen.w * .62), h = Math.round(screen.h * .78);
    for (let i = 0; i < n; i++) cells.push({ x: (i * 32) % (screen.w - w + 1), y: (i * 32) % (screen.h - h + 1), w, h });
    return { mode, screen, cols: 1, rows: n, cells };
  }
  if (mode === 'stack') {
    const w = Math.round(screen.w * .72), h = Math.round(screen.h * .86);
    for (let i = 0; i < n; i++) cells.push({ x: (screen.w - w) >> 1, y: (screen.h - h) >> 1, w, h });
    return { mode, screen, cols: 1, rows: 1, cells };
  }
  const cols = Math.ceil(Math.sqrt(n * screen.w / screen.h / 1.6)) || 1, rows = Math.ceil(n / cols);
  for (let i = 0; i < n; i++) {
    const row = Math.floor(i / cols), inRow = row === rows - 1 ? n - row * cols : cols, col = i % cols;
    const y = Math.round(screen.h * row / rows), h = Math.round(screen.h * (row + 1) / rows) - y;
    const x = Math.round(screen.w * col / inRow), w = Math.round(screen.w * (col + 1) / inRow) - x;
    cells.push({ x, y, w, h });
  }
  return { mode, screen, cols, rows, cells };
}
// ---- fingerprint protection + launch controls ----
Object.assign(mock.profiles[1], { kernel: 'chromium' });
Object.assign(mock.profiles[1].fingerprint, { canvasMode: 'noise', webglImageMode: 'noise', audioMode: 'real', audioNoise: false, clientRectsMode: 'off', clientRectNoise: false,
  mediaDevicesMode: 'custom', mediaMics: 1, mediaSpeakers: 2, mediaCameras: 1, speechVoices: 'custom', webgpu: 'off', doNotTrack: true,
  portScanProtection: true, scanWhitelist: '3000, 8080, localhost:5173', windowMode: 'custom', windowWidth: 1440, windowHeight: 900, windowX: 120, windowY: 60,
  hardwareAcceleration: true, blockVideo: true, startupParameters: '--lang=en-US\n--disable-features=Translate' });
Object.assign(mock.profiles[4], { kernel: 'chromium' });
Object.assign(mock.profiles[4].fingerprint, { webrtc: 'disabled', randomFingerprintOnLaunch: true, windowMode: 'maximized', blockImages: true, blockSound: true, hardwareAcceleration: false });
const wait = (ms = 250) => new Promise(r => setTimeout(r, ms));
// ---- mcp / audit (mock state) ----
const mcpMock = { running: !location.hash.includes('mcperr'), enabled: true, port: location.hash.includes('mcperr') ? 25326 : 25326, error: location.hash.includes('mcperr') ? 'listen tcp 127.0.0.1:25326: bind: Only one usage of each socket address (protocol/network address/port) is normally permitted.' : '', calls: 128, lastCall: Date.now() - 42e3, lastClient: 'cursor-vscode 1.7.2', startedAt: Date.now() - 3 * 3600e3 };
const MCP_TOOLS = ['list_environments', 'create_environment', 'start_environment', 'stop_environment', 'list_proxies', 'assign_proxy', 'list_groups', 'create_group', 'set_environment_group', 'list_tags', 'list_kernels'];
function mcpMockStatus() {
  if (!mock.settings.apiKey) mock.settings.apiKey = 'gb_7f3a9c21e84b4d06a5c1f2e9d3b87a40';
  const key = mock.settings.apiKey as string, keyRequired = !mock.settings.apiKeyOff, url = `http://127.0.0.1:${mcpMock.port}/mcp`;
  const http: any = { url }; if (keyRequired) http.headers = { 'X-API-Key': key };
  const exe = 'C:\\Program Files\\GhostBrowser\\GhostBrowser.exe';
  return { ...mcpMock, url, keyRequired, tools: MCP_TOOLS, calls: mcpMock.running ? mcpMock.calls : 0, lastCall: mcpMock.running ? mcpMock.lastCall : 0, lastClient: mcpMock.running ? mcpMock.lastClient : '', startedAt: mcpMock.running ? mcpMock.startedAt : 0, exePath: exe, protocol: '2025-06-18',
    configHttp: JSON.stringify({ mcpServers: { ghostbrowser: http } }, null, 2),
    configStdio: JSON.stringify({ mcpServers: { ghostbrowser: { command: exe, args: ['--mcp-stdio'] } } }, null, 2),
    claudeCodeCl: `claude mcp add --transport http ghostbrowser ${url}` + (keyRequired ? ` --header "X-API-Key: ${key}"` : '') };
}
if (!mock.settings.apiKey) mock.settings.apiKey = 'gb_7f3a9c21e84b4d06a5c1f2e9d3b87a40';
const auditMock = { list: [] as any[] };
const auditPush = (source: string, action: string, target: string, name: string, error: string, detail: string, time = Date.now()) =>
  auditMock.list.unshift({ id: rid(), time, source, action, target, name, ok: !error, error: error || undefined, detail: detail || undefined });
if (!location.hash.includes('auditempty')) {
  const P = mock.profiles; const pick = (i: number) => P[i % P.length];
  const rows: [number, string, string, number, string, string][] = [
    // [minutes ago, source, action, profile index (-1 none), error, detail]
    [2, 'mcp', 'mcp.call', -1, '', 'list_environments {"group":"广告投放"} · cursor-vscode 1.7.2'],
    [3, 'mcp', 'mcp.call', -1, '', 'start_environment {"id":"FB-Ads-01"} · cursor-vscode 1.7.2'],
    [3, 'mcp', 'profile.start', 1, '', 'chromium · CDP 127.0.0.1:52144'],
    [5, 'mcp', 'mcp.call', -1, '代理 US Residential 01 连接超时 (10s)', 'assign_proxy {"proxy":"px1","ids":["FB-Ads-02"]} · cursor-vscode 1.7.2'],
    [7, 'ui', 'profile.stop', 0, '', ''],
    [9, 'sync', 'sync.push', 0, '', '24117248 B, 3180 ms'],
    [12, 'api', 'profile.start', 4, '', '/profile/start · puppeteer'],
    [12, 'api', 'profile.stop', 4, '', '/profile/stop'],
    [18, 'ui', 'proxy.assign', 2, '', 'px2'],
    [18, 'ui', 'proxy.assign', 3, '', 'px2'],
    [25, 'ui', 'group.set', 5, '', '电商 养号'],
    [31, 'mcp', 'mcp.call', -1, '', 'create_group {"name":"TikTok 矩阵"} · claude-ai 0.9.4'],
    [31, 'mcp', 'group.create', -1, '', ''],
    [33, 'mcp', 'mcp.call', -1, '', 'create_environment {"count":3,"os":"android","group":"TikTok 矩阵"} · claude-ai 0.9.4'],
    [33, 'mcp', 'profile.create', 4, '', '创建 3 个环境'],
    [40, 'sync', 'sync.pull', 3, 'sync HTTP 413: Request Entity Too Large', '0 B, 912 ms'],
    [46, 'ui', 'profile.edit', 3, '', ''],
    [52, 'ui', 'profile.clone', 1, '', '源: FB-Ads-01'],
    [58, 'api', 'profile.start', 2, '内核启动失败：chrome.exe 退出码 0xC0000005', '/profile/start'],
    [64, 'ui', 'mcp.start', -1, '', 'port 25326'],
    [70, 'ui', 'mcp.stop', -1, '', ''],
    [95, 'sync', 'sync.push', 1, '', '81264640 B, 9120 ms'],
    [120, 'ui', 'profile.start', 0, '', ''],
    [135, 'mcp', 'mcp.call', -1, '', 'list_proxies {} · claude-ai 0.9.4'],
    [140, 'mcp', 'profile.edit', 2, '', '分配代理「DE Datacenter」给 2 个环境'],
    [160, 'api', 'group.create', -1, '', '/group'],
    [190, 'ui', 'profile.delete', 5, '', ''],
    [220, 'ui', 'profile.create', 5, '', 'phone/android'],
    [260, 'sync', 'sync.push', 2, 'dial tcp 203.0.113.8:443: i/o timeout', '0 B, 30004 ms'],
    [300, 'mcp', 'mcp.call', -1, 'unknown environment "Shopify-Store-Z"', 'stop_environment {"id":"Shopify-Store-Z"} · claude-ai 0.9.4'],
    [340, 'ui', 'group.create', -1, '', ''],
    [420, 'api', 'profile.start', 3, '', '/profile/start · playwright'],
    [500, 'ui', 'profile.edit', 0, '', ''],
    [640, 'sync', 'sync.pull', 0, '', '18874368 B, 2410 ms'],
    [800, 'ui', 'proxy.assign', 1, '', 'px1'],
    [1020, 'mcp', 'mcp.call', -1, '', 'list_kernels {} · cursor-vscode 1.7.2'],
    [1300, 'ui', 'profile.start', 3, '', ''],
    [1800, 'ui', 'profile.create', 2, '', 'desktop/windows'],
    [2600, 'sync', 'sync.push', 3, '', '52428800 B, 6020 ms'],
    [3900, 'ui', 'profile.create', 0, '', 'desktop/windows'],
  ];
  const names: Record<string, string> = { 'group.create': 'TikTok 矩阵' };
  for (const [m, src, act, pi, err, det] of rows.reverse()) {
    const p = pi >= 0 ? pick(pi) : null;
    const nm = act === 'mcp.call' ? det.split(' ')[0] : p ? p.name : (names[act] || (act === 'group.create' && src === 'api' ? '亚马逊欧洲站' : ''));
    auditPush(src, act, p ? p.id : '', act === 'group.create' && src === 'ui' ? '店铺运营' : nm, err, det, Date.now() - m * 60e3);
  }
}


// ---- cloud delete (mock data) ----
const cdelNow = Date.now() / 1000;
const cdelMock: any[] = [
  { id: 'c0a1f3e9b2d4c5a6', name: 'TikTok-Shop-03', version: 5, updatedAt: cdelNow - 7200, deviceName: 'OFFICE-PC', size: 46e6, lockedBy: '', localDeleted: true, deletedVersion: 5, deletedAt: cdelNow - 86400, cloudNewer: false, pendingDelete: false },
  { id: 'c7d2e8f1a3b9c4d0', name: 'Amazon-Seller-UK-主店铺-长名称测试环境-用于检查省略号效果', version: 9, updatedAt: cdelNow - 1500, deviceName: 'LAPTOP-HOME-WITH-A-VERY-LONG-HOSTNAME', size: 88e6, lockedBy: '', localDeleted: true, deletedVersion: 7, deletedAt: cdelNow - 3 * 86400, cloudNewer: true, pendingDelete: false },
  { id: 'c3b4a5d6e7f8a9b0', name: 'Shopee-MY-01', version: 0, updatedAt: 0, deviceName: '', size: 0, lockedBy: '', localDeleted: false, deletedVersion: 0, deletedAt: 0, cloudNewer: false, pendingDelete: true, pendingError: 'Post "https://vps.example.com/ghostsync.php?r=/api/profiles/c3b4a5d6e7f8a9b0/delete": dial tcp: i/o timeout' },
  { id: 'c9e8d7c6b5a4f3e2', name: 'Etsy-02', version: 2, updatedAt: cdelNow - 5 * 86400, deviceName: 'OFFICE-PC', size: 12e6, lockedBy: '', localDeleted: false, deletedVersion: 0, deletedAt: 0, cloudNewer: false, pendingDelete: false },
];

// ---- recycle bin (mock data) ----
const trashDaysMock = 30;
const trashMock: any[] = [
  { id: 't1a2b3c4d5e6f708', name: 'FB-广告-07', group: '广告投放', tags: ['养号'], storage: 'local', cloudVersion: 0, createdAt: cdelNow - 40 * 86400, lastRun: cdelNow - 3 * 86400, fingerprint: mockFP(), trashed: true, trashedAt: cdelNow - 2 * 3600, size: 182e6, hasData: true, cloudNote: '仅本机', restoreAsNew: false },
  { id: 't2b3c4d5e6f70819', name: 'Amazon-Seller-DE-备用店铺-长名称测试环境-用于检查省略号效果是否正常', group: '电商', tags: ['主力'], storage: 'cloud', cloudVersion: 6, createdAt: cdelNow - 90 * 86400, lastRun: cdelNow - 6 * 86400, fingerprint: mockFP('macos'), trashed: true, trashedAt: cdelNow - 4 * 86400, size: 64e6, hasData: true, cloudNote: '云端副本保留', restoreAsNew: false },
  { id: 't3c4d5e6f708192a', name: 'TikTok-Shop-11', group: '', tags: [], storage: 'cloud', cloudVersion: 3, createdAt: cdelNow - 20 * 86400, lastRun: 0, fingerprint: mockFP(), trashed: true, trashedAt: cdelNow - 26 * 86400, size: 0, hasData: false, cloudNote: '云端已删除', restoreAsNew: true },
];
const trashPush = (p: any, cloudDeleted: boolean) => {
  if (!p) return;
  trashMock.unshift({ ...p, trashed: true, trashedAt: Date.now() / 1000, size: 40e6 + Math.round(Math.random() * 120e6), hasData: true, cloudNote: cloudDeleted ? '云端已删除' : p.cloudVersion > 0 || p.storage === 'cloud' ? '云端副本保留' : '仅本机', restoreAsNew: cloudDeleted });
};

// ---- 环境自检 mock state ----
const envMock: { st: Record<string, { fixed: Set<string>; log: any[] }>; clockSkew: number | null; changes: Record<string, any[]>; state: (id: string) => { fixed: Set<string>; log: any[] } } = {
  st: {}, clockSkew: location.hash.includes('clock-ok') ? 4 : location.hash.includes('clock-none') ? null : 187,
  changes: {
    language: [{ item: 'language', label: '语言跟随 IP', field: 'languageFromIP', old: '固定值', new: '跟随 IP' }, { item: 'language', label: '语言', field: 'languages', old: 'zh-CN, zh, en', new: 'en-US, en' }],
    timezone: [{ item: 'timezone', label: '时区跟随 IP', field: 'timezoneFromIP', old: '固定值', new: '跟随 IP' }, { item: 'timezone', label: '时区', field: 'timezone', old: 'America/Vancouver', new: 'America/Los_Angeles' }],
    webrtc: [{ item: 'webrtc', label: 'WebRTC', field: 'webrtc', old: '真实', new: '替换为代理 IP' }],
    seed: [{ item: 'seed', label: '噪声种子', field: 'noiseSeed', old: '0', new: '1840275531' }],
  },
  state(id) {
    if (!this.st[id]) {
      const t = Date.now() / 1000;
      this.st[id] = { fixed: new Set(), log: [
        { id: rid(), profileId: id, at: t - 86400 * 2 - 3600, kind: 'restore', items: ['geo', 'fonts'], changes: [{ item: 'geo', label: '定位跟随 IP', field: 'geoFromIP', old: '跟随 IP', new: '固定值' }, { item: 'geo', label: '纬度', field: 'latitude', old: '34.0522', new: '31.2304' }, { item: 'geo', label: '经度', field: 'longitude', old: '-118.2437', new: '121.4737' }, { item: 'fonts', label: '字体列表', field: 'fonts', old: '49 个字体', new: '58 个字体' }], skipped: [], restored: false, restoreOf: 'old1' },
        { id: 'old1', profileId: id, at: t - 86400 * 3, kind: 'repair', items: ['geo', 'fonts'], changes: [{ item: 'geo', label: '定位跟随 IP', field: 'geoFromIP', old: '固定值', new: '跟随 IP' }, { item: 'geo', label: '纬度', field: 'latitude', old: '31.2304', new: '34.0522' }, { item: 'geo', label: '经度', field: 'longitude', old: '121.4737', new: '-118.2437' }, { item: 'fonts', label: '字体列表', field: 'fonts', old: '58 个字体', new: '49 个字体' }], skipped: ['系统时钟：系统级设置，无法自动修复，请开启系统自动对时'], restored: true, restoredAt: t - 86400 * 2 - 3600 },
      ] };
    }
    return this.st[id];
  },
};

// ---- terminal ----
export type TerminalTool = { id: string; name: string; bin: string; installed: boolean; version: string; path: string; installUrl: string; installCmd: string; desc: string };
export type TerminalSession = { id: string; tool: string; name: string; dir: string; profileId: string; profileName: string; tz: string; lang: string; proxy: string; relay: string; host: string; pid: number; tracked: boolean; started: number };
const termMock = {
  prefs: { tool: 'claude', dir: '', envMode: 'profile', profileId: '', tz: 'Asia/Tokyo', lang: 'ja-JP', host: 'console' } as any,
  sessions: [] as TerminalSession[],
  tools: (): TerminalTool[] => {
    const none = location.hash.includes('term-none');
    return [
      { id: 'claude', name: 'Claude Code', bin: 'claude', installed: !none, version: none ? '' : '2.0.14', path: none ? '' : 'C:\\Users\\Ghost\\AppData\\Roaming\\npm\\claude.cmd', installUrl: 'https://docs.anthropic.com/en/docs/claude-code/setup', installCmd: 'npm install -g @anthropic-ai/claude-code', desc: 'Anthropic 的编程智能体' },
      { id: 'codex', name: 'Codex CLI', bin: 'codex', installed: !none, version: none ? '' : '0.46.0', path: none ? '' : 'C:\\Users\\Ghost\\AppData\\Roaming\\npm\\codex.cmd', installUrl: 'https://github.com/openai/codex', installCmd: 'npm install -g @openai/codex', desc: 'OpenAI 的终端编程助手' },
      { id: 'gemini', name: 'Gemini CLI', bin: 'gemini', installed: false, version: '', path: '', installUrl: 'https://github.com/google-gemini/gemini-cli', installCmd: 'npm install -g @google/gemini-cli', desc: 'Google 的开源命令行智能体' },
      { id: 'shell', name: 'PowerShell', bin: 'powershell.exe', installed: true, version: '', path: 'C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe', installUrl: 'https://learn.microsoft.com/powershell/', installCmd: '', desc: 'Windows PowerShell 纯终端' },
    ];
  },
};

const M = {
 NativePresets: async (): Promise<NativePreset[]> => [{id:'demo', name:'Windows / Intel', configId:'3285798'}],
 UseUndetectableKit: async () => { mock.settings.kernelMode='undetectable'; mock.settings.undetectableKit='undetectable-kit'; return {...mock.settings}; },
 CreateNativeProfile: async (_id: string, name: string) => { const p={...mock.profiles[0], id:rid(), name:name||'Native Profile',native:{templateKey:'windows-intel-a',sourceConfigId:'3285798',data:{},configMeta:{}}}; mock.profiles.push(p); return p; },
 ExportNativeJSON: async (id: string) => JSON.stringify(mock.profiles.find(p=>p.id===id),null,2),
 ImportNativeJSON: async (raw: string) => { const p={...mock.profiles[0],...JSON.parse(raw),id:rid()}; mock.profiles.push(p); return p; },
 NativeSnapshot: async (_id: string) => JSON.stringify({preview:true, message:'浏览器预览模式不执行原生指纹验证'},null,2),
  ListProfiles: async () => [...mock.profiles],
  SaveProfile: async (p: any) => { if (!p.id) { p.id = rid(); p.createdAt = Date.now() / 1000; } mock.profiles = [p, ...mock.profiles.filter(x => x.id !== p.id)]; return p; },
  DeleteProfiles: async (ids: string[]) => { mock.profiles = mock.profiles.filter(p => !ids.includes(p.id)); },
  CreateProfiles: async (prefix: string, os: string, n: number, proxyId: string, group: string) => {
    const out = []; for (let i = 1; i <= n; i++) { const p = { id: rid(), name: n > 1 ? `${prefix}-${i}` : prefix, group, tags: [], notes: '', storage: 'local', proxyId, startUrls: [], cookies: '', fingerprint: mockFP(os), createdAt: Date.now() / 1000, lastRun: 0, cloudVersion: 0, syncedAt: 0 }; mock.profiles.unshift(p); out.push(p); } return out;
  },
  CloneProfile: async (id: string) => { const p = mock.profiles.find(x => x.id === id); const c = { ...p, id: rid(), name: p.name + ' (copy)', fingerprint: mockFP(p.fingerprint.os) }; mock.profiles.unshift(c); return c; },
  NewFingerprint: async (os: string) => mockFP(os),
  NewDeviceFingerprint: async (t: string, os: string) => mockDevFP(t, os),
  CreateTypedProfiles: async (prefix: string, t: string, os: string, n: number, proxyId: string, group: string) => {
    const out = []; for (let i = 1; i <= n; i++) { const fp = mockDevFP(t, os); const p = { id: rid(), name: n > 1 ? `${prefix}-${i}` : prefix, group, tags: [], notes: '', storage: 'local', proxyId, startUrls: [], cookies: '', fingerprint: fp, kernel: (fp.os !== 'windows' || !['desktop', 'laptop'].includes(t)) ? 'chromium' : '', createdAt: Date.now() / 1000, lastRun: 0, cloudVersion: 0, syncedAt: 0 }; mock.profiles.unshift(p); out.push(p); } return out;
  },
  DeviceTypes: async (): Promise<any[]> => [],
  DevicePresetsByType: async (t: string, os: string): Promise<any[]> => MOCK_DEV.filter(d => d.deviceType === (t || 'desktop') && (!os || d.os === os)),
  FingerprintCatalogFor: async (os: string, t: string): Promise<any> => {
    if (['windows', 'macos', 'linux'].includes(os)) return (M as any).FingerprintCatalog(os);
    const ua = os === 'android' ? `Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 ${t === 'tablet' || t === 'tv' ? '' : 'Mobile '}Safari/537.36` : 'Mozilla/5.0 (X11; CrOS x86_64 14541.0.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36';
    const devs = MOCK_DEV.filter(d => d.os === os);
    return { osVersions: os === 'android' ? ['15', '14', '13', '12'] : ['16093.68.0'], browserMajors: ['141'], userAgents: [ua], platforms: [os === 'android' ? 'Linux armv81' : 'Linux x86_64'], vendors: [...new Set(devs.map(d => d.webglVendor))], gpus: devs.map(d => ({ vendor: d.webglVendor, renderer: d.webglRenderer })), screens: devs.map(d => `${d.screenWidth}x${d.screenHeight}@${d.pixelRatio}`), cores: ['4', '8', '10'], memory: ['2', '4', '8'], custom: {}, fonts: ['Roboto', 'Noto Sans', 'Noto Serif', 'Noto Color Emoji', 'Droid Sans Mono'] };
  },
  ChromiumPath: async () => mock.settings.chromiumPath || (!inWails && location.hash.includes('nokernel') ? '' : 'C:\\Program Files\\Chromium\\Application\\chrome.exe'),
  ChromiumMajor: async () => '141',
  PickChromium: async () => 'C:\\Program Files\\Chromium\\Application\\chrome.exe',
  LibraryAdd: async (_os: string, _f: string, _v: string) => {},
  LibraryRemove: async (_os: string, _f: string, _v: string) => {},
  SaveDevicePreset: async (p: any) => ({ ...p, id: 'my-1', name: p.name || '我的预设', custom: true }),
  DeleteDevicePreset: async (_id: string) => {},
  PendingSyncPrompt: async (): Promise<any> => location.hash.includes('syncprompt') ? { seconds: 3, action: 'pull', held: false, deadline: Date.now() + 3000, items: [
    { id: 'p1', name: 'Shopify-Store-A', action: 'pull', reason: 'cloud_newer', conflict: false, cloudVersion: 7, localVersion: 5, cloudUpdatedAt: Date.now() / 1000 - 3600, cloudDevice: 'OFFICE-PC', localLastRun: Date.now() / 1000 - 86400, size: 18e6 },
    { id: 'p2', name: 'FB Ads 02', action: 'pull', reason: 'cloud_newer', conflict: true, cloudVersion: 3, localVersion: 2, cloudUpdatedAt: Date.now() / 1000 - 600, cloudDevice: 'LAPTOP', localLastRun: Date.now() / 1000 - 1200, size: 42e6 }] } : null,
  SyncHold: async () => {},
  SyncDecide: async (_ids: string[]) => true,
  SyncNow: async (): Promise<any> => ({ phase: 'done', pulled: 0, pushed: 0, failed: 0, message: '同步完成：下载 0，上传 0，失败 0' }),
  FingerprintCatalog: async (os: string): Promise<any> => {
    const majors = ['142', '141', '140', '139', '138'];
    const tok = os === 'macos' ? 'Macintosh; Intel Mac OS X 10_15_7' : os === 'linux' ? 'X11; Linux x86_64' : 'Windows NT 10.0; Win64; x64';
    return {
      osVersions: os === 'macos' ? ['15.5', '15.3', '14.7', '13.7'] : os === 'linux' ? ['x86_64'] : ['11', '10'], browserMajors: majors,
      userAgents: majors.map(m => `Mozilla/5.0 (${tok}) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/${m}.0.0.0 Safari/537.36`),
      platforms: [os === 'macos' ? 'MacIntel' : os === 'linux' ? 'Linux x86_64' : 'Win32'],
      vendors: os === 'macos' ? ['Google Inc. (Apple)'] : ['Google Inc. (NVIDIA)', 'Google Inc. (Intel)'],
      gpus: os === 'macos' ? [{ vendor: 'Google Inc. (Apple)', renderer: 'ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)' }] : [{ vendor: 'Google Inc. (NVIDIA)', renderer: 'ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002504) Direct3D11 vs_5_0 ps_5_0, D3D11)' }, { vendor: 'Google Inc. (Intel)', renderer: 'ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)' }],
      screens: ['1920x1080@1', '2560x1440@1'], cores: ['4', '8', '16'], memory: ['2', '4', '8'], custom: {},
      fonts: os === 'macos' ? ['Helvetica', 'Helvetica Neue', 'Menlo', 'PingFang SC'] : ['Arial', 'Calibri', 'Consolas', 'Microsoft YaHei', 'Segoe UI'],
    };
  },
  DevicePresets: async (os: string): Promise<any[]> => [
    { id: 'win-001', name: 'Win11 · RTX 4060 · 1920x1080', os: 'windows', osVersion: '11', webglVendor: 'Google Inc. (NVIDIA)', webglRenderer: 'ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 (0x00002882) Direct3D11 vs_5_0 ps_5_0, D3D11)', screenWidth: 1920, screenHeight: 1080, pixelRatio: 1, hardwareConcurrency: 12, deviceMemory: 8, maxTouchPoints: 0 },
    { id: 'mac-001', name: 'macOS 15.3 · Apple M3 Pro · 1512x982', os: 'macos', osVersion: '15.3', webglVendor: 'Google Inc. (Apple)', webglRenderer: 'ANGLE (Apple, ANGLE Metal Renderer: Apple M3 Pro, Unspecified Version)', screenWidth: 1512, screenHeight: 982, pixelRatio: 2, hardwareConcurrency: 12, deviceMemory: 8, maxTouchPoints: 0 },
  ].filter(d => !os || d.os === os),
  GPUOptions: async (os: string) => os === 'macos'
    ? [{ vendor: 'Google Inc. (Apple)', renderer: 'ANGLE (Apple, ANGLE Metal Renderer: Apple M1, Unspecified Version)' }, { vendor: 'Google Inc. (Apple)', renderer: 'ANGLE (Apple, ANGLE Metal Renderer: Apple M2, Unspecified Version)' }]
    : [{ vendor: 'Google Inc. (NVIDIA)', renderer: 'ANGLE (NVIDIA, NVIDIA GeForce RTX 3060 (0x00002504) Direct3D11 vs_5_0 ps_5_0, D3D11)' }, { vendor: 'Google Inc. (Intel)', renderer: 'ANGLE (Intel, Intel(R) Iris(R) Xe Graphics (0x00009A49) Direct3D11 vs_5_0 ps_5_0, D3D11)' }],
  ScreenOptions: async (os: string) => os === 'macos' ? ['1440x900@2', '1512x982@2', '1728x1117@2'] : ['1920x1080@1', '1366x768@1', '1536x864@1.25', '2560x1440@1'],
  StartProfile: async (id: string) => { await wait(600); if (!inWails && location.hash.includes('nokernel') && !mock.settings.chromiumPath) throw '没有找到开源内核：请在「设置 → 开源内核」选择 chrome.exe（Chromium、Chrome 或你自己编译的内核）'; const r = { profileId: id, pid: 1234, debugPort: 9222, wsUrl: 'ws://127.0.0.1:9222/devtools/browser/x', startedAt: Date.now() / 1000 }; mock.running.push(r); return r; },
  StopProfile: async (id: string) => { mock.running = mock.running.filter(r => r.profileId !== id); },
  Running: async () => [...mock.running],
  ExportCookies: async (_id: string, format: string) => {
    await wait(300); const exp = Math.floor(Date.now() / 1000) + 86400 * 180;
    const cs = [['.facebook.com', 'c_user', '100089234567812', true, true], ['.facebook.com', 'xs', '23%3AaB9kQ2xYz-pL%3A2%3A1728900000%3A-1%3A11092', true, true], ['.facebook.com', 'datr', 'Xh0nZ3w9mK2pQaTbV8r1sL4e', true, true], ['.facebook.com', 'fr', '0Ab1Cd2Ef3Gh4.AWX9kLmNoPq.BnDkY7', true, true], ['.google.com', 'NID', '518=Kx3mP9qL2vR8tY1wZ4aB6cD0eF', true, true], ['.google.com', 'SID', 'g.a000pQhZ8kL2mN4oP6qR8sT0uV2wX4yZ', true, false], ['www.amazon.com', 'session-id', '135-7924681-3058247', true, false], ['www.amazon.com', 'ubid-main', '132-4485921-7765103', true, false]] as [string, string, string, boolean, boolean][];
    if (format === 'netscape') return '# Netscape HTTP Cookie File\n# Exported by GhostBrowser\n\n' + cs.map(([d, n, v, s, h]) => `${h ? '#HttpOnly_' : ''}${d}\t${d.startsWith('.') ? 'TRUE' : 'FALSE'}\t/\t${s ? 'TRUE' : 'FALSE'}\t${exp}\t${n}\t${v}`).join('\n') + '\n';
    return JSON.stringify(cs.map(([d, n, v, s, h]) => ({ domain: d, name: n, value: v, path: '/', secure: s, httpOnly: h, sameSite: 'no_restriction', expirationDate: exp })), null, 2);
  },
  ListProxies: async () => [...mock.proxies],
  SaveProxy: async (p: any) => { if (!p.id) p.id = rid(); if (!p.status) p.status = 'unknown'; mock.proxies = [...mock.proxies.filter(x => x.id !== p.id), p]; return p; },
  DeleteProxy: async (id: string) => { mock.proxies = mock.proxies.filter(p => p.id !== id); },
  ParseProxy: async (l: string) => ({ type: 'http', host: l.split(':')[0], port: +l.split(':')[1] || 0, name: l } as any),
  ImportProxies: async (t: string) => t.split('\n').filter(Boolean).map(l => { const p: any = { id: rid(), name: l, type: 'http', host: l.split(':')[0], port: +l.split(':')[1] || 0, status: 'unknown' }; mock.proxies.push(p); return p; }),
  CheckProxy: async (id: string) => { await wait(900); const p = mock.proxies.find(x => x.id === id); p.status = 'ok'; p.latency = 140; p.ip = p.host; p.country = p.country || 'US'; return p; },
  CheckProxyDraft: async (p: any) => { await wait(900); return { ...p, status: 'ok', ip: p.host, country: 'US', city: 'Ashburn', timezone: 'America/New_York', latency: 150 }; },
  CheckAllProxies: async () => { await wait(1400); return mock.proxies; },
  MatchGeo: async (proxyId: string, _inline: any) => { await wait(700); const px = mock.proxies.find(x => x.id === proxyId); return { ok: true, error: '', stale: false, direct: !px, ip: px?.ip || '73.12.4.9', country: px?.country || 'US', region: px?.region || 'California', city: px?.city || 'Los Angeles', isp: px?.isp || 'Comcast Cable', ipType: px?.ipType || 'residential', timezone: px?.timezone || 'America/Los_Angeles', languages: ['en-US', 'en'], lat: 34.0522, lon: -118.2437, latency: px?.latency || 160 } as any; },
  CheckProxies: async (ids: string[]) => { await wait(1200); for (const p of mock.proxies) if (!ids.length || ids.includes(p.id)) Object.assign(p, { status: 'ok', latency: 120 + Math.round(Math.random() * 300), connectMs: 600, ip: p.host, country: p.country || 'US', city: 'Los Angeles', region: 'California', isp: 'Comcast Cable', ipType: 'residential', checkedAt: Date.now() / 1000 }); return mock.proxies; },
  SpeedTestProxies: async (ids: string[]) => { await wait(1500); for (const p of mock.proxies) if (!ids.length || ids.includes(p.id)) Object.assign(p, { speedKbps: 8000 + Math.round(Math.random() * 40000), speedAt: Date.now() / 1000 }); return mock.proxies; },
  DeleteProxies: async (ids: string[]) => { mock.proxies = mock.proxies.filter(p => !ids.includes(p.id)); },
  PreviewProxies: async (text: string, def: string) => text.split('\n').map((raw, i) => ({ raw: raw.trim(), line: i + 1 })).filter(x => x.raw && !x.raw.startsWith('#')).map(({ raw, line }) => {
    const m = /^(?:(\w+):\/\/)?(?:([^:@]+):([^@]+)@)?([\w.-]+):(\d+)(?::([^:]+):(.+))?$/.exec(raw);
    if (!m) return { line, raw, ok: false, error: '无法识别格式', dup: false, proxy: {} as any };
    const port = +m[5]; const type = m[1] || (def === 'auto' ? (port === 1080 ? 'socks5' : 'http') : def);
    return { line, raw, ok: true, error: '', dup: mock.proxies.some(p => p.host === m[4] && p.port === port), proxy: { type, host: m[4], port, user: m[2] || m[6] || '', pass: m[3] || m[7] || '', name: `${m[4]}:${port}`, typeGuessed: !m[1] && def === 'auto' } as any };
  }),
  ImportProxyList: async (list: any[], tag: string) => list.map((p, i) => { const x = { ...p, id: rid(), status: 'unknown', tag, name: tag ? `${tag} #${i + 1}` : p.name }; mock.proxies.push(x); return x; }),
  GetSettings: async () => ({ ...mock.settings }),
  SaveSettings: async (s: any) => (mock.settings = s),
  DefaultKernelPath: async () => mock.settings.undetectableKit ? `${mock.settings.undetectableKit}\\runtime\\chrome\\chrome.exe` : mock.settings.kernelPath,
  PickKernel: async () => mock.settings.kernelPath,
  PickFolder: async () => mock.settings.profilesDir,
  KernelMajor: async () => '141',
  TestSync: async () => { await wait(700); if (!mock.settings.syncServer) throw '请先填写同步服务器地址'; },
  Transfers: async (): Promise<any[]> => [],
  PushProfile: async (id: string) => { const q = mock.profiles.find(x => x.id === id); await simTransfer(id, q?.name || id, 'up', 38 << 20, 212 << 20, 6000); const p = mock.profiles.find(x => x.id === id); p.storage = 'cloud'; p.cloudVersion++; p.syncedAt = Date.now() / 1000; },
  PullProfile: async (id: string) => { const q = mock.profiles.find(x => x.id === id); await simTransfer(id, q?.name || id, 'down', 64 << 20, 96 << 20, 7000); },
  ListCloud: async () => mock.profiles.filter(p => p.storage === 'cloud').map(p => ({ id: p.id, name: p.name, version: p.cloudVersion, sha256: 'e3b0c442…', meta: null, lockedBy: '' })),
  DataDir: async () => 'C:\\Users\\me\\AppData\\Roaming\\GhostBrowser',
  EnvStatus: async (): Promise<any> => ({ ready: true, kitDir: 'undetectable-kit', chromeExe: 'undetectable-kit\\runtime\\chrome\\chrome.exe', phase: 'ready', pct: 100, done: 0, total: 0, speedBps: 0, message: '环境就绪', updatedAt: Math.floor(Date.now() / 1000), manifestFromServer: false }),
  EnvRecheck: async (_force: boolean): Promise<any> => ({ ready: true, kitDir: 'undetectable-kit', chromeExe: 'undetectable-kit\\runtime\\chrome\\chrome.exe', phase: 'ready', pct: 100, done: 0, total: 0, speedBps: 0, message: '环境就绪', updatedAt: Math.floor(Date.now() / 1000), manifestFromServer: false }),
  EnvCancel: async () => {},
  ResetAPIKey: async () => (mock.settings = { ...mock.settings, apiKey: 'gb_' + rid() + rid() }),
  SyncServerInfo: async (): Promise<any> => ({ features: ['chunks', 'history', 'multiuser'], keepVersions: 3, maxBody: 8388608, usedBytes: 734003200, freeBytes: 21474836480, user: 'owner', protocol: 2 }),
  CloudHistory: async (_id: string): Promise<any[]> => [{ version: 3, size: 18000, updatedAt: Date.now() / 1000 - 600, deviceName: 'Office-PC', current: true }, { version: 2, size: 17000, updatedAt: Date.now() / 1000 - 86400, deviceName: 'Laptop', current: false }],
  RestoreVersion: async (_id: string, _v: number) => { await wait(900); },
  SyncLog: async (): Promise<any[]> => [{ time: Date.now() / 1000 - 120, id: 'x', name: 'Shop US 01', action: 'push', ok: true, error: '', bytes: 2400000, ms: 3200, version: 4 }, { time: Date.now() / 1000 - 3600, id: 'y', name: 'Ads 02', action: 'pull', ok: false, error: 'sync HTTP 413: Request Entity Too Large', bytes: 0, ms: 900, version: 2 }],
  ClearSyncLog: async () => {},
  RetrySync: async (_id: string, _a: string) => { await wait(600); },
  SyncBusy: async (): Promise<string[]> => [],
  RotateProxy: async (id: string) => { await wait(1500); const p = mock.proxies.find(x => x.id === id); p.ip = `45.${Math.round(Math.random() * 255)}.${Math.round(Math.random() * 255)}.${Math.round(Math.random() * 255)}`; return p; },
  AssignProxies: async (ids: string[], px: string[], _mode: string) => { ids.forEach((id, i) => { const p = mock.profiles.find(x => x.id === id); if (p) p.proxyId = px.length ? px[i % px.length] : ''; }); return ids.length; },
  SetProfilesGroup: async (ids: string[], group: string, tag: string) => { ids.forEach(id => { const p = mock.profiles.find(x => x.id === id); if (!p) return; if (group !== '\u0000') p.group = group; if (tag && !(p.tags || []).includes(tag)) p.tags = [...(p.tags || []), tag]; }); return ids.length; },
  StopProfiles: async (_ids: string[]) => {},
  PushProfiles: async (ids: string[]): Promise<any> => { await wait(900); return { ok: ids.length, failed: 0, errors: [] }; },
  FocusProfile: async (_id: string) => {},
  // ---- mcp / audit ----
  McpStatus: async (): Promise<any> => mcpMockStatus(),
  StartMcp: async (port: number): Promise<any> => { await wait(500); if (port > 0) mcpMock.port = port; if (mcpMock.port < 1024) { mcpMock.running = false; mcpMock.error = `listen tcp 127.0.0.1:${mcpMock.port}: bind: An attempt was made to access a socket in a way forbidden by its access permissions.`; auditPush('ui', 'mcp.start', '', '', mcpMock.error, `port ${mcpMock.port}`); throw mcpMock.error; } Object.assign(mcpMock, { running: true, enabled: true, error: '', startedAt: Date.now() }); auditPush('ui', 'mcp.start', '', '', '', `port ${mcpMock.port}`); return mcpMockStatus(); },
  StopMcp: async (): Promise<any> => { await wait(250); Object.assign(mcpMock, { running: false, enabled: false, error: '' }); auditPush('ui', 'mcp.stop', '', '', '', ''); return mcpMockStatus(); },
  AuditLog: async (f: any): Promise<any> => {
    const all = auditMock.list;
    const r: any = { items: [], total: 0, all: all.length, failed: all.filter(e => !e.ok).length, categories: {}, sources: {}, targets: [] };
    const tg: Record<string, any> = {};
    const q = (f.query || '').trim().toLowerCase();
    for (const e of all) {
      const cat = e.action.split('.')[0];
      r.categories[cat] = (r.categories[cat] || 0) + 1; r.sources[e.source] = (r.sources[e.source] || 0) + 1;
      if (e.target) (tg[e.target] ||= { id: e.target, name: e.name, count: 0 }).count++;
      const hit = (!f.category || cat === f.category) && (!f.action || e.action === f.action) && (!f.source || e.source === f.source) && (!f.target || e.target === f.target) && (!f.onlyFail || !e.ok) && (!f.since || e.time >= f.since) && (!q || [e.name, e.detail, e.error, e.target, e.action].join('\u0000').toLowerCase().includes(q));
      if (hit) { r.total++; if (!f.limit || r.items.length < f.limit) r.items.push(e); }
    }
    r.targets = Object.values(tg).sort((a: any, b: any) => b.count - a.count || a.name.localeCompare(b.name));
    return r;
  },
  ClearAuditLog: async () => { auditMock.list = []; },
  ExportAuditLog: async (_f: any, format: string) => { await wait(300); return `C:\\Users\\me\\Documents\\ghostbrowser-audit-20261014-0930.${format === 'json' ? 'json' : 'csv'}`; },
  // ---- arrange + syncer ----
  ArrangePreview: async (n: number, mode: string): Promise<any> => mockArrange(n, mode),
  ArrangeWindows: async (ids: string[], mode: string): Promise<any> => {
    await wait(700);
    const runs = (ids && ids.length ? mock.running.filter(r => ids.includes(r.profileId)) : mock.running);
    if (!runs.length) throw new Error('没有正在运行的环境');
    const lay = mockArrange(runs.length, mode);
    const failed = runs.map(r => mock.profiles.find(p => p.id === r.profileId)).filter(p => p && /Gmail/.test(p.name)).map(p => `${p.name}：窗口已最小化，无法调整位置`);
    return { ...lay, arranged: runs.length - failed.length, total: runs.length, failed };
  },
  StartSync: async (master: string, followers: string[]): Promise<any> => {
    await wait(500);
    if (!mock.running.some(r => r.profileId === master)) throw new Error('主控环境没有在运行');
    const fs = followers.filter(id => id !== master && mock.running.some(r => r.profileId === id));
    if (!fs.length) throw new Error('请至少选择一个正在运行的跟随环境');
    clearInterval(mockSync.timer);
    const nm = (id: string) => mock.profiles.find(p => p.id === id)?.name || id;
    mockSync.st = { running: true, master, masterName: nm(master), followers: fs.map(id => ({ id, name: nm(id), ok: true, events: 0, error: '', latencyMs: 0, tabs: 2 })), events: 0, lastEvent: '', startedAt: Date.now(), reason: '', options: sync2Norm(sync2Mock.prefs), tabs: 2 };
    const kinds = ['click', 'click', 'scroll', 'scroll', 'scroll', 'key', 'text', 'select', 'nav', 'tabswitch', 'tabnew'];
    mockSync.timer = setInterval(() => {
      const st = mockSync.st;
      if (!mock.running.some(r => r.profileId === st.master)) { clearInterval(mockSync.timer); mockSync.st = { ...st, running: false, reason: '主控浏览器已关闭' }; mockEmit('syncer:event', mockSync.st); return; }
      const ev = kinds[Math.floor(Math.random() * kinds.length)];
      const followers = st.followers.map((f: any, i: number) => {
        if (!f.ok) return f;
        if (i === 2 && st.events >= 14) return { ...f, ok: false, error: '标签页已关闭，已移出同步' };
        const o = st.options || {}; return { ...f, events: f.events + 1, latencyMs: Math.round((o.delayMin || 0) + Math.random() * ((o.delayMax || 0) - (o.delayMin || 0)) + 6 + Math.random() * 14) };
      });
      mockSync.st = { ...st, followers, events: st.events + 1, lastEvent: ev };
      mockEmit('syncer:event', mockSync.st);
    }, 650);
    return mockSync.st;
  },
  StopSync: async (): Promise<any> => { clearInterval(mockSync.timer); mockSync.st = { ...mockSync.st, running: false, reason: '' }; mockEmit('syncer:event', mockSync.st); return mockSync.st; },
  SyncerState: async (): Promise<any> => ({ ...mockSync.st }),
  // ---- rpa ----
  ListWorkflows: async (): Promise<any[]> => rpaMock.wfs.map(w => JSON.parse(JSON.stringify(w))),
  SaveWorkflow: async (w: any): Promise<any> => {
    if (!String(w.name || '').trim()) throw '请填写流程名称';
    const now = Math.floor(Date.now() / 1000);
    const i = rpaMock.wfs.findIndex(x => x.id === w.id && w.id);
    const x = i >= 0 ? { ...rpaMock.wfs[i], ...w, name: w.name.trim(), updatedAt: now } : { lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0, vars: {}, ...w, id: w.id || rid(), name: w.name.trim(), createdAt: now, updatedAt: now };
    if (i >= 0) rpaMock.wfs[i] = x; else rpaMock.wfs.unshift(x);
    return JSON.parse(JSON.stringify(x));
  },
  DeleteWorkflow: async (id: string) => { rpaMock.wfs = rpaMock.wfs.filter(w => w.id !== id); },
  ValidateWorkflow: async (w: any): Promise<any[]> => rpaValidate(w),
  DuplicateWorkflow: async (id: string): Promise<any> => {
    const w = rpaMock.wfs.find(x => x.id === id); if (!w) throw '流程不存在';
    const fresh = (l: any[]): any[] => (l || []).map(s => ({ ...s, id: rid(), children: s.children && fresh(s.children), else: s.else && fresh(s.else) }));
    const now = Math.floor(Date.now() / 1000);
    const x = { ...JSON.parse(JSON.stringify(w)), id: rid(), name: w.name + ' 副本', steps: fresh(w.steps), createdAt: now, updatedAt: now, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0 };
    rpaMock.wfs.splice(rpaMock.wfs.indexOf(w) + 1, 0, x); return x;
  },
  RunWorkflow: async (id: string, ids: string[]): Promise<string> => rpaSimRun(id, ids, { concurrency: 1, autoStart: true, closeAfter: false }),
  RunWorkflowWith: async (id: string, ids: string[], opt: any): Promise<string> => rpaSimRun(id, ids, opt),
  CancelWorkflowRun: async (runId: string): Promise<boolean> => { const r = rpaMock.runs.find(x => x.id === runId); if (!r || r.status !== 'running') return false; r.cancel = true; return true; },
  WorkflowRuns: async (wfId: string): Promise<any[]> => rpaMock.runs.filter(r => r.workflowId === wfId).map(rpaPub),
  WorkflowRun: async (runId: string): Promise<any> => { const r = rpaMock.runs.find(x => x.id === runId); if (!r) throw '运行记录不存在'; return rpaPub(r); },
  RpaScreenshot: async (path: string): Promise<string> => { await wait(120); return rpaShot(path); },
  ExportWorkflow: async (id: string): Promise<string> => { const w = rpaMock.wfs.find(x => x.id === id); if (!w) throw '流程不存在'; return JSON.stringify({ format: 'ghostbrowser-rpa', version: 1, workflow: { ...w, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0 } }, null, 2); },
  ImportWorkflow: async (raw: string): Promise<any> => {
    let o: any; try { o = JSON.parse(raw); } catch (e: any) { throw `不是有效的 JSON：${e.message}`; }
    const w = o.workflow || o; if (!w.steps?.length) throw 'JSON 里没有步骤（steps）';
    return M.SaveWorkflow({ ...w, id: '', name: w.name || '导入的流程' });
  },
  SaveWorkflowFile: async (id: string): Promise<string> => { const w = rpaMock.wfs.find(x => x.id === id); return `C:\\Users\\me\\Desktop\\${w?.name || 'workflow'}.rpa.json`; },
  OpenWorkflowFile: async (): Promise<string> => { await wait(200); return JSON.stringify({ format: 'ghostbrowser-rpa', version: 1, workflow: { name: 'Shopee 店铺签到', description: '每天打开卖家中心领取签到积分', vars: {}, steps: [
    { id: 'i1', type: 'goto', params: { url: 'https://seller.shopee.sg/', timeout: '30000' } },
    { id: 'i2', type: 'wait_for', params: { selector: 'text:签到', state: 'visible', timeout: '15000' } },
    { id: 'i3', type: 'click', params: { selector: 'text:签到' } },
    { id: 'i4', type: 'screenshot', params: { fullPage: 'false', saveAs: '' } }] } }, null, 2); },
  // ---- extensions / cookies / mihomo ----
  ListExtensions: async (): Promise<any[]> => { await wait(200); if (location.hash.includes('noext')) return []; return extMock.list.map(e => ({ ...e, profiles: mock.profiles.filter(p => (p.extensionIds || []).includes(e.id)).length })); },
  AddExtensionByID: async (raw: string): Promise<any> => {
    await wait(1100); const m = /([a-p]{32})/.exec(raw); if (!m) throw '无法识别扩展 ID：应为 32 位 a-p 字母，或 Chrome 应用商店链接';
    if (extMock.list.some(e => e.id === m[1])) throw '这个扩展已经安装过了';
    const e = { id: m[1], name: 'Cookie-Editor', version: '1.13.0', description: '简单高效的 Cookie 管理器，可以快速新建、编辑、删除、导入和导出 Cookie。', source: 'store', origin: `https://chromewebstore.google.com/detail/${m[1]}`, dir: `C:\\Users\\me\\AppData\\Roaming\\GhostBrowser\\extensions\\${m[1]}`, icon: extIcon('#f6a623', 'C'), manifestVersion: 3, size: 412000, addedAt: Date.now() / 1000, profiles: 0 };
    extMock.list.unshift(e); return e;
  },
  AddExtensionFromURL: async (src: string): Promise<any> => {
    await wait(900); if (!/^(https?:\/\/|[a-z]:\\|\/)/i.test(src)) throw '请输入 http(s) 下载链接或本地 .crx / .zip 路径';
    const id = rid() + rid(); const e = { id: id.replace(/[^a-p]/g, 'a').slice(0, 32).padEnd(32, 'b'), name: 'Proxy SwitchyOmega 3', version: '3.0.2', description: '轻松快捷地管理和切换多个代理设置（ZeroOmega 分支，支持 MV3）。', source: /^https?:/.test(src) ? 'url' : 'file', origin: src, dir: '', icon: extIcon('#3b82f6', 'Ω'), manifestVersion: 3, size: 820000, addedAt: Date.now() / 1000, profiles: 0 };
    extMock.list.unshift(e); return e;
  },
  RemoveExtension: async (id: string) => { await wait(300); extMock.list = extMock.list.filter(e => e.id !== id); mock.profiles.forEach(p => { if (p.extensionIds) p.extensionIds = p.extensionIds.filter((x: string) => x !== id); }); },
  SetProfileExtensions: async (profileID: string, ids: string[]) => { const p = mock.profiles.find(x => x.id === profileID); if (!p) throw '环境不存在'; p.extensionIds = ids.filter(i => extMock.list.some(e => e.id === i)); },
  ImportCookies: async (id: string, format: string, data: string): Promise<any> => {
    await wait(700); const p = mock.profiles.find(x => x.id === id); if (!p) throw '环境不存在';
    let list: any[] = [];
    const t = data.trim(); if (!t) throw '没有可导入的内容';
    if ((format === 'json' || format === 'auto') && /^[\[{]/.test(t)) { try { const j = JSON.parse(t); list = Array.isArray(j) ? j : (j.cookies || [j]); } catch { throw 'JSON 解析失败：请检查格式'; } }
    else list = t.split('\n').filter(l => l.trim() && (!l.startsWith('#') || l.startsWith('#HttpOnly_'))).map(l => l.split('\t')).filter(c => c.length >= 7).map(c => ({ domain: c[0].replace('#HttpOnly_', ''), path: c[2], secure: c[3] === 'TRUE', expirationDate: +c[4], name: c[5], value: c[6] }));
    const good = list.filter(c => c && c.name && c.domain); if (!good.length) throw '没有识别到任何 Cookie';
    const running = mock.running.some(r => r.profileId === id);
    if (running) return { count: good.length, skipped: list.length - good.length, mode: 'live', staged: p.cookies || '' };
    const prev = (() => { try { return JSON.parse(p.cookies || '[]'); } catch { return []; } })();
    p.cookies = JSON.stringify([...prev, ...good], null, 0); cookieMock.staged[id] = (cookieMock.staged[id] || 0) + good.length;
    return { count: good.length, skipped: list.length - good.length, mode: 'staged', staged: p.cookies };
  },
  CookieInfo: async (id: string): Promise<any> => { const p = mock.profiles.find(x => x.id === id); let staged = 0; try { staged = JSON.parse(p?.cookies || '[]').length; } catch { /* */ } return { running: mock.running.some(r => r.profileId === id), snapshotAt: p?.lastRun ? p.lastRun + 1800 : 0, snapshot: p?.lastRun ? 47 : 0, staged }; },
  MihomoConnect: async (controller: string, secret: string): Promise<any> => {
    await wait(800); if (!/^(https?:\/\/)?[\w.-]+:\d+\/?$/.test(controller.trim())) throw '控制器地址格式不对，例如 http://127.0.0.1:9090';
    if (secret === 'wrong') throw 'Mihomo 拒绝访问（HTTP 401）：密钥不正确';
    mihomoMock.status = { ...mihomoMock.status, configured: true, connected: true, controller: controller.trim(), hasSecret: !!secret, error: '' }; return { ...mihomoMock.status };
  },
  MihomoStatus: async (): Promise<any> => ({ ...mihomoMock.status }),
  MihomoProxies: async (): Promise<any> => { await wait(300); if (!mihomoMock.status.connected) throw '尚未连接 Mihomo'; return { groups: mihomoMock.groups.map(g => ({ ...g })), nodes: mihomoMock.nodes.map(n => ({ ...n })) }; },
  MihomoSelect: async (group: string, node: string) => { await wait(250); const g = mihomoMock.groups.find(x => x.name === group); if (!g) throw '策略组不存在'; if (g.type !== 'Selector') throw `${g.type} 策略组会自动选择节点，不能手动切换`; g.now = node; },
  MihomoTestDelay: async (node: string): Promise<number> => { await wait(400 + Math.random() * 900); const n = mihomoMock.nodes.find(x => x.name === node); if (n && /过期|故障/.test(n.name)) { n.alive = false; n.delay = 0; throw '测速超时（5 秒）'; } const d = 60 + Math.round(Math.random() * 420); if (n) { n.delay = d; n.alive = true; } return d; },
  MihomoAddProxy: async (): Promise<any> => { await wait(400); const s = mihomoMock.status; const ex = mock.proxies.find(p => p.host === s.listenHost && p.port === s.listenPort); if (ex) return ex; const p = { id: rid(), name: `Mihomo · ${s.listenHost}:${s.listenPort}`, type: s.listenKind === 'socks' ? 'socks5' : 'http', host: s.listenHost, port: s.listenPort, user: '', pass: '', tag: 'mihomo', changeIpUrl: '', status: 'unknown' }; mock.proxies.push(p); return p; },
  // ---- chromium compat ----
  ChromiumCompat: async (path: string) => {
    const p = path || mock.settings.chromiumPath || 'C:\\Program Files\\Chromium\\Application\\chrome.exe';
    if (/google[\\/]chrome[\\/]application/i.test(p)) return { path: p, version: '141.0.7390.122', major: 141, level: 'warn', message: '品牌版 Google Chrome 141 不再支持命令行加载扩展：扩展中心分配的扩展不会生效（指纹不受影响）。需要扩展请改用 Chromium 或 Chrome for Testing' };
    if (/1[01]\d\./.test(p)) return { path: p, version: '115.0.5790.170', major: 115, level: 'warn', message: 'Chrome 115 可以使用，但新标签页会多加载一次以注入指纹，窗口同步可能无效，建议升级到 120 以上' };
    if (/old|10\d\./i.test(p)) return { path: p, version: '104.0.5112.79', major: 104, level: 'unsupported', message: 'Chrome 104 版本过旧，不受支持：需要 110 及以上，推荐 120 以上' };
    return { path: p, version: '141.0.7390.122', major: 141, level: 'ok', message: '' };
  },
  // ---- kernel install ----
  KernelInstallStatus: async (): Promise<any> => ({ ...kernelMock.p }),
  KernelInstallStart: async (source: string, url: string): Promise<any> => { if (kernelMock.p.running) throw '已有内核正在安装'; Object.assign(mock.settings, { kernelSource: source, kernelUrl: url }); simKernel(source); return { ...kernelMock.p }; },
  KernelInstallCancel: async () => { kernelMock.cancel = true; },
  KernelList: async (): Promise<any[]> => kernelMock.list.map(k => ({ ...k, active: k.path === mock.settings.chromiumPath })),
  KernelDelete: async (dir: string) => { await wait(300); kernelMock.list = kernelMock.list.filter(k => k.dir !== dir); },
  // ---- syncer2 ----
  GetSyncerPrefs: async (): Promise<any> => JSON.parse(JSON.stringify(sync2Mock.prefs)),
  SaveSyncerPrefs: async (p: any): Promise<any> => {
    const o = sync2Norm(p); sync2Mock.prefs = { ...p, ...o, followers: (p.followers || []).filter((f: string) => f && f !== p.master) };
    if (mockSync.st.running) { mockSync.st = { ...mockSync.st, options: o }; mockEmit('syncer:event', mockSync.st); }
    return JSON.parse(JSON.stringify(sync2Mock.prefs));
  },
  SetSyncOptions: async (o: any): Promise<any> => { mockSync.st = { ...mockSync.st, options: sync2Norm(o) }; mockEmit('syncer:event', mockSync.st); return mockSync.st; },
  NormalizeSyncURL: async (u: string): Promise<string> => sync2URL(u),
  SyncTabAction: async (master: string, followers: string[], action: string, url: string, _newTab: boolean): Promise<any> => {
    await wait(action === 'unify' ? 1300 : 650);
    const ids = action === 'unify' || action === 'size' ? followers.filter(f => f !== master) : [master, ...followers].filter(Boolean);
    if ((action === 'unify' || action === 'size') && !master) throw new Error('请先选择主控窗口');
    if (!ids.length) throw new Error('请先选择窗口');
    const u = action === 'open' ? sync2URL(url) : '';
    const nm = (id: string) => mock.profiles.find(p => p.id === id)?.name || id;
    const failed = ids.filter(id => !mock.running.some(r => r.profileId === id)).map(id => `${nm(id)}：没有在运行`);
    return { action, done: ids.length - failed.length, total: ids.length, failed, url: u };
  },
  // ---- rpa recorder ----
  StartRecording: async (profileId: string, opts: any): Promise<any> => recSimStart(profileId, opts),
  StopRecording: async (): Promise<any> => recSimStop('user'),
  PauseRecording: async (): Promise<any> => recSimPause(true),
  ResumeRecording: async (): Promise<any> => recSimPause(false),
  RecordingState: async (): Promise<any> => recSim.pub(),
  RecordRemoveStep: async (id: string): Promise<any> => { recSim.st.steps = recSim.st.steps.filter((s: any) => s.id !== id); return recSim.emit(); },
  RecordUpdateStep: async (step: any): Promise<any> => { recSim.st.steps = recSim.st.steps.map((s: any) => (s.id === step.id ? { ...step, params: { ...step.params, _rec: s.params._rec || step.params._rec || '' } } : s)); return recSim.emit(); },
  FinalizeRecording: async (steps: any[], o: any): Promise<any> => recFinalize(steps, o),
  RecordPickMode: async (mode: string): Promise<any> => {
    const S = recSim.st; if (!S?.active || mode === 'none') return recSim.pub();
    setTimeout(() => { if (!recSim.st?.active) return; S.steps.push(mode === 'extract' ? rs('extract_text', { selector: '.order-total', var: `text${S.steps.filter((x: any) => x.type === 'extract_text').length + 1}` }) : rs('wait_for', { selector: '.toast-success', state: 'visible', timeout: '10000' })); recSim.emit(); }, 1400);
    return recSim.pub();
  },
  // ---- cloud delete ----
  _cdelInit: (() => { if (location.hash.includes('cloud')) Object.assign(mock.settings, { syncServer: 'https://vps.example.com/ghostsync.php', syncToken: 'x'.repeat(40), syncPassword: 'pw', deleteMode: '' }); return 0; })(),
  SetDeleteMode: async (m: string) => { mock.settings.deleteMode = m; },
  DeleteProfilesMode: async (ids: string[], mode: string, force: boolean): Promise<any[]> => {
    await wait(700); const now = Date.now() / 1000;
    return ids.map(id => {
      const p = mock.profiles.find(x => x.id === id); const co = cdelMock.find(x => x.id === id); const name = p?.name || co?.name || id;
      if (mode === 'cloud' && !force && /FB/i.test(name)) return { id, name, local: false, cloud: 'locked', error: '该环境正在被另一台设备（LAPTOP-HOME）同步，暂时不能删除；稍后重试或选择强制删除。' };
      if (p) trashPush(p, mode === 'cloud' && p.storage === 'cloud'); // recycle bin
      mock.profiles = mock.profiles.filter(x => x.id !== id); mock.running = mock.running.filter(r => r.profileId !== id);
      if (mode === 'cloud') { cdelMock.splice(0, cdelMock.length, ...cdelMock.filter(x => x.id !== id)); return { id, name, local: !!p, cloud: p && p.storage !== 'cloud' ? 'skipped' : 'deleted' }; }
      if (p && p.storage === 'cloud') cdelMock.unshift({ id, name, version: p.cloudVersion, updatedAt: now - 600, deviceName: 'DESKTOP-01', size: 31e6, lockedBy: '', localDeleted: true, deletedVersion: p.cloudVersion, deletedAt: now, cloudNewer: false, pendingDelete: false });
      return { id, name, local: !!p };
    });
  },
  CloudOnly: async (): Promise<any> => { await wait(250); return { items: cdelMock.map(x => ({ ...x })) }; },
  RestoreLocalDeleted: async (id: string) => {
    const it = cdelMock.find(x => x.id === id); if (!it) return; await simTransfer(id, it.name, 'down', 31 << 20, 180 << 20, 2500);
    cdelMock.splice(0, cdelMock.length, ...cdelMock.filter(x => x.id !== id));
    mock.profiles.unshift({ id, name: it.name, group: '', tags: [], notes: '', storage: 'cloud', proxyId: '', startUrls: [], cookies: '', fingerprint: mockFP(), createdAt: Date.now() / 1000, lastRun: 0, cloudVersion: it.version, syncedAt: Date.now() / 1000 });
  },
  PendingCloudDeletes: async (): Promise<number> => cdelMock.filter(x => x.pendingDelete).length,
  RetryCloudDeletes: async (): Promise<number> => { await wait(600); return cdelMock.filter(x => x.pendingDelete).length; },
  // ---- recycle bin ----
  ListTrash: async (): Promise<any[]> => { await wait(220); return trashMock.map(x => ({ ...x, expiresAt: x.trashedAt + trashDaysMock * 86400 })).sort((a, b) => b.trashedAt - a.trashedAt); },
  RestoreFromTrash: async (ids: string[]): Promise<any[]> => {
    await wait(500);
    return ids.map(id => {
      const t = trashMock.find(x => x.id === id); if (!t) return { id, name: id, error: '回收站里没有这个环境' };
      trashMock.splice(trashMock.indexOf(t), 1);
      const { trashed, trashedAt, size, hasData, cloudNote, restoreAsNew, ...p } = t;
      const nid = restoreAsNew ? rid() : id;
      mock.profiles.unshift({ notes: '', proxyId: '', startUrls: [], cookies: '', syncedAt: 0, ...p, id: nid, ...(restoreAsNew ? { cloudVersion: 0, storage: 'local' } : {}) });
      return { id: nid, oldId: restoreAsNew ? id : undefined, name: t.name };
    });
  },
  PurgeTrash: async (ids: string[]): Promise<number> => { await wait(400); const n = trashMock.filter(x => ids.includes(x.id)).length; trashMock.splice(0, trashMock.length, ...trashMock.filter(x => !ids.includes(x.id))); return n; },
  EmptyTrash: async (): Promise<number> => { await wait(500); const n = trashMock.length; trashMock.length = 0; return n; },
  TrashDays: async (): Promise<number> => trashDaysMock,
  // ---- session store ----
  SessionStoreInfo: async (id: string): Promise<any> => {
    const p = mock.profiles.find(x => x.id === id); const on = (mock.settings as any).syncSessionStorage !== false;
    if (!p?.lastRun) return { enabled: on, schema: 0, savedAt: 0, cookies: 0, partitioned: 0, origins: 0, lsKeys: 0, idbDbs: 0, idbRecords: 0, idbPartial: false, dropped: 0, bytes: 0, capBytes: ((mock.settings as any).sessionStoreCapMB || 32) * 1048576 };
    return { enabled: on, schema: on ? 2 : 1, savedAt: p.lastRun + 1800, cookies: 47, partitioned: on ? 3 : 0, origins: on ? 12 : 0, lsKeys: on ? 86 : 0, idbDbs: on ? 5 : 0, idbRecords: on ? 1204 : 0, idbPartial: on && /FB|Face/i.test(p.name), dropped: 0, bytes: on ? 1932000 : 41000, capBytes: ((mock.settings as any).sessionStoreCapMB || 32) * 1048576 };
  },
  // ---- 环境自检 / 一键修复 ----
  CheckEnvironment: async (id: string): Promise<any> => {
    await wait(900); const p = mock.profiles.find(x => x.id === id); if (!p) throw new Error('环境不存在');
    const st = envMock.state(id); const fixed = (k: string) => st.fixed.has(k);
    const items: any[] = [
      { key: 'exit', label: '出口 IP / 国家', status: 'pass', actual: '203.0.113.48  US · California · Los Angeles', expected: '代理出口 · US', detail: 'Cox Communications 住宅 IP', fixable: false, fixHint: '' },
      fixed('language') ? { key: 'language', label: '语言', status: 'pass', actual: '跟随 IP → en-US, en', expected: 'en-US, en', detail: '启动时按出口 IP 自动设置', fixable: false, fixHint: '' }
        : { key: 'language', label: '语言', status: 'fail', actual: 'zh-CN, zh, en', expected: 'en-US, en', detail: '首选语言与出口国家不符，容易被识别', fixable: true, fixHint: '改为跟随 IP，并写入出口国家的语言' },
      fixed('timezone') ? { key: 'timezone', label: '时区', status: 'pass', actual: '跟随 IP → America/Los_Angeles', expected: 'America/Los_Angeles（UTC-07:00）', detail: '启动时按出口 IP 自动设置', fixable: false, fixHint: '' }
        : { key: 'timezone', label: '时区', status: 'warn', actual: 'America/Vancouver', expected: 'America/Los_Angeles（UTC-07:00）', detail: 'UTC 偏移相同（UTC-07:00），但时区名不同，夏令时切换日可能露馅', fixable: true, fixHint: '改为跟随 IP，并写入出口 IP 的时区' },
      { key: 'geo', label: '地理位置', status: 'pass', actual: '跟随 IP → 34.0522, -118.2437（Los Angeles）', expected: '34.0522, -118.2437（Los Angeles）', detail: '启动时按出口 IP 自动设置', fixable: false, fixHint: '' },
      fixed('webrtc') ? { key: 'webrtc', label: 'WebRTC 防泄漏', status: 'pass', actual: '替换为代理 IP', expected: '替换为代理 IP / 禁用 UDP', detail: 'WebRTC 不会暴露本机真实 IP', fixable: false, fixHint: '' }
        : { key: 'webrtc', label: 'WebRTC 防泄漏', status: 'fail', actual: '真实', expected: '替换为代理 IP / 禁用 UDP', detail: '使用代理时 WebRTC 为真实模式，会泄漏本机真实 IP', fixable: true, fixHint: '改为「替换为代理 IP」' },
      { key: 'webdriver', label: 'navigator.webdriver', status: 'pass', actual: 'false（内核原生）', expected: 'false', detail: '定制内核原生隐藏自动化标记（按配置判断，未实时读取）', fixable: false, fixHint: '' },
      { key: 'fonts', label: '字体数量', status: 'pass', actual: '49 个', expected: '25–120 个，且属于 Windows 系统', detail: '数量合理', fixable: false, fixHint: '' },
      fixed('seed') ? { key: 'seed', label: 'Canvas / WebGL 噪声种子', status: 'pass', actual: 'Canvas 噪声 · WebGL 噪声 · 种子 1840275531', expected: '已开启噪声且种子非 0', detail: '已配置独立噪声种子', fixable: false, fixHint: '' }
        : { key: 'seed', label: 'Canvas / WebGL 噪声种子', status: 'warn', actual: 'Canvas 噪声 · WebGL 噪声 · 种子 0', expected: '已开启噪声且种子非 0', detail: '种子为 0（仍会按环境 ID 派生），建议生成独立种子', fixable: true, fixHint: '生成新的随机噪声种子' },
      { key: 'ua', label: 'UA / 平台一致性', status: 'pass', actual: p.fingerprint.userAgent || 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.0.0 Safari/537.36', expected: 'Windows · navigator.platform Win32', detail: 'UA、平台、WebGL 渲染器相互一致', fixable: false, fixHint: '' },
    ];
    const clock = await M.CheckSystemClock();
    items.push({ key: 'clock', label: '系统时钟', status: clock.status, actual: clock.status === 'unknown' ? '尚未确认' : `${clock.skewSecs > 0 ? '+' : ''}${clock.skewSecs} 秒`, expected: '偏差 ≤ 2 分钟', detail: clock.detail, fixable: false, fixHint: clock.status === 'fail' ? '系统级设置，只能手动修复' : '' });
    const n = (s: string) => items.filter(i => i.status === s).length;
    return { profileId: id, profileName: p.name, kernel: 'custom', checkedAt: Date.now() / 1000, exit: { ok: true, ip: '203.0.113.48', country: 'US', city: 'Los Angeles' }, items, pass: n('pass'), warn: n('warn'), fail: n('fail'), fixable: items.filter(i => i.fixable && i.status !== 'pass').map(i => i.key), clock };
  },
  RepairEnvironment: async (id: string, items: string[]): Promise<any> => {
    await wait(800); const st = envMock.state(id); const p = mock.profiles.find(x => x.id === id);
    const all = ['language', 'timezone', 'webrtc', 'seed']; const want = (items && items.length ? items : all).filter(k => all.includes(k) && !st.fixed.has(k));
    const skipped = (items || []).filter(k => k === 'clock').map(() => '系统时钟：系统级设置，无法自动修复，请开启系统自动对时');
    if (!want.length) return { entry: null, changed: 0, skipped, profile: p };
    const changes = want.flatMap(k => envMock.changes[k]); want.forEach(k => st.fixed.add(k));
    const entry = { id: rid(), profileId: id, at: Date.now() / 1000, kind: 'repair', items: want, changes, skipped, restored: false };
    st.log.unshift(entry); return { entry, changed: changes.length, skipped, profile: p };
  },
  RepairHistory: async (id: string): Promise<any[]> => { await wait(250); return envMock.state(id).log.map((e: any) => ({ ...e })); },
  RestoreRepair: async (id: string, entryId: string): Promise<any> => {
    await wait(600); const st = envMock.state(id); const e = st.log.find((x: any) => x.id === entryId);
    if (!e) throw new Error('修复记录不存在'); if (e.restored) throw new Error('该记录已恢复过');
    e.restored = true; e.restoredAt = Date.now() / 1000; e.items.forEach((k: string) => st.fixed.delete(k));
    const back = e.changes.map((c: any) => ({ ...c, old: c.new, new: c.old }));
    const r = { id: rid(), profileId: id, at: Date.now() / 1000, kind: 'restore', items: e.items, changes: back, skipped: [], restored: false, restoreOf: e.id };
    st.log.unshift(r); return { entry: r, changed: back.length, skipped: [], profile: mock.profiles.find(x => x.id === id) };
  },
  CheckSystemClock: async (): Promise<any> => {
    await wait(500); const now = Math.floor(Date.now() / 1000); const skew = envMock.clockSkew;
    if (skew === null) return { status: 'unknown', skewSecs: 0, source: '', localTime: now, serverTime: 0, error: 'dial tcp: i/o timeout', detail: '尚未确认：无法连接参考服务器，稍后重试' };
    const a = Math.abs(skew); const txt = a < 60 ? `${a} 秒` : `${Math.floor(a / 60)} 分 ${a % 60} 秒`;
    return a > 120 ? { status: 'fail', skewSecs: skew, source: 'https://www.cloudflare.com/', localTime: now, serverTime: now - skew, error: '', detail: `系统时钟比标准时间${skew > 0 ? '快' : '慢'} ${txt}，超过 2 分钟。请在系统设置里开启「自动设置时间」并立即同步（本程序不会修改系统时间）。` }
      : { status: 'pass', skewSecs: skew, source: 'https://www.cloudflare.com/', localTime: now, serverTime: now - skew, error: '', detail: `偏差 ${txt}，在 2 分钟以内` };
  },
  // ---- fingerprint extras ----
  RegenerateSeed: async (id: string): Promise<number> => { await wait(450); const seed = Math.floor(Math.random() * 2 ** 40) + 1; const p = mock.profiles.find(x => x.id === id); if (p) (p.fingerprint as any).noiseSeed = seed; return seed; },
  NewNoiseSeed: async (): Promise<number> => Math.floor(Math.random() * 2 ** 40) + 1,
  // ---- terminal ----
  TerminalTools: async (): Promise<TerminalTool[]> => { await wait(location.hash.includes('term-slow') ? 60000 : 650); return termMock.tools(); },
  TerminalInfo: async (): Promise<any> => ({ prefs: { ...termMock.prefs }, wt: true, home: 'C:\\Users\\Ghost', os: 'windows' }),
  PickTerminalDir: async (_start: string): Promise<string> => { await wait(300); return 'D:\\Projects\\shop-automation'; },
  CheckTerminalDir: async (dir: string): Promise<string> => { await wait(150); const d = dir.trim().replace(/^"|"$/g, ''); if (!d) return 'C:\\Users\\Ghost'; if (/missing|不存在/i.test(d) || !/^[a-zA-Z]:\\|^~|^\//.test(d)) throw new Error(`目录不存在：${d}`); return d; },
  LaunchTerminal: async (req: any): Promise<TerminalSession> => {
    await wait(700); Object.assign(termMock.prefs, req);
    const t = termMock.tools().find(x => x.id === req.tool); if (!t || !t.installed) throw new Error(`没有找到 ${t?.name || req.tool}：请先安装（${t?.installCmd}）`);
    const p = mock.profiles.find(x => x.id === req.profileId); const px = p && mock.proxies.find(x => x.id === p.proxyId);
    const s: TerminalSession = { id: rid().slice(0, 12), tool: t.id, name: t.name, dir: req.dir || 'C:\\Users\\Ghost', profileId: req.envMode === 'profile' ? req.profileId : '', profileName: req.envMode === 'profile' ? p?.name || '' : '',
      tz: req.envMode === 'profile' ? p?.fingerprint.timezone || '' : req.envMode === 'custom' ? req.tz : '', lang: req.envMode === 'profile' ? (p?.fingerprint.languages || [])[0] || '' : req.envMode === 'custom' ? req.lang : '',
      proxy: req.envMode === 'profile' && px ? `${px.type.toUpperCase()} ${px.host}:${px.port}` : '', relay: req.envMode === 'profile' && px && (px.user || px.type.startsWith('socks')) ? '127.0.0.1:' + (40000 + Math.floor(Math.random() * 9999)) : '',
      host: req.host || 'console', pid: 1000 + Math.floor(Math.random() * 30000), tracked: req.host !== 'wt', started: Date.now() / 1000 };
    termMock.sessions.unshift(s); mockEmit('terminal:changed', [...termMock.sessions]); return s;
  },
  ListTerminals: async (): Promise<TerminalSession[]> => [...termMock.sessions],
  CloseTerminal: async (id: string) => { await wait(300); termMock.sessions = termMock.sessions.filter(x => x.id !== id); mockEmit('terminal:changed', [...termMock.sessions]); },
  // ---- firefox kernel ----
  _ffInit: (() => { if (!inWails) { const fp = { ...mockFP('windows'), userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0', hardwareConcurrency: 4, ffFontVisibility: 1, gpc: true, languageFromIP: true }; mock.profiles.splice(2, 0, { id: 'ff-demo', name: 'Reddit-Firefox-01', group: '社媒', tags: ['Firefox'], notes: '', storage: 'local', proxyId: 'px2', startUrls: ['https://www.reddit.com'], cookies: '', fingerprint: fp, kernel: 'firefox', createdAt: Date.now() / 1000 - 7200, lastRun: 0, cloudVersion: 0, syncedAt: 0 }); } return 0; })(),
  FirefoxInfo: async (): Promise<any> => {
    const none = location.hash.includes('nofirefox') && !ffMock.installed;
    const auto = none ? '' : ffMock.installed || 'C:\\Program Files\\Mozilla Firefox\\firefox.exe';
    const cur = (mock.settings.firefoxPath || '').trim();
    const missing = !!cur && !/firefox\.exe$/i.test(cur);
    const path = cur && !missing ? cur : auto;
    const version = !path ? '' : /esr/i.test(path) || ffMock.installed ? '128.5.0esr' : '133.0.3';
    return { path, auto, version, esr: /esr/.test(version), custom: !!cur, missing, downloadUrl: 'https://www.mozilla.org/zh-CN/firefox/new/' };
  },
  PickFirefox: async (): Promise<string> => 'D:\\Portable\\FirefoxESR\\firefox.exe',
  FirefoxDownloadPage: async () => { window.open('https://www.mozilla.org/zh-CN/firefox/new/', '_blank'); },
  FirefoxInstallStatus: async (): Promise<any> => ({ ...ffMock.p }),
  FirefoxInstallCancel: async () => { ffMock.cancel = true; },
  FirefoxInstallStart: async (channel: string): Promise<any> => { if (ffMock.p.running) throw 'Firefox 正在安装'; simFirefox(channel); return { ...ffMock.p }; },
  CreateTypedProfilesKernel: async (prefix: string, t: string, os: string, n: number, proxyId: string, group: string, kernel: string): Promise<any[]> => {
    const out: any[] = await (M as any).CreateTypedProfiles(prefix, kernel === 'firefox' && !['desktop', 'laptop'].includes(t) ? 'desktop' : t, os, n, proxyId, group);
    if (kernel === 'firefox') out.forEach(p => { p.kernel = 'firefox'; p.fingerprint = { ...p.fingerprint, userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0' }; });
    return out;
  },
};
// ---- firefox kernel ---- (install simulation for the preview)
const ffMock = { p: { phase: 'idle', running: false, source: 'esr', done: 0, total: 0, pct: 0, speed: 0, eta: -1, message: '' } as any, cancel: false, installed: '' };
function simFirefox(channel: string) {
  const E = (x: any) => { ffMock.p = { ...ffMock.p, ...x }; mockEmit('firefox:install', { ...ffMock.p }); };
  const ver = channel === 'release' ? '133.0.3' : '128.5.0esr', total = 71_303_168;
  ffMock.cancel = false;
  E({ phase: 'resolve', running: true, source: channel, done: 0, total: 0, pct: 0, speed: 0, eta: -1, message: '正在获取官方下载地址…', hint: '', error: '', version: '', path: '' });
  setTimeout(() => {
    E({ phase: 'download', version: ver, revision: ver, total, message: '正在下载 Firefox 官方安装包…' });
    let done = 0;
    const iv = setInterval(() => {
      if (ffMock.cancel) { clearInterval(iv); return E({ phase: 'cancelled', running: false, speed: 0, eta: -1, message: '已取消安装', hint: '已下载的部分会保留，重新安装时从断点继续' }); }
      const sp = 6.5 * 1048576; done = Math.min(total, done + sp / 4);
      E({ done, pct: done * 100 / total, speed: sp, eta: Math.round((total - done) / sp) });
      if (done >= total) {
        clearInterval(iv);
        E({ phase: 'verify', done: 0, total: 0, pct: 0, message: '正在校验 SHA-512…' });
        setTimeout(() => {
          E({ phase: 'extract', message: '正在静默安装 Firefox…' });
          setTimeout(() => {
            const path = `C:\\Users\\me\\AppData\\Roaming\\GhostBrowser\\firefox\\${channel === 'release' ? 'release' : 'esr'}\\firefox.exe`;
            ffMock.installed = path; mock.settings.firefoxPath = path;
            E({ phase: 'done', running: false, path, pct: 100, speed: 0, eta: 0, message: '安装完成，已设为 Firefox 内核' });
          }, 1600);
        }, 900);
      }
    }, 250);
  }, 800);
}

export const api: typeof M = inWails ? (Go as any) : M;

// In the browser preview, events come from a local bus so the mock can simulate them.
const bus: Record<string, Set<(...a: any[]) => void>> = {};
export const mockEmit = (name: string, ...a: any[]) => bus[name]?.forEach(f => f(...a));
export const onEvent = (name: string, cb: (...a: any[]) => void) => {
  const rt = (window as any).runtime;
  if (rt && rt.EventsOn) return rt.EventsOn(name, cb);
  (bus[name] ||= new Set()).add(cb);
  return () => { bus[name].delete(cb); };
};

// Simulated transfer for the preview: phases, bytes and a wobbly speed.
function simTransfer(id: string, name: string, dir: 'up' | 'down', total: number, saved: number, ms: number) {
  const started = Date.now();
  return new Promise<void>(res => {
    const pre = dir === 'up' ? 'scan' : 'download';
    mockEmit('sync:progress', { id, name, dir, phase: pre, done: 0, total: 0, speed: 0, eta: -1, saved, started, final: false });
    let done = 0;
    setTimeout(() => {
      const ph = dir === 'up' ? 'upload' : 'download';
      const iv = setInterval(() => {
        const sp = (total / (ms / 1000)) * (0.7 + Math.random() * 0.6);
        done = Math.min(total, done + sp * 0.2);
        mockEmit('sync:progress', { id, name, dir, phase: ph, done, total, speed: sp, eta: Math.round((total - done) / sp), saved, started, final: false });
        if (done >= total) {
          clearInterval(iv);
          mockEmit('sync:progress', { id, name, dir, phase: 'done', done: total, total, speed: total / (ms / 1000), eta: 0, saved, started, final: true });
          res();
        }
      }, 200);
    }, 700);
  });
}

// ---------------- rpa mock ----------------
const RNOW = Math.floor(Date.now() / 1000);
const rpaMock = {
  wfs: [
    { id: 'wf-amz', name: '亚马逊商品价格采集', description: '搜索关键词，翻 5 页采集标题与价格，低于目标价时截图留证', vars: { keyword: 'wireless earbuds', target: '25' }, createdAt: RNOW - 86400 * 6, updatedAt: RNOW - 7200, lastRunAt: RNOW - 1260, lastStatus: 'ok', lastRunId: 'run-a2', runCount: 14, steps: [
      { id: 'a1', type: 'goto', params: { url: 'https://www.amazon.com/s?k={{keyword}}', timeout: '30000' } },
      { id: 'a2', type: 'wait_for', params: { selector: '[data-component-type="s-search-result"]', state: 'visible', timeout: '15000' } },
      { id: 'a3', type: 'loop', params: { mode: 'count', count: '5', indexVar: 'page' }, children: [
        { id: 'a4', type: 'extract_text', name: '读取首个商品价格', params: { selector: '.s-result-item .a-price .a-offscreen', var: 'price', attr: '', trim: 'true' } },
        { id: 'a5', type: 'condition', params: { kind: 'js', code: 'return parseFloat(vars.price.replace(/[^0-9.]/g, "")) < Number(vars.target);' }, children: [
          { id: 'a6', type: 'screenshot', name: '低价截图', params: { fullPage: 'false', saveAs: 'deal_shot' } },
        ], else: [
          { id: 'a7', type: 'delay', params: { ms: '800', maxMs: '1600' } },
        ] },
        { id: 'a8', type: 'condition', params: { kind: 'not_exists', selector: 'a.s-pagination-next' }, children: [{ id: 'a9', type: 'break_loop', params: {} }], else: [] },
        { id: 'a10', type: 'click', params: { selector: 'a.s-pagination-next', double: 'false' } },
        { id: 'a11', type: 'scroll', params: { mode: 'by', x: '0', y: '900' } },
      ] },
      { id: 'a12', type: 'execute_js', params: { code: 'return document.querySelectorAll(".s-result-item").length;', saveAs: 'items' } },
    ] },
    { id: 'wf-gm', name: 'Gmail 养号 · 每日浏览', description: '登录检查、随机阅读邮件、滚动停留，模拟真人使用', vars: {}, createdAt: RNOW - 86400 * 12, updatedAt: RNOW - 86400, lastRunAt: RNOW - 5400, lastStatus: 'partial', lastRunId: 'run-g1', runCount: 37, steps: [
      { id: 'g1', type: 'goto', params: { url: 'https://mail.google.com/mail/u/0/#inbox', timeout: '30000' } },
      { id: 'g2', type: 'condition', params: { kind: 'exists', selector: 'input[type=email]' }, children: [
        { id: 'g3', type: 'fill', params: { selector: 'input[type=email]', text: '{{email}}', clear: 'true' } },
        { id: 'g4', type: 'click', params: { selector: 'text:下一步' } },
      ], else: [] },
      { id: 'g5', type: 'loop', params: { mode: 'count', count: '3', indexVar: 'n' }, children: [
        { id: 'g6', type: 'click', params: { selector: 'xpath://tr[contains(@class,"zA")][{{n}}]' } },
        { id: 'g7', type: 'delay', params: { ms: '3000', maxMs: '8000' } },
        { id: 'g8', type: 'scroll', params: { mode: 'by', x: '0', y: '700' } },
      ] },
      { id: 'g9', type: 'select_tab', disabled: true, params: { index: '1' } },
    ] },
    { id: 'wf-tt', name: 'TikTok Shop 订单导出', description: '', vars: {}, createdAt: RNOW - 86400 * 2, updatedAt: RNOW - 86400 * 2, lastRunAt: RNOW - 86400, lastStatus: 'failed', lastRunId: 'run-t1', runCount: 3, steps: [
      { id: 't1', type: 'goto', params: { url: 'https://seller-us.tiktok.com/order', timeout: '30000' } },
      { id: 't2', type: 'click', continueOnError: true, params: { selector: 'text:导出订单' } },
      { id: 't3', type: 'upload', params: { selector: 'input[type=file]', files: 'D:\\exports\\template.xlsx' } },
    ] },
    { id: 'wf-fb', name: 'Facebook 广告账户巡检（新建，未运行）', description: '打开广告管理器，检查账户状态是否受限，提取余额并截图存档，超长名称用于测试省略显示效果', vars: {}, createdAt: RNOW - 3600, updatedAt: RNOW - 3600, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0, steps: [
      { id: 'f1', type: 'goto', params: { url: 'https://adsmanager.facebook.com/', timeout: '30000' } },
      { id: 'f2', type: 'extract_text', params: { selector: '', var: 'balance', attr: '', trim: 'true' } },
    ] },
    { id: 'wf-pay', name: '独立站结账 · 信用卡 iframe', description: '在支付网关的跨域 iframe 里填写卡号并提交，读取支付结果', vars: { card: '4242 4242 4242 4242' }, createdAt: RNOW - 86400, updatedAt: RNOW - 3600, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0, steps: [
      { id: 'p1', type: 'goto', params: { url: 'https://shop.example.com/checkout', timeout: '30000' } },
      { id: 'p2', type: 'fill', params: { selector: '#email', text: '{{email}}', clear: 'true' } },
      { id: 'p3', type: 'switch_frame', params: { mode: 'frame', frame: 'iframe[title="Secure card payment input frame"]', timeout: '10000' } },
      { id: 'p4', type: 'fill', params: { selector: 'input[name=cardnumber]', text: '{{card}}', clear: 'true' } },
      { id: 'p5', type: 'fill', params: { selector: 'input[name=exp-date]', text: '12 / 29', clear: 'true' } },
      { id: 'p6', type: 'switch_frame', params: { mode: 'top' } },
      { id: 'p7', type: 'click', params: { selector: 'button#submit', double: 'false' } },
      { id: 'p8', type: 'wait_for', params: { selector: 'text:支付成功', state: 'visible', timeout: '20000', frame: 'iframe#pay-result >> iframe[name=status]' } },
      { id: 'p9', type: 'extract_text', params: { selector: '.order-no', var: 'order_no', attr: '', trim: 'true', frame: 'iframe#pay-result' } },
    ] },
  ] as any[],
  runs: [] as any[],
};
const mkTarget = (p: any, status: string, steps: any[], vars: any, err = '') => ({ profileId: p?.id || rid(), profileName: p?.name || '已删除的环境', status, error: err, startedAt: RNOW - 1300, endedAt: RNOW - 1240, autoStarted: false, steps, vars });
rpaMock.runs.push(
  { id: 'run-a2', workflowId: 'wf-amz', workflowName: '亚马逊商品价格采集', status: 'ok', startedAt: RNOW - 1300, endedAt: RNOW - 1228, concurrency: 3, stepCount: 12, targets: [mkTarget(mock.profiles[0], 'ok', [{ stepId: 'a1', index: 0, type: 'goto', status: 'ok', log: 'https://www.amazon.com/s?k=wireless+earbuds', ms: 2140 }, { stepId: 'a12', index: 11, type: 'execute_js', status: 'ok', log: '48', ms: 31, vars: { price: '$22.99', items: '48' } }], { price: '$22.99', items: '48', page: '5' })] },
  { id: 'run-a1', workflowId: 'wf-amz', workflowName: '亚马逊商品价格采集', status: 'partial', startedAt: RNOW - 86400 - 600, endedAt: RNOW - 86400 - 480, concurrency: 5, stepCount: 12, targets: [mkTarget(mock.profiles[0], 'ok', [], {}), mkTarget(mock.profiles[3], 'failed', [], {}, '等待元素：等待 [data-component-type="s-search-result"] 超时（15 秒）')] },
  { id: 'run-a0', workflowId: 'wf-amz', workflowName: '亚马逊商品价格采集', status: 'cancelled', startedAt: RNOW - 86400 * 3, endedAt: RNOW - 86400 * 3 + 40, concurrency: 1, stepCount: 11, targets: [mkTarget(mock.profiles[1], 'cancelled', [], {}, '已取消')] },
  { id: 'run-g1', workflowId: 'wf-gm', workflowName: 'Gmail 养号 · 每日浏览', status: 'partial', startedAt: RNOW - 5500, endedAt: RNOW - 5400, concurrency: 3, stepCount: 9, targets: [mkTarget(mock.profiles[5], 'ok', [], {}), mkTarget(mock.profiles[4], 'failed', [], {}, '浏览器未打开：请先启动这个环境，或勾选「自动启动未运行的环境」')] },
  { id: 'run-t1', workflowId: 'wf-tt', workflowName: 'TikTok Shop 订单导出', status: 'failed', startedAt: RNOW - 86460, endedAt: RNOW - 86400, concurrency: 1, stepCount: 3, targets: [mkTarget(mock.profiles[4], 'failed', [], {}, '上传文件：找不到元素 input[type=file]')] },
);
const rpaPub = (r: any) => { const { cancel, ...x } = r; return JSON.parse(JSON.stringify(x)); };
function rpaValidate(w: any): any[] {
  const out: any[] = [];
  if (!String(w.name || '').trim()) out.push({ stepId: '', message: '流程名称不能为空' });
  if (!(w.steps || []).length) out.push({ stepId: '', message: '流程里还没有步骤' });
  const names: Record<string, string> = { goto: '打开网址', click: '点击', fill: '输入文本', extract_text: '提取文本', execute_js: '执行 JS', upload: '上传文件', select_tab: '切换标签页', condition: '条件判断', loop: '循环', switch_frame: '切换框架' };
  const go = (l: any[], inLoop: boolean) => (l || []).forEach(s => {
    const p = s.params || {}; const lab = s.name || names[s.type] || s.type;
    const req = (k: string, what: string) => { if (!String(p[k] || '').trim()) out.push({ stepId: s.id, message: `${lab}：缺少${what}` }); };
    if (!s.disabled) {
      if (s.type === 'goto') { req('url', '网址'); if (p.url === 'https://') out.push({ stepId: s.id, message: `${lab}：缺少网址` }); }
      if (s.type === 'click' || s.type === 'fill') req('selector', '选择器');
      if (s.type === 'scroll' && p.mode === 'el') req('selector', '元素选择器');
      for (const k of ['var', 'saveAs', 'indexVar']) { const n = String(p[k] || '').trim(); if (n && !/^[A-Za-z_\u4e00-\u9fff][\w\u4e00-\u9fff.-]*$/.test(n)) out.push({ stepId: s.id, message: `${lab}：变量名「${n}」要以字母、下划线或汉字开头` }); }
      if (s.type === 'extract_text') { req('selector', '选择器'); req('var', '变量名'); }
      if (s.type === 'execute_js') req('code', '代码');
      if (s.type === 'upload') { req('selector', '选择器'); req('files', '文件路径'); }
      if (s.type === 'select_tab') req('index', '标签页序号');
      if (s.type === 'switch_frame' && p.mode === 'frame') req('frame', 'iframe 选择器');
      const cond = () => p.kind === 'js' ? req('code', '判断代码') : (p.kind === 'equals' || p.kind === 'contains') ? req('left', '比较值') : req('selector', '选择器');
      if (s.type === 'condition') cond();
      if (s.type === 'loop' && p.mode === 'while') cond();
      if (s.type === 'break_loop' && !inLoop) out.push({ stepId: s.id, message: '「跳出循环」必须放在循环里' });
    }
    go(s.children, inLoop || s.type === 'loop'); go(s.else, inLoop);
  });
  go(w.steps, false);
  return out;
}
// A stylised page thumbnail so screenshots look like something in the preview.
function rpaShot(path: string) {
  let h = 0; for (const c of path) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  const hue = h % 360;
  const cards = [0, 1, 2].map(i => `<rect x="${24 + i * 196}" y="120" width="180" height="200" rx="10" fill="hsl(${hue},30%,${18 + i * 3}%)"/><rect x="${36 + i * 196}" y="250" width="120" height="10" rx="5" fill="#cfd6ff" opacity=".6"/><rect x="${36 + i * 196}" y="270" width="60" height="12" rx="6" fill="#3ee6a8"/>`).join('');
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="640" height="400"><rect width="640" height="400" fill="#0d1017"/><rect width="640" height="44" fill="hsl(${hue},40%,22%)"/><rect x="16" y="14" width="220" height="16" rx="8" fill="#fff" opacity=".25"/><rect x="24" y="66" width="300" height="22" rx="6" fill="#fff" opacity=".12"/>${cards}<rect x="24" y="340" width="592" height="40" rx="10" fill="#fff" opacity=".05"/></svg>`;
  return 'data:image/svg+xml;base64,' + btoa(svg);
}
// Simulated run: walks the tree (taking the "then" branch, honouring loop counts and break_loop) and emits rpa:event like rpa_run.go.
function rpaSimRun(wfId: string, ids: string[], opt: any): string {
  const wf = rpaMock.wfs.find(w => w.id === wfId); if (!wf) throw '流程不存在或已删除';
  const issues = rpaValidate(wf); if (issues.length) throw `流程有 ${issues.length} 处需要修改：${issues[0].message}`;
  if (!ids.length) throw '请至少选择一个环境';
  const flatAll: any[] = []; const fw = (l: any[]) => (l || []).forEach(s => { flatAll.push(s); fw(s.children); fw(s.else); }); fw(wf.steps);
  const index: Record<string, number> = {}; flatAll.forEach((s, i) => (index[s.id] = i));
  const now = () => Math.floor(Date.now() / 1000);
  const run: any = { id: 'run-' + rid().slice(0, 8), workflowId: wf.id, workflowName: wf.name, status: 'running', startedAt: now(), endedAt: 0, concurrency: opt?.concurrency || 1, stepCount: flatAll.length, targets: ids.map(id => ({ profileId: id, profileName: mock.profiles.find(p => p.id === id)?.name || id, status: 'pending', error: '', startedAt: 0, endedAt: 0, autoStarted: false, steps: [], vars: {} })) };
  rpaMock.runs.unshift(run); Object.assign(wf, { lastRunAt: run.startedAt, lastStatus: 'running', lastRunId: run.id, runCount: wf.runCount + 1 });
  const E = (e: any) => mockEmit('rpa:event', { runId: run.id, index: 0, ...e });
  E({ profileId: '', phase: 'run', status: 'running' });
  const sleep = (ms: number) => new Promise(r => setTimeout(r, ms));
  const logOf = (s: any, vars: any, iter: number) => { const p = s.params || {}; const fill = (t: string) => String(t || '').replace(/\{\{\s*(\w+)\s*\}\}/g, (_, k) => vars[k] ?? ''); switch (s.type) {
    case 'goto': return fill(p.url); case 'click': return fill(p.selector); case 'fill': return `${p.selector} ← ${fill(p.text)}`; case 'delay': return `${p.ms} ms`;
    case 'wait_for': return `${p.selector} 已可见`; case 'scroll': return `滚动 (${p.x || 0}, ${p.y || 600})`; case 'extract_text': return `${p.var} = ${vars[p.var]}`;
    case 'execute_js': return String(vars[p.saveAs] ?? '48'); case 'screenshot': return '已保存截图'; case 'condition': return '条件成立 → 则'; case 'loop': return `完成 ${iter} 轮`; default: return ''; } };
  const target = async (t: any, ti: number) => {
    const failAt = ti === 1 ? flatAll.findIndex(s => s.type === 'click' || s.type === 'upload') : -1;
    if (!(mock.running.some(r => r.profileId === t.profileId))) {
      if (!opt?.autoStart) { Object.assign(t, { status: 'failed', error: '浏览器未打开：请先启动这个环境，或勾选「自动启动未运行的环境」', endedAt: now() }); E({ profileId: t.profileId, phase: 'target', status: 'failed', error: t.error }); return; }
      Object.assign(t, { status: 'starting', startedAt: now(), autoStarted: true }); E({ profileId: t.profileId, phase: 'target', status: 'starting' }); await sleep(700);
    }
    t.status = 'running'; t.startedAt ||= now(); E({ profileId: t.profileId, phase: 'target', status: 'running' });
    const vars: any = { ...(wf.vars || {}) }; let iter = 0; let failed = '';
    const BRK = {};
    const exec = async (s: any): Promise<void> => {
      if (run.cancel) throw new Error('已取消');
      const base = { profileId: t.profileId, phase: 'step', stepId: s.id, index: index[s.id], type: s.type, iter };
      if (s.disabled) { const ev = { ...base, status: 'skipped', log: '已停用', ms: 0 }; t.steps.push(ev); E(ev); return; }
      E({ ...base, status: 'running' });
      const t0 = Date.now();
      if (s.type === 'loop') {
        const n = s.params.mode === 'while' ? 3 : Math.min(+s.params.count || 1, 6); const outer = iter; let done = 0;
        try { for (let i = 1; i <= n; i++) { iter = i; vars[s.params.indexVar || 'loop_index'] = String(i); done = i; for (const c of s.children || []) await exec(c); } }
        catch (e) { if (e !== BRK) { iter = outer; throw e; } }
        iter = outer;
        const ev = { ...base, iter, status: 'ok', log: `完成 ${done} 轮`, ms: Date.now() - t0, vars: { ...vars } }; t.steps.push(ev); E(ev); return;
      }
      await sleep(s.type === 'delay' ? 520 : s.type === 'goto' ? 640 : 260 + Math.random() * 240);
      if (run.cancel) { const ev = { ...base, status: 'failed', log: '已取消', ms: Date.now() - t0 }; t.steps.push(ev); E(ev); throw new Error('已取消'); }
      if (index[s.id] === failAt && !failed) {
        const msg = `找不到元素 ${s.params.selector}（等待 8 秒）`;
        const ev = { ...base, status: 'failed', log: msg + (s.continueOnError ? '（已设置出错继续）' : ''), ms: Date.now() - t0, vars: { ...vars } }; t.steps.push(ev); E(ev);
        if (!s.continueOnError) { failed = `${s.name || ({ click: '点击', upload: '上传文件' } as any)[s.type] || s.type}：${msg}`; throw new Error(failed); }
        return;
      }
      if (s.type === 'break_loop') { const ev = { ...base, status: 'ok', log: '跳出循环', ms: 1 }; t.steps.push(ev); E(ev); throw BRK; }
      if (s.type === 'extract_text') vars[s.params.var] = s.params.var === 'price' ? `$${(18 + Math.random() * 14).toFixed(2)}` : '1,280.50 USD';
      if (s.type === 'execute_js' && s.params.saveAs) vars[s.params.saveAs] = '48';
      const shot = s.type === 'screenshot' ? `C:\\GhostBrowser\\rpa\\shots\\${run.id}\\${t.profileId}-${index[s.id] + 1}-${Date.now()}.png` : '';
      if (shot && s.params.saveAs) vars[s.params.saveAs] = shot;
      const ev: any = { ...base, status: 'ok', log: logOf(s, vars, iter), ms: Date.now() - t0, vars: { ...vars }, shot };
      t.steps.push(ev); E(ev);
      if (s.type === 'condition') {
        const branch = s.params.kind === 'not_exists' && iter < 3 ? s.else : s.children;
        for (const c of branch || []) await exec(c);
      }
    };
    try { for (const s of wf.steps) await exec(s); Object.assign(t, { status: 'ok', vars }); }
    catch (e: any) { Object.assign(t, { status: run.cancel ? 'cancelled' : 'failed', error: run.cancel ? '已取消' : String(e?.message || e), vars }); }
    t.endedAt = now(); E({ profileId: t.profileId, phase: 'target', status: t.status, error: t.error, vars });
    if (t.autoStarted && opt?.closeAfter) mock.running = mock.running.filter(r => r.profileId !== t.profileId);
  };
  (async () => {
    const queue = run.targets.map((t: any, i: number) => [t, i]);
    const worker = async () => { while (queue.length) { const [t, i] = queue.shift(); await target(t, i); } };
    await Promise.all(Array.from({ length: Math.max(1, Math.min(run.concurrency, run.targets.length)) }, worker));
    const st = run.targets.map((t: any) => t.status); const ok = st.filter((s: string) => s === 'ok').length;
    run.status = run.cancel ? 'cancelled' : ok === st.length ? 'ok' : ok === 0 ? 'failed' : 'partial'; run.endedAt = now();
    wf.lastStatus = run.status;
    E({ profileId: '', phase: 'run', status: run.status });
  })();
  return run.id;
}

if (!inWails) (window as any).__mock = { api: M, emit: mockEmit };

// ---- extensions / cookies / mihomo mock state ----
function extIcon(bg: string, ch: string) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${bg}"/><stop offset="1" stop-color="${bg}" stop-opacity=".72"/></linearGradient></defs><rect width="64" height="64" rx="14" fill="url(#g)"/><text x="32" y="43" font-family="Arial, sans-serif" font-size="30" font-weight="700" text-anchor="middle" fill="#fff">${ch}</text></svg>`;
  return 'data:image/svg+xml;base64,' + btoa(unescape(encodeURIComponent(svg)));
}
const UBO_ICON = 'data:image/svg+xml;base64,' + btoa('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><path d="M32 4 8 12v18c0 15 10 26 24 30 14-4 24-15 24-30V12Z" fill="#800000"/><path d="M32 10 14 16v14c0 11 7 20 18 24 11-4 18-13 18-24V16Z" fill="#c00"/><text x="32" y="40" font-family="Arial" font-size="20" font-weight="700" text-anchor="middle" fill="#fff">uO</text></svg>');
const MM_ICON = 'data:image/svg+xml;base64,' + btoa('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64"><rect width="64" height="64" rx="14" fill="#fff4e8"/><path d="M10 12 28 25l-3-8Zm44 0L36 25l3-8ZM16 44l5 8 9-3-1-6Zm32 0-5 8-9-3 1-6ZM20 30l-4 6 12 1Zm24 0 4 6-12 1Z" fill="#e2761b"/><path d="M18 22 28 25 30 40 22 44 14 36Zm28 0-10 3-2 15 8 4 8-8Z" fill="#f6851b"/><path d="M28 40h8l-1 6h-6Z" fill="#233447"/></svg>');
const extMock = { list: [
  { id: 'cjpalhdlnbpafiamejdnhcphjbkeiagm', name: 'uBlock Origin', version: '1.62.0', description: '一款高效的广告拦截器：占用极少的 CPU 和内存，同时可以加载并执行数千条过滤规则。', source: 'store', origin: 'https://chromewebstore.google.com/detail/cjpalhdlnbpafiamejdnhcphjbkeiagm', dir: '', icon: UBO_ICON, manifestVersion: 2, size: 3_900_000, addedAt: Date.now() / 1000 - 86400 * 6, profiles: 0 },
  { id: 'nkbihfbeogaeaoehlefnkodbefgpgknn', name: 'MetaMask', version: '12.20.1', description: '以太坊浏览器钱包：在浏览器中安全地管理数字资产、连接去中心化应用并签署交易。', source: 'store', origin: 'https://chromewebstore.google.com/detail/nkbihfbeogaeaoehlefnkodbefgpgknn', dir: '', icon: MM_ICON, manifestVersion: 3, size: 21_400_000, addedAt: Date.now() / 1000 - 86400 * 3, profiles: 0 },
  { id: 'nngceckbapebfimnlniiiahkandclblb', name: 'Bitwarden 密码管理器', version: '2025.9.0', description: '在家中、工作中或旅途中，Bitwarden 都能轻松保护您的所有密码、通行密钥和敏感信息。', source: 'store', origin: 'https://chromewebstore.google.com/detail/nngceckbapebfimnlniiiahkandclblb', dir: '', icon: extIcon('#175ddc', 'B'), manifestVersion: 3, size: 15_800_000, addedAt: Date.now() / 1000 - 86400 * 2, profiles: 0 },
  { id: 'kbfnbcaeplbcioakkpcpgfkobkghlhen', name: 'Grammarly: AI Writing and Grammar Checker App', version: '14.1251.0', description: '借助 Grammarly 的 AI 写作助手提升写作水平：语法检查、改写、语气调整，适用于你在网页上书写的任何内容。', source: 'url', origin: 'https://cdn.example.com/crx/grammarly-14.1251.0.crx', dir: '', icon: extIcon('#15c39a', 'G'), manifestVersion: 3, size: 9_600_000, addedAt: Date.now() / 1000 - 86400, profiles: 0 },
  { id: 'aapbdbdomjkkjkaonfhkkikfgjllcleb', name: '翻译助手（内部版）', version: '0.9.4', description: '团队内部打包的划词翻译扩展，从本地 .zip 安装。', source: 'file', origin: 'D:\\tools\\translator-0.9.4.zip', dir: '', icon: extIcon('#8b5cf6', '译'), manifestVersion: 3, size: 640_000, addedAt: Date.now() / 1000 - 3600 * 5, profiles: 0 },
] as any[] };
mock.profiles.forEach((p, i) => { p.extensionIds = i < 4 ? ['cjpalhdlnbpafiamejdnhcphjbkeiagm'] : []; if (i === 1 || i === 2) p.extensionIds.push('nkbihfbeogaeaoehlefnkodbefgpgknn'); });
const cookieMock = { staged: {} as Record<string, number> };
const mihomoMock = {
  status: { configured: false, connected: false, controller: '', hasSecret: false, version: 'v1.19.13', meta: true, mode: 'rule', listenHost: '127.0.0.1', listenPort: 7890, listenKind: 'mixed', error: '' } as any,
  groups: [
    { name: '🚀 节点选择', type: 'Selector', now: '🇺🇸 美国 洛杉矶 01', all: ['♻️ 自动选择', '🇺🇸 美国 洛杉矶 01', '🇺🇸 美国 圣何塞 02', '🇯🇵 日本 东京 01', '🇯🇵 日本 大阪 02', '🇸🇬 新加坡 01', '🇭🇰 香港 IPLC 01', '🇩🇪 德国 法兰克福 01', '🇬🇧 英国 伦敦 01', '⚠️ 过期节点 测试'], delay: 0, alive: true, udp: true, isGroup: true },
    { name: '♻️ 自动选择', type: 'URLTest', now: '🇭🇰 香港 IPLC 01', all: ['🇺🇸 美国 洛杉矶 01', '🇯🇵 日本 东京 01', '🇸🇬 新加坡 01', '🇭🇰 香港 IPLC 01'], delay: 0, alive: true, udp: true, isGroup: true },
    { name: '🛒 电商平台', type: 'Selector', now: '🇺🇸 美国 圣何塞 02', all: ['🚀 节点选择', '🇺🇸 美国 洛杉矶 01', '🇺🇸 美国 圣何塞 02', '🇬🇧 英国 伦敦 01', '🇩🇪 德国 法兰克福 01'], delay: 0, alive: true, udp: true, isGroup: true },
    { name: '🐟 漏网之鱼', type: 'Fallback', now: '🇯🇵 日本 东京 01', all: ['🇯🇵 日本 东京 01', '🇸🇬 新加坡 01'], delay: 0, alive: true, udp: true, isGroup: true },
  ] as any[],
  nodes: [
    ['🇺🇸 美国 洛杉矶 01', 'Vmess', 168], ['🇺🇸 美国 圣何塞 02', 'Trojan', 182], ['🇯🇵 日本 东京 01', 'Hysteria2', 74], ['🇯🇵 日本 大阪 02', 'Vless', 96],
    ['🇸🇬 新加坡 01', 'Shadowsocks', 121], ['🇭🇰 香港 IPLC 01', 'Shadowsocks', 38], ['🇩🇪 德国 法兰克福 01', 'Vless', 241], ['🇬🇧 英国 伦敦 01', 'Trojan', 0], ['⚠️ 过期节点 测试', 'Vmess', 0],
  ].map(([name, type, delay]) => ({ name, type, delay, alive: delay !== 0 || name === '🇬🇧 英国 伦敦 01', udp: type !== 'Trojan', isGroup: false, now: '', all: [] })) as any[],
};
if (!inWails && location.hash.includes('mihomo')) mihomoMock.status = { ...mihomoMock.status, configured: true, connected: true, controller: 'http://127.0.0.1:9090', hasSecret: true };

// ---- kernel install mock ----
const KDIR = 'C:\\Users\\me\\AppData\\Roaming\\GhostBrowser\\kernels\\';
const kernelMock = {
  p: { phase: 'idle', running: false, source: 'mirror', done: 0, total: 0, pct: 0, speed: 0, eta: -1, message: '' } as any,
  cancel: false,
  list: (!inWails && location.hash.includes('nokernel') ? [] : [{ name: 'chromium-155.0.7980.0', version: '155.0.7980.0', path: KDIR + 'chromium-155.0.7980.0\\chrome.exe', dir: KDIR + 'chromium-155.0.7980.0', size: 861_204_480, installed: Date.now() - 86400e3 * 9, active: false }]) as any[],
};
// Simulated install: resolve → download (wobbly speed) → extract → verify → done.
// ?#kfail makes the download fail at ~40%; the "official" source fails with HTTP 403.
function simKernel(source: string) {
  const K = kernelMock; K.cancel = false;
  const E = (patch: any) => { K.p = { ...K.p, ...patch }; mockEmit('kernel:install', { ...K.p }); };
  const total = 360_101_018, unpacked = 855_087_658, rev = '1710741', ver = '157.0.8086.0';
  const failAt = location.hash.includes('kfail') ? 0.4 : 2;
  const slow = location.hash.includes('kslow') ? 8 : 1;
  E({ phase: 'resolve', running: true, source, revision: '', version: '', path: '', done: 0, total: 0, pct: 0, speed: 0, eta: -1, resumed: false, message: '正在解析最新版本…', hint: '', error: '' });
  const stop = (msg: string, hint: string, err: string) => E({ phase: 'error', running: false, speed: 0, eta: -1, message: msg, hint, error: err });
  setTimeout(() => {
    if (K.cancel) return E({ phase: 'cancelled', running: false, message: '已取消安装', hint: '已下载的部分会保留，重新安装时从断点继续' });
    if (source === 'official') return stop('下载版本列表失败: HTTP 403', '该下载源无法访问此文件，请切换到「国内镜像」或填写自定义下载地址', '下载版本列表失败: HTTP 403');
    let done = K.p.phase === 'cancelled' ? 0 : 0;
    E({ phase: 'download', revision: rev, total, message: '正在下载 Chromium 内核…', url: 'https://cdn.npmmirror.com/binaries/chromium-browser-snapshots/Win_x64/1710741/chrome-win.zip' });
    let sp = 9.6 * 1048576 / slow;
    const iv = setInterval(() => {
      if (K.cancel) { clearInterval(iv); return E({ phase: 'cancelled', running: false, speed: 0, eta: -1, message: '已取消安装', hint: '已下载的部分会保留，重新安装时从断点继续' }); }
      sp = sp * 0.75 + (7 + Math.random() * 6) * 1048576 / slow * 0.25;
      done = Math.min(total, done + sp * 0.25 * 4);
      if (done / total >= failAt) { clearInterval(iv); return stop('网络连接失败或被中断', source === 'mirror' ? '可能无法访问该下载源，请检查网络或代理设置，也可以切换到「自定义地址」使用其他镜像；点击「重试」会从断点继续' : '可能无法访问该下载源，请切换到「国内镜像」或填写自定义下载地址；点击「重试」会从断点继续', '网络错误: read tcp 192.168.1.8:52114->101.37.183.5:443: wsarecv: An existing connection was forcibly closed by the remote host.'); }
      E({ done, pct: done * 100 / total, speed: sp, eta: Math.round((total - done) / sp) });
      if (done >= total) {
        clearInterval(iv);
        let x = 0;
        E({ phase: 'extract', done: 0, total: unpacked, pct: 0, speed: 0, eta: -1, message: '正在解压…' });
        const iv2 = setInterval(() => {
          x = Math.min(unpacked, x + unpacked / 12);
          E({ done: x, pct: x * 100 / unpacked });
          if (x >= unpacked) {
            clearInterval(iv2);
            E({ phase: 'verify', done: 0, total: 0, pct: 0, message: '正在校验内核…' });
            setTimeout(() => {
              const dir = KDIR + 'chromium-' + ver, path = dir + '\\chrome.exe';
              K.list = [{ name: 'chromium-' + ver, version: ver, path, dir, size: 861_204_480 + 2_113_536, installed: Date.now(), active: true }, ...K.list.filter(k => k.dir !== dir)];
              mock.settings.chromiumPath = path;
              E({ phase: 'done', running: false, version: ver, path, pct: 100, speed: 0, eta: 0, message: '安装完成，已设为默认开源内核', hint: '' });
            }, 700);
          }
        }, 160);
      }
    }, 250);
  }, 900);
}

// ---- rpa recorder mock: a scripted user session streamed as rpa:record ----
const recSim = {
  st: null as any,
  timer: 0 as any,
  i: 0,
  pub: (): any => (recSim.st ? JSON.parse(JSON.stringify(recSim.st)) : { active: false, paused: false, steps: [], vars: {}, profileId: '', profileName: '', startedAt: 0, pausedMs: 0, pausedAt: 0, tabs: 0, options: {} }),
  emit: (): any => { const p = recSim.pub(); mockEmit('rpa:record', p); return p; },
};
const rs = (type: string, params: Record<string, string>, name = ''): any => ({ id: rid(), type, name, params });
// Each op either appends a step or keeps typing into the last fill.
const recScript = (): Array<() => void> => {
  const S = recSim.st;
  const push = (s: any) => { S.steps.push(s); };
  const typeInto = (sel: string, full: string, pw = false) => {
    const ops: Array<() => void> = [];
    for (let n = 1; n <= full.length; n += Math.max(1, Math.ceil(full.length / 4))) {
      const v = full.slice(0, n);
      ops.push(() => { const l = S.steps[S.steps.length - 1]; if (l?.type === 'fill' && l.params.selector === sel) l.params.text = pw ? '{{password}}' : v; else push(rs('fill', { selector: sel, text: pw ? '{{password}}' : v, ...(pw ? { _rec: 'password' } : {}) })); });
    }
    ops.push(() => { const l = S.steps[S.steps.length - 1]; l.params.text = pw ? '{{password}}' : full; if (pw) S.vars.password = ''; });
    return ops;
  };
  return [
    () => push(rs('goto', { url: 'https://seller.example-shop.com/login' })),
    ...typeInto('#username', 'seller_0921'),
    ...typeInto('input[name="password"]', 'hunter22', true),
    () => push(rs('click', { selector: 'text:记住我 || xpath://form/label[2]' })),
    () => push(rs('click', { selector: 'text:登录 || xpath://form/button' })),
    () => push(rs('delay', { ms: '2300', _rec: 'think' }, '停顿')),
    () => push(rs('wait_for', { selector: '.dashboard-card', state: 'visible', timeout: '10000' })),
    () => push(rs('click', { selector: 'text:订单管理' })),
    ...typeInto('#order-search', '2026101488'),
    () => push(rs('execute_js', { code: 'el.form.requestSubmit()', _rec: 'enter' }, '按回车键')),
    () => push(rs('fill', { selector: '#status-filter', text: 'shipped' })),
    () => push(rs('extract_text', { selector: '.order-row:nth-of-type(1) .amount', var: 'text1' })),
    () => push(rs('click', { selector: '[data-testid="export-csv"]' })),
    () => push(rs('select_tab', { index: '2' })),
    () => push(rs('click', { selector: 'button[aria-label="下载报表"]' })),
    () => push(rs('upload', { selector: '#invoice-file', files: '{{upload_file}}' }, '上传文件（录制时：invoice-1014.pdf）')),
    () => { S.vars.upload_file = ''; push(rs('close_tab', { index: '2' })); },
    () => push(rs('click', { selector: '#refresh-list', double: 'true' })),
  ];
};
const recSimStart = async (profileId: string, opts: any) => {
  await wait(700);
  if (recSim.st?.active) throw '已经在录制另一个环境，请先停止';
  const p = mock.profiles.find(x => x.id === profileId); if (!p) throw '环境不存在或已删除';
  if (!mock.running.some(r => r.profileId === profileId)) mock.running.push({ profileId, pid: 4242, debugPort: 9333, wsUrl: 'ws://127.0.0.1:9333/devtools/browser/x', started: Date.now() / 1000 });
  recSim.st = { active: true, paused: false, profileId, profileName: p.name, startedAt: Date.now(), pausedMs: 0, pausedAt: 0, steps: [], vars: {}, tabs: 1, options: opts || {} };
  const ops = recScript(); recSim.i = 0;
  const auto = !location.hash.includes('recmanual');
  clearInterval(recSim.timer);
  if (auto) recSim.timer = setInterval(() => {
    const S = recSim.st; if (!S?.active || S.paused) return;
    if (recSim.i >= ops.length) { clearInterval(recSim.timer); return; }
    ops[recSim.i++](); recSim.emit();
  }, 900);
  (window as any).__recStep = (n = 1) => { for (let k = 0; k < n && recSim.i < ops.length; k++) ops[recSim.i++](); return recSim.emit(); };
  return recSim.pub();
};
const recSimPause = async (p: boolean) => {
  const S = recSim.st; if (!S?.active) return recSim.pub();
  if (p && !S.paused) { S.paused = true; S.pausedAt = Date.now(); }
  if (!p && S.paused) { S.pausedMs += Date.now() - S.pausedAt; S.paused = false; S.pausedAt = 0; }
  return recSim.emit();
};
const recSimStop = async (by: string) => {
  await wait(300);
  const S = recSim.st; if (!S) return recSim.pub();
  clearInterval(recSim.timer);
  if (S.paused) await recSimPause(false);
  S.active = false; S.stoppedBy = by;
  return recSim.emit();
};
const recFinalize = (steps: any[], o: any) => {
  let out = JSON.parse(JSON.stringify(steps || [])).filter((s: any) => !(s.type === 'delay' && s.params._rec === 'think' && !o.keepDelays));
  out.forEach((s: any) => { if (s.params._rec === 'password' && !o.passwordVar) s.params.text = 'hunter22'; });
  if (o.mergeFills) {
    const bar = new Set(['click', 'goto', 'execute_js', 'select_tab', 'close_tab', 'upload', 'condition', 'loop']);
    out = out.filter((s: any, i: number) => {
      if (s.type !== 'fill') return true;
      for (let j = i + 1; j < out.length; j++) { if (bar.has(out[j].type)) break; if (out[j].type === 'fill' && out[j].params.selector === s.params.selector) return false; }
      return true;
    });
  }
  const vars: Record<string, string> = {};
  out.forEach((s: any) => { delete s.params._rec; Object.values(s.params).forEach((v: any) => { for (const m of String(v).matchAll(/\{\{\s*(password|upload_file\d*)\s*\}\}/g)) vars[m[1]] = ''; }); });
  return { steps: out, vars };
};
