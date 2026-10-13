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
const mock = {
  profiles: [] as any[],
  proxies: [
    { id: 'px1', name: 'US Residential 01', type: 'socks5', host: '45.12.30.8', port: 1080, user: 'u', pass: 'p', changeIpUrl: '', ip: '45.12.30.8', country: 'US', city: 'Ashburn', timezone: 'America/New_York', lat: 39.04, lon: -77.49, latency: 182, status: 'ok', checkedAt: Date.now() / 1000 },
    { id: 'px2', name: 'DE Datacenter', type: 'http', host: '88.198.4.21', port: 3128, user: '', pass: '', changeIpUrl: '', ip: '88.198.4.21', country: 'DE', city: 'Falkenstein', timezone: 'Europe/Berlin', lat: 50.47, lon: 12.37, latency: 96, status: 'ok', checkedAt: Date.now() / 1000 },
    { id: 'px3', name: 'JP Mobile', type: 'http', host: '133.18.2.40', port: 8000, user: 'a', pass: 'b', changeIpUrl: 'https://example/rotate', ip: '', country: '', city: '', timezone: '', lat: 0, lon: 0, latency: 0, status: 'fail', checkedAt: 0 },
  ] as any[],
  running: [] as any[],
  settings: { kernelPath: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe', kernelMode: 'stock', profilesDir: 'C:\\Users\\me\\AppData\\Roaming\\GhostBrowser\\profiles', apiPort: 25325, apiEnabled: true, defaultOS: 'windows', syncServer: '', syncToken: '', syncPassword: '', deviceName: 'DESKTOP-01', deviceId: 'abc', theme: 'dark' } as any,
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
  GetSettings: async () => ({ ...mock.settings }),
  SaveSettings: async (s: any) => (mock.settings = s),
  DefaultKernelPath: async () => mock.settings.undetectableKit ? `${mock.settings.undetectableKit}\\runtime\\chrome\\chrome.exe` : mock.settings.kernelPath,
  PickKernel: async () => mock.settings.kernelPath,
  PickFolder: async () => mock.settings.profilesDir,
  KernelMajor: async () => '141',
  TestSync: async () => { await wait(700); if (!mock.settings.syncServer) throw '请先填写同步服务器地址'; },
  PushProfile: async (id: string) => { await wait(900); const p = mock.profiles.find(x => x.id === id); p.storage = 'cloud'; p.cloudVersion++; p.syncedAt = Date.now() / 1000; },
  PullProfile: async (_id: string) => { await wait(900); },
  ListCloud: async () => mock.profiles.filter(p => p.storage === 'cloud').map(p => ({ id: p.id, name: p.name, version: p.cloudVersion, sha256: 'e3b0c442…', meta: null, lockedBy: '' })),
  DataDir: async () => 'C:\\Users\\me\\AppData\\Roaming\\GhostBrowser',
};

export const api: typeof M = inWails ? (Go as any) : M;

export const onEvent = (name: string, cb: (...a: any[]) => void) => {
  const rt = (window as any).runtime;
  if (rt && rt.EventsOn) return rt.EventsOn(name, cb);
  return () => {};
};
