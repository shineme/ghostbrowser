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
const wait = (ms = 250) => new Promise(r => setTimeout(r, ms));

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
  ChromiumPath: async () => mock.settings.chromiumPath || 'C:\\Program Files\\Chromium\\Application\\chrome.exe',
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
  StartProfile: async (id: string) => { await wait(600); const r = { profileId: id, pid: 1234, debugPort: 9222, wsUrl: 'ws://127.0.0.1:9222/devtools/browser/x', startedAt: Date.now() / 1000 }; mock.running.push(r); return r; },
  StopProfile: async (id: string) => { mock.running = mock.running.filter(r => r.profileId !== id); },
  Running: async () => [...mock.running],
  ExportCookies: async (_id: string) => '[]',
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
};

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

if (!inWails) (window as any).__mock = { api: M, emit: mockEmit };
