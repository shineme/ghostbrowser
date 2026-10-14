import React, { useEffect, useState } from 'react';
import { Combo, SearchSelect, FontPicker, CountrySelect, LanguagePicker, TimezoneSelect, countryOfTz } from './locales';
import { api, Profile, Proxy, Fingerprint } from './api';
import { Field, Toggle, Icon, flag, useToast, useConfirm, useExit, Segmented, Avatar, DevIcon, SelectField } from './ui';
import { CookieModal } from './CookieModal';
import { ExtIcon, Ext } from './ExtensionsView';
import { GuardPane, LaunchPane, seedShort } from './FpControls';
import { KernelMissingNotice } from './KernelInstall';
import { FoxIcon, FirefoxMissingNotice, FirefoxFpPane, FirefoxFontPane, FirefoxGuardPane, FirefoxLaunchPane, useFirefoxInfo, ffVersionLabel } from './FirefoxKernel';
import { DEVICE_TYPES, TRUST, devType, osName, osVerLabel, needsOpen, uaOSOf } from './devices';

type FPCatalog = { osVersions: string[]; browserMajors: string[]; userAgents: string[]; platforms: string[]; vendors: string[]; gpus: { vendor: string; renderer: string; custom?: string }[]; fonts: string[]; screens: string[]; cores: string[]; memory: string[]; custom: Record<string, string[]> };
const EMPTY_CAT: FPCatalog = { osVersions: [], browserMajors: [], userAgents: [], platforms: [], vendors: [], gpus: [], fonts: [], screens: [], cores: [], memory: [], custom: {} };
// Keep the current value visible even when it is outside the catalog (e.g. an imported profile).
const opt = <T,>(list: T[], cur: T): T[] => (cur === '' || cur === undefined || cur === null || list.includes(cur)) ? list : [cur, ...list];


export const ProfileDrawer: React.FC<{
  profile: Profile | null; proxies: Proxy[]; onClose: () => void; onSaved: () => void;
}> = ({ profile, proxies, onClose, onSaved }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [pxText, setPxText] = useState('');
  const [pxBusy, setPxBusy] = useState(false);
  const [pxInfo, setPxInfo] = useState('');
  const [extraProxies, setExtraProxies] = useState<Proxy[]>([]);
  const quickProxy = async () => {
    setPxBusy(true); setPxInfo('');
    try {
      const pv: any[] = await api.PreviewProxies(pxText, 'auto') as any;
      const hit = pv.find(r => r.ok);
      if (!hit) { setPxInfo(`无法识别：${pv[0]?.error || '格式不对'}`); return; }
      const same = proxies.find(x => x.host === hit.proxy.host && x.port === hit.proxy.port && (x.user || '') === (hit.proxy.user || ''));
      const saved = same || (await api.ImportProxyList([hit.proxy], ''))[0];
      const r = await api.CheckProxy(saved.id);
      setExtraProxies(x => [r, ...x]); setP(cur => ({ ...cur, proxyId: r.id })); setPxText('');
      setPxInfo(r.status === 'ok' ? `✓ ${r.type.toUpperCase()} · ${r.country} ${r.region || ''} ${r.city || ''} · ${r.ipType === 'hosting' ? '机房' : r.ipType === 'mobile' ? '移动' : '住宅'} IP · ${r.latency} ms${same ? '（已在代理列表中）' : '，已加入代理列表'}` : `✗ 不可用：${r.error || '连接失败'}（已加入代理列表，可稍后重试）`);
    } catch (e: any) { setPxInfo(String(e?.message || e)); } finally { setPxBusy(false); }
  };
  const [tab, setTab] = useState<'basic' | 'fp' | 'adv' | 'guard' | 'launch'>('basic');
  // ---- extensions / cookies ----
  const [exts, setExts] = useState<Ext[] | null>(null);
  const [cookieOpen, setCookieOpen] = useState(false);
  useEffect(() => { api.ListExtensions().then(l => setExts((l || []) as Ext[])).catch(() => setExts([])); }, []);
  const [p, setP] = useState<Profile>(() => profile ? JSON.parse(JSON.stringify(profile)) : blank());
  const [gpus, setGpus] = useState<{ vendor: string; renderer: string }[]>([]);
  const [devices, setDevices] = useState<any[]>([]);
  const [device, setDevice] = useState('');
  const [cat, setCat] = useState<FPCatalog>(EMPTY_CAT);
  const [screens, setScreens] = useState<string[]>([]);
  const fp = p.fingerprint;
  const [out, close] = useExit(onClose);
  const [flash, setFlash] = useState(0);
  // Firefox strict mode wins over "language / time zone follows IP" (same rule as normalizeFirefoxFP).
  const setFP = (patch: Partial<Fingerprint>) => setP(s => {
    const f: any = { ...s.fingerprint, ...patch };
    if ((s as any).kernel === 'firefox' && f.ffStrict) { f.languageFromIP = false; f.timezoneFromIP = false; }
    return { ...s, fingerprint: f as Fingerprint };
  });
  // ---- fingerprint extras: new noise seed (saved at once for existing profiles; cookies untouched) ----
  const regenSeed = async () => {
    const ok = await ask({
      title: '重新生成指纹种子？', ok: '重新生成',
      body: <>Canvas、WebGL、音频、ClientRects 噪声和媒体设备 ID 会全部换新，网站会把它当成另一台设备。<br />Cookie、登录状态、代理和其他指纹参数<b>保持不变</b>。{profile?.id ? '新种子立即保存，运行中的环境需重启后生效。' : ''}</>,
    });
    if (!ok) return;
    try {
      const seed = profile?.id ? await api.RegenerateSeed(profile.id) : await api.NewNoiseSeed();
      setFP({ noiseSeed: seed } as any);
      toast(`已生成新种子 ${seedShort(seed)}`);
    } catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  // Exit-IP match: what timezone / languages / coordinates the selected proxy implies.
  const [geo, setGeo] = useState<any>(null);
  const [geoBusy, setGeoBusy] = useState(false);
  const followAny = fp.timezoneFromIP || fp.languageFromIP || fp.geoFromIP;
  const runGeo = async (force = false) => {
    setGeoBusy(true);
    try {
      const m: any = await api.MatchGeo(p.proxyId || '', (p.inlineProxy || null) as any);
      setGeo(m);
      if (m.ok) setP(s => {
        const f = s.fingerprint, patch: Partial<Fingerprint> = {};
        if (f.timezoneFromIP || force) patch.timezone = m.timezone;
        if (f.languageFromIP || force) patch.languages = m.languages;
        if ((f.geoFromIP || force) && (m.lat || m.lon)) { patch.latitude = m.lat; patch.longitude = m.lon; }
        if (force) Object.assign(patch, { timezoneFromIP: true, languageFromIP: true, geoFromIP: true });
        return { ...s, fingerprint: { ...f, ...patch } as Fingerprint };
      });
    } catch (e: any) { setGeo({ ok: false, error: String(e?.message || e) }); } finally { setGeoBusy(false); }
  };
  useEffect(() => { setGeo(null); if (followAny || tab === 'adv') runGeo(); /* eslint-disable-next-line */ }, [p.proxyId, p.inlineProxy?.host]);
  useEffect(() => { if (tab === 'adv' && !geo && !geoBusy) runGeo(); /* eslint-disable-next-line */ }, [tab]);
  const ipTypeName = (t: string) => t === 'hosting' ? '机房' : t === 'mobile' ? '移动' : t === 'residential' ? '住宅' : '';
  const geoCard = (
    <div className={`geo-card ${geo && !geo.ok ? 'bad' : geo?.stale ? 'warn' : ''}`}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Icon.globe s={15} />
        <b>{p.proxyId || p.inlineProxy ? '代理出口' : '本机出口（未选代理）'}</b>
        <div className="spacer" style={{ flex: 1 }} />
        <button type="button" className="btn sm" disabled={geoBusy} onClick={() => runGeo()}>{geoBusy ? '检测中…' : '重新检测'}</button>
        {geo?.ok && !(fp.timezoneFromIP && fp.languageFromIP && fp.geoFromIP) && !((p as any).kernel === 'firefox' && (fp as any).ffStrict) && <button type="button" className="btn sm primary" onClick={() => runGeo(true)}>全部跟随 IP</button>}
      </div>
      {!geo && <div className="sub">{geoBusy ? '正在检测出口 IP…' : '尚未检测'}</div>}
      {geo && !geo.ok && <div className="sub">检测失败：{geo.error}。启动时会再试一次；仍失败会拒绝启动，避免时区和 IP 对不上。</div>}
      {geo?.ok && <>
        <div className="num" style={{ marginTop: 4 }}>{geo.ip} · {geo.country} {[geo.region, geo.city].filter(Boolean).join(' · ')}{geo.ipType ? ` · ${ipTypeName(geo.ipType)}` : ''}{geo.latency ? ` · ${geo.latency} ms` : ''}</div>
        <div className="sub">应匹配：时区 {geo.timezone} · 语言 {(geo.languages || []).join(', ')} · 坐标 {(+geo.lat).toFixed(4)}, {(+geo.lon).toFixed(4)}</div>
        {geo.stale && <div className="sub">现在连不上代理，以上是上次检测的结果（{geo.error}）。</div>}
      </>}
    </div>
  );

  const dt = devType(fp.deviceType);
  const dtInfo = DEVICE_TYPES.find(d => d.id === dt)!;
  const firefox = (p as any).kernel === 'firefox';
  const [ffInfo] = useFirefoxInfo(firefox);
  const forcedOpen = !firefox && needsOpen(fp.os, dt);
  const openKernel = !firefox && (forcedOpen || (p as any).kernel === 'chromium');
  const desk = dt === 'desktop' || dt === 'laptop';
  const loadPresets = () => api.DevicePresetsByType(dt, fp.os).then(d => setDevices(d || [])).catch(() => setDevices([]));
  useEffect(() => {
    api.GPUOptions(fp.os).then(setGpus);
    loadCat();
    loadPresets();
    api.ScreenOptions(fp.os).then(setScreens);
  }, [fp.os, dt]);

  const native = !!(p as any).native && !openKernel && !firefox;
  // Firefox: desktop only; strict mode and "language follows IP" exclude each other.
  const setKernel = (k: string) => setP(s => ({ ...s, kernel: k, ...(k === 'firefox' ? { native: undefined } : {}), ...(k === 'firefox' && (s.fingerprint as any).ffStrict ? { fingerprint: { ...s.fingerprint, languageFromIP: false, timezoneFromIP: false } } : {}) } as any));
  const changeType = async (t: string) => {
    const info = DEVICE_TYPES.find(d => d.id === t)!;
    const os = info.oses.includes(fp.os) ? fp.os : info.oses[0];
    const f: any = await api.NewDeviceFingerprint(t, os);
    setDevice('');
    setP(s => ({ ...s, fingerprint: { ...f, deviceType: t }, kernel: needsOpen(os, t) ? 'chromium' : (s as any).kernel } as any));
    setFlash(x => x + 1);
  };
  const applyDevice = (id: string) => {
    const d = devices.find(x => x.id === id); if (!d) return;
    setDevice(id);
    setFP({ ...(d.userAgent ? { userAgent: d.userAgent.replace('%s', fp.browserVersion) } : {}), ...(d.platform ? { platform: d.platform } : {}), deviceModel: d.model || '', deviceName: d.deviceName || '', osVersion: d.osVersion, webglVendor: d.webglVendor, webglRenderer: d.webglRenderer, screenWidth: d.screenWidth, screenHeight: d.screenHeight, pixelRatio: d.pixelRatio, hardwareConcurrency: d.hardwareConcurrency, deviceMemory: d.deviceMemory, maxTouchPoints: d.maxTouchPoints });
    setFlash(x => x + 1);
  };
  const loadCat = () => api.FingerprintCatalogFor(fp.os, dt).then(c => setCat({ ...EMPTY_CAT, ...(c as any) })).catch(() => setCat(EMPTY_CAT));
  const libAdd = async (field: string, v: string) => { try { await api.LibraryAdd(fp.os, field, v); toast('已存入我的指纹库'); loadCat(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const libRemove = async (field: string, v: string) => { try { await api.LibraryRemove(fp.os, field, v); toast('已从指纹库删除'); loadCat(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const lib = (field: string, options: string[]) => ({ options, custom: cat.custom?.[field] || [], onSave: (v: string) => libAdd(field, v), onRemove: (v: string) => libRemove(field, v) });
  const vendorGpus = cat.gpus.filter(g => g.vendor === fp.webglVendor);
  const [scrText, setScrText] = useState('');
  useEffect(() => { setScrText(`${fp.screenWidth}x${fp.screenHeight}@${fp.pixelRatio}`); }, [fp.screenWidth, fp.screenHeight, fp.pixelRatio]);
  // A typed UA that belongs to another OS is allowed, but flagged.
  const uaOS = uaOSOf(fp.userAgent);
  const mobileUA = /Mobile Safari/.test(fp.userAgent);
  const uaWarn = uaOS && uaOS !== fp.os ? `这个 UA 属于 ${osName(uaOS)}，与所选系统不一致，容易被识别`
    : fp.os === 'android' && mobileUA !== (dt === 'phone' || dt === 'foldable') ? (mobileUA ? '平板/电视的 UA 不应带 Mobile' : '手机的 UA 应带 Mobile') : '';
  const savePreset = async () => {
    try {
      const d: any = await api.SaveDevicePreset({ id: device.startsWith('my-') ? device : '', name: '', os: fp.os, deviceType: dt, model: fp.deviceModel || '', deviceName: fp.deviceName || '', osVersion: fp.osVersion, webglVendor: fp.webglVendor, webglRenderer: fp.webglRenderer, screenWidth: fp.screenWidth, screenHeight: fp.screenHeight, pixelRatio: fp.pixelRatio, hardwareConcurrency: fp.hardwareConcurrency, deviceMemory: fp.deviceMemory, maxTouchPoints: fp.maxTouchPoints } as any);
      setDevice(d.id); toast(`已保存设备预设：${d.name}`);
      loadPresets();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const delPreset = async () => { try { await api.DeleteDevicePreset(device); setDevice(''); toast('已删除设备预设'); loadPresets(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  // Linked edits: UA and browser major always move together; renderer always belongs to the vendor.
  const setMajor = (m: string) => setFP({ browserVersion: m, userAgent: cat.userAgents.find(u => u.includes(`Chrome/${m}.`)) || fp.userAgent });
  const setUA = (ua: string) => { const m = /Chrome\/(\d+)\./.exec(ua); setFP({ userAgent: ua, ...(m ? { browserVersion: m[1] } : {}) }); };
  const setVendor = (v: string) => { const g = cat.gpus.find(x => x.vendor === v); setFP({ webglVendor: v, webglRenderer: g ? g.renderer : fp.webglRenderer }); };
  const regen = async () => { const f: any = native ? await api.NewFingerprint('windows') : await api.NewDeviceFingerprint(dt, fp.os); setDevice(''); setFP({ ...f, deviceType: dt }); setFlash(x => x + 1); toast('已生成新指纹'); };
  const changeOS = async (os: string) => { const f: any = await api.NewDeviceFingerprint(dt, os); setDevice(''); setP(s => ({ ...s, fingerprint: { ...f, deviceType: dt }, kernel: needsOpen(os, dt) ? 'chromium' : (s as any).kernel } as any)); setFlash(x => x + 1); };
  const touchOpts = dt === 'phone' || dt === 'tablet' || dt === 'foldable' ? [5, 10] : (dt === 'laptop' && fp.os === 'windows') || dt === 'chromebook' ? [0, 10] : [0];

  const save = async () => {
    if (!p.name.trim()) { toast('请填写环境名称', 'bad'); setTab('basic'); return; }
    try { await api.SaveProfile(p); } catch (e) { toast(String(e), 'bad'); return; }
    toast('环境已保存'); onSaved(); close();
  };

  const scr = `${fp.screenWidth}x${fp.screenHeight}@${fp.pixelRatio}`;
  const setScr = (v: string) => { const m = /(\d+)x(\d+)@([\d.]+)/.exec(v); if (m) setFP({ screenWidth: +m[1], screenHeight: +m[2], pixelRatio: +m[3] }); };

  return (
    <div className={`overlay ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="drawer">
        <div className="dhead">
          <div className="glyph"><Icon.fingerprint s={20} /></div>
          <div>
            <h3>{profile ? '编辑环境' : '新建环境'}</h3>
            <div className="sub">{profile ? p.name : '一个环境 = 一套指纹 + 代理 + 独立数据目录'}</div>
          </div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div>
        </div>

        <div className="dbody">
          <div key={flash} className={`fpcard ${flash ? 'flash' : ''}`}>
            <Avatar seed={fp.noiseSeed} os={fp.os} />
            <div style={{ minWidth: 0 }}>
              <div className="ua">{firefox ? ((fp as any).ffUaOverride && fp.userAgent ? fp.userAgent : `${ffVersionLabel(ffInfo)} · 使用 Firefox 真实 User-Agent`) : (fp.userAgent || '保存时按所选系统自动生成 User-Agent')}</div>
              <div className="chips">
                <span className="devchip"><DevIcon t={dt} s={12} />{fp.deviceName || dtInfo.name}</span>{!firefox && <span>{fp.platform}</span>}<span className="num">{fp.screenWidth}×{fp.screenHeight} @{fp.pixelRatio}x</span>
                <span className="num">{fp.hardwareConcurrency} 核 · {fp.deviceMemory} GB</span>
                {firefox ? <span className="ffchip"><FoxIcon s={11} /> Firefox · WebGL {(fp as any).ffWebglOff ? '禁用' : '真实'}{(fp as any).ffStrict ? ' · 严格模式' : ''}</span> : <span>{(fp.webglRenderer || 'WebGL 自动').replace(' (TM)', '').replace(/^ANGLE \(([^,]+),\s*/, '').split(' (')[0].split(',')[0]}</span>}
              </div>
            </div>
            <i className="scan" />
          </div>
          <Segmented value={tab} onChange={setTab} items={[['basic', '基本信息', <Icon.layers />], ['fp', '指纹配置', <Icon.fingerprint />], ['adv', '地区与字体', <Icon.globe />], ['guard', '防护', <Icon.shield />], ['launch', '启动', <Icon.bolt />]]} />

          {tab === 'basic' && <div className="tabpane">
            <Field label="环境名称"><input type="text" value={p.name} onChange={e => setP({ ...p, name: e.target.value })} placeholder="例如 FB-Ads-01" autoFocus /></Field>
            <div className="grid2">
              <Field label="分组"><input type="text" value={p.group} onChange={e => setP({ ...p, group: e.target.value })} placeholder="广告投放 / 电商…" /></Field>
              <Field label="标签 (逗号分隔)"><input type="text" value={(p.tags || []).join(', ')} onChange={e => setP({ ...p, tags: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })} /></Field>
            </div>
            <Field label="存储方式">
              <SelectField value={p.storage || 'local'} onChange={v => setP({ ...p, storage: v })} options={[['local', '本地 — 只存在这台电脑'], ['cloud', '云端 — 同步到你的 VPS']]} />
            </Field>
            <Field label="代理">
              <SelectField value={p.proxyId || ''} onChange={v => setP({ ...p, proxyId: v })} options={[['', '不使用代理（直连）'] as [string, string], ...[...extraProxies.filter(x => !proxies.some(y => y.id === x.id)), ...proxies].map(px => [px.id, `${flag(px.country)} · ${px.name} · ${px.type}${px.status === 'ok' ? ` · ${px.city || px.ip}` : ''}`] as [string, string])]} />
              <div className="combo-row" style={{ marginTop: 6 }}>
                <input type="text" value={pxText} onChange={e => setPxText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') quickProxy(); }} placeholder="或直接粘贴一个代理，自动识别格式并检测，如 user:pass@1.2.3.4:8080" />
                <button type="button" className="btn sm combo-btn" disabled={pxBusy || !pxText.trim()} onClick={quickProxy}>{pxBusy ? '检测中…' : '识别并使用'}</button>
              </div>
              {pxInfo && <div className="sub" style={{ marginTop: 4 }}>{pxInfo}</div>}
            </Field>
            <Field label="启动页 (每行一个)"><textarea value={(p.startUrls || []).join('\n')} onChange={e => setP({ ...p, startUrls: e.target.value.split('\n') })} placeholder="https://facebook.com" /></Field>
            {profile
              ? <Field label="Cookies"><div className="ck-row">
                  <span className="ck-row-ic"><Icon.lock s={15} /></span>
                  <span className="ck-row-t"><b>导入 / 导出 Cookie</b><span className="sub">JSON（Cookie-Editor / EditThisCookie）或 Netscape cookies.txt{stagedN(p.cookies) ? ` · 待写入 ${stagedN(p.cookies)} 条` : ''}</span></span>
                  <button type="button" className="btn sm" onClick={() => setCookieOpen(true)}><Icon.upload s={13} /> 管理</button>
                </div></Field>
              : <Field label="导入 Cookies (JSON, 下次启动时写入)" hint="支持 Cookie-Editor / EditThisCookie 的导出格式"><textarea value={p.cookies} onChange={e => setP({ ...p, cookies: e.target.value })} placeholder="[ { &quot;name&quot;: …, &quot;domain&quot;: … } ]" /></Field>}
            {firefox ? <Field label="扩展"><div className="sub">Firefox 内核暂不支持扩展中心的 Chromium 扩展。</div></Field> : <Field label={`扩展${exts && exts.length ? `（已启用 ${(p.extensionIds || []).filter(id => exts.some(e => e.id === id)).length} / ${exts.length}）` : ''}`} hint={exts && exts.length ? '保存环境后，下次启动时加载' : undefined}>
              {exts === null ? <div className="sub">正在读取扩展…</div>
                : exts.length === 0 ? <div className="sub">还没有安装扩展，可以在左侧「扩展中心」添加。</div>
                : <div className="dr-exts">{exts.map(e => { const on = (p.extensionIds || []).includes(e.id); return <div key={e.id} className={`dr-ext ${on ? 'on' : ''}`} onClick={() => setP(s => ({ ...s, extensionIds: on ? (s.extensionIds || []).filter(x => x !== e.id) : [...(s.extensionIds || []), e.id] }))}>
                    <ExtIcon e={e} size={26} /><span className="dr-ext-t"><b className="ellip">{e.name}</b><span className="sub num">v{e.version} · MV{e.manifestVersion}</span></span><div className={`sw ${on ? 'on' : ''}`} />
                  </div>; })}</div>}
            </Field>}
            <Field label="备注"><textarea value={p.notes} onChange={e => setP({ ...p, notes: e.target.value })} /></Field>
          </div>}

          {tab === 'fp' && <div className="tabpane">
            <div className="section-title" style={{ marginTop: 0 }}>设备类型</div>
            <div className="devgrid">
              {DEVICE_TYPES.map(d => { const off = firefox && d.id !== 'desktop' && d.id !== 'laptop'; return <button type="button" key={d.id} className={`devtile ${d.id === dt ? 'on' : ''} ${off ? 'off' : ''}`} disabled={off} onClick={() => !off && d.id !== dt && changeType(d.id)} title={off ? 'Firefox 内核只支持桌面设备' : d.tip}>
                <DevIcon t={d.id} s={20} /><b>{d.name}</b><i className={`trust ${d.trust}`} />
              </button>; })}
            </div>
            <div className="sub" style={{ margin: '6px 0 12px' }}><span className={`badge ${TRUST[dtInfo.trust][1]}`}><span className="d" />{TRUST[dtInfo.trust][0]}</span> {dtInfo.tip}</div>
            <div className="kernel-row">
              <span className="sub" style={{ margin: 0 }}>内核</span>
              {forcedOpen && !desk
                ? <span className="kchip"><Icon.cpu s={13} /> 开源内核（Chromium）<small>{`${dtInfo.name}只能用开源内核`}</small></span>
                : <Segmented className="compact kseg" value={firefox ? 'firefox' : openKernel ? 'chromium' : ''} onChange={setKernel} items={[...(forcedOpen ? [] : [['', '定制内核'] as [string, string]]), ['chromium', forcedOpen ? `开源内核（${osName(fp.os)} 必需）` : '开源内核'], ['firefox', 'Firefox', <FoxIcon s={13} />]]} />}
              {firefox && <span className={`ff-vchip ${ffInfo && !ffInfo.path ? 'miss' : ''}`} title={ffInfo?.path || ''}>{ffVersionLabel(ffInfo)}</span>}
            </div>
            {firefox && <div className="sub" style={{ margin: '-4px 0 12px' }}>用本机的 Mozilla Firefox 启动，独立配置目录；指纹通过 Firefox 偏好设置（user.js）和时区环境变量实现，通过 WebDriver BiDi 控制。只支持桌面设备。</div>}
            {firefox && <FirefoxMissingNotice />}
            {firefox && <FirefoxFpPane fp={fp as any} setFP={setFP as any} info={ffInfo} />}
            {!firefox && <>
            {openKernel && <div className="sub" style={{ margin: '-4px 0 12px' }}>{dt === 'desktop' || dt === 'laptop' || dt === 'chromebook' ? '用 CDP 设置 UA、UA-CH、时区和语言，页面脚本补上显卡、屏幕、噪声和字体。' : '用 Chrome 开发者工具同款的设备模拟：UA 与 UA-CH、视口、DPR、触控、屏幕方向，页面脚本补上显卡、噪声和字体。'}{dt === 'tv' ? ' 电视浏览器的内核版本较老，和当前内核的特性不一致。' : ''}</div>}
            {openKernel && <KernelMissingNotice />}
            {native && <div className="sub" style={{ marginBottom: 12 }}>原生模板 {p.native?.templateKey} · 配置 {p.native?.sourceConfigId}。以下项目会写入 startInfo，高级参数通过启动 JSON 管理。</div>}
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 10 }}>
              <button className="btn sm" onClick={regen}><Icon.dice s={14} /> {native ? '加载默认模板值' : '重新随机'}</button>
            </div>
            {!native && devices.length > 0 && <Field label={`设备预设（${devices.length} 台 ${osName(fp.os)} ${dtInfo.name}）`} hint="选择一台真实机型，显卡、分辨率、CPU、内存一次填好"><SearchSelect value={device} options={devices.map(d => [d.id, (d.custom ? '★ ' : '') + d.name])} onChange={applyDevice} placeholder={dt === 'phone' || dt === 'foldable' ? '搜索机型，如 S24 / Pixel / 小米' : dt === 'tablet' ? '搜索机型，如 Tab S9 / Pad' : '搜索机型，如 4060 / M3 / Iris'} /></Field>}
            <div className="grid3">
              <Field label="操作系统"><SelectField value={fp.os} onChange={v => changeOS(v)} options={dtInfo.oses.filter(v => !native || v === 'windows').map(v => [v, osName(v)] as [string, string])} /></Field>
              <Field label="系统版本"><Combo value={fp.osVersion} {...lib('osVersions', cat.osVersions)} onChange={v => setFP({ osVersion: v })} render={v => osVerLabel(fp.os, v)} /></Field>
              <Field label="浏览器大版本"><Combo value={fp.browserVersion} {...lib('browserMajors', cat.browserMajors)} onChange={setMajor} render={v => `Chrome ${v}`} /></Field>
            </div>
            <Field label="User-Agent" hint={`可手动输入或粘贴；列表只含 ${osName(fp.os)} ${dtInfo.name}的 UA，★ 为你存入库的。大版本会按 UA 自动同步`}>
              <Combo multiline value={fp.userAgent} {...lib('userAgents', cat.userAgents)} onChange={setUA} placeholder="粘贴 User-Agent，或点 ▾ 选择" />
              {uaWarn && <div className="sub" style={{ color: 'var(--warn, #f5c451)', marginTop: 4 }}>⚠ {uaWarn}</div>}
            </Field>
            <div className="grid2">
              <Field label="平台 (navigator.platform)"><Combo value={fp.platform} {...lib('platforms', cat.platforms)} onChange={v => setFP({ platform: v })} /></Field>
              <Field label="分辨率" hint="格式 宽x高@缩放，如 1920x1080@1.25"><Combo value={scrText} {...lib('screens', cat.screens.length ? cat.screens : screens)} onChange={v => { setScrText(v); setScr(v); }} /></Field>
            </div>
            {fp.os === 'android' && <div className="grid2">
              <Field label="机型名称"><input type="text" value={fp.deviceName || ''} onChange={e => setFP({ deviceName: e.target.value })} placeholder="如 Samsung Galaxy S24 Ultra" /></Field>
              <Field label="型号 (UA-CH model)" hint="网站通过 Client Hints 读到的型号"><input type="text" value={fp.deviceModel || ''} onChange={e => setFP({ deviceModel: e.target.value })} placeholder="如 SM-S928B" /></Field>
            </div>}
            <div className="grid3">
              <Field label="CPU 核心数"><Combo value={String(fp.hardwareConcurrency ?? '')} {...lib('cores', cat.cores)} onChange={v => /^\d*$/.test(v) && setFP({ hardwareConcurrency: +v })} /></Field>
              <Field label="内存 (GB)" hint="Chrome 上报最高 8"><Combo value={String(fp.deviceMemory ?? '')} {...lib('memory', cat.memory)} onChange={v => /^\d*$/.test(v) && setFP({ deviceMemory: +v })} /></Field>
              <Field label={native ? '触控点（继承模板）' : '触控点'}><SelectField disabled={native} value={String(fp.maxTouchPoints ?? 0)} onChange={v => setFP({ maxTouchPoints: +v })} options={opt(touchOpts, fp.maxTouchPoints).map(n => [String(n), n === 0 ? '0（无触屏）' : `${n}（触屏）`] as [string, string])} /></Field>
            </div>
            <div className="grid2">
              <Field label="WebGL Vendor"><Combo value={fp.webglVendor} {...lib('vendors', cat.vendors)} onChange={setVendor} /></Field>
              <Field label="WebGL 显卡" hint="列表只含所选 Vendor 的显卡；★ 存入库会连同 Vendor 一起保存">
                <Combo multiline value={fp.webglRenderer} options={vendorGpus.map(g => g.renderer)} custom={vendorGpus.filter(g => g.custom).map(g => g.renderer)}
                  onChange={v => setFP({ webglRenderer: v })} onSave={r => libAdd('gpus', `${fp.webglVendor}\x1f${r}`)} onRemove={r => libRemove('gpus', `${fp.webglVendor}\x1f${r}`)} placeholder="如 ANGLE (NVIDIA, NVIDIA GeForce RTX 4060 ...)" />
              </Field>
            </div>
            {!native && <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 6 }}>
              <button className="btn sm" onClick={savePreset}><Icon.check s={14} /> 把当前机型存为我的设备预设</button>
              {device.startsWith('my-') && <button className="btn sm" onClick={delPreset}>删除这个我的预设</button>}
            </div>}
            </>}
          </div>}

          {tab === 'adv' && native && <div className="tabpane">
            {geoCard}
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v, timezoneFromIP: false })} /></Field>
            <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v, languageFromIP: false })} /></Field>
            <Field label="二进制模板"><SelectField value={p.native?.templateKey || ''} onChange={v => setP({ ...p, native: { ...p.native!, templateKey: v } as any })} options={[['windows-intel-a', 'Windows Intel · A'], ['windows-intel-a2', 'Windows Intel · A2'], ['windows-intel-b', 'Windows Intel · B']]} /></Field>
            <div className="sub">字体、媒体设备、语音、WebGL 参数和未映射的噪声开关继承模板。高级配置可在「指纹库 / JSON」导出启动 JSON，修改后导入为新环境。此模式直接使用内核和 CDP，需保持 GhostBrowser 运行。</div>
          </div>}

          {tab === 'adv' && !native && <div className="tabpane">
            <div className="section-title">地区 — 跟随代理出口 IP</div>
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            {geoCard}
            {firefox && (fp as any).ffStrict && <div className="notice warn ff-strict"><Icon.lock s={14} /><span className="grow">已开启 Firefox 严格模式：时区固定为 UTC+0、语言为 en-US，「跟随 IP」和下方手动设置不生效。可在「防护」里关闭严格模式。</span></div>}
            <div className={firefox && (fp as any).ffStrict ? 'ff-locked' : ''}>
            <Toggle on={fp.timezoneFromIP} onChange={v => setFP({ timezoneFromIP: v })} label="时区跟随 IP" hint={fp.timezoneFromIP && geo?.ok ? `→ ${geo.timezone}` : '关闭则使用下方手动时区'} />
            {!fp.timezoneFromIP && <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v })} /></Field>}
            <Toggle on={fp.languageFromIP} onChange={v => setFP({ languageFromIP: v })} label="语言跟随 IP" hint={fp.languageFromIP && geo?.ok ? `→ ${(geo.languages || []).join(', ')}` : '按出口国家（及地区）设置 Accept-Language 和 navigator.languages'} />
            {!fp.languageFromIP && <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v })} /></Field>}
            <Toggle on={fp.geoFromIP} onChange={v => setFP({ geoFromIP: v })} label="地理定位跟随 IP" hint={fp.geoFromIP && geo?.ok ? `→ ${(+geo.lat).toFixed(4)}, ${(+geo.lon).toFixed(4)}（${geo.city || geo.country}）` : '关闭则使用下方手动坐标'} />
            </div>
            {!fp.geoFromIP && <div className="grid2"><Field label="纬度"><input type="number" step="0.0001" value={fp.latitude} onChange={e => setFP({ latitude: +e.target.value })} /></Field><Field label="经度"><input type="number" step="0.0001" value={fp.longitude} onChange={e => setFP({ longitude: +e.target.value })} /></Field></div>}

            {firefox ? <FirefoxFontPane fp={fp as any} setFP={setFP as any} /> : <><div className="section-title">字体</div>
            <Field label={`字体（${(fp.fonts || []).length} / ${cat.fonts.length} 种 ${osName(fp.os)} 字体）`} hint="只列出该系统自带字体">
              <FontPicker value={fp.fonts || []} all={cat.fonts} onChange={v => setFP({ fonts: v })} onSave={f => libAdd('fonts', f)} />
            </Field></>}
          </div>}

          {tab === 'guard' && <div className="tabpane">{firefox ? <FirefoxGuardPane fp={fp as any} setFP={setFP as any} /> : <GuardPane fp={fp as any} setFP={setFP as any} open={openKernel} onRegenSeed={regenSeed} />}</div>}
          {tab === 'launch' && <div className="tabpane">{firefox ? <FirefoxLaunchPane fp={fp as any} setFP={setFP as any} /> : <LaunchPane fp={fp as any} setFP={setFP as any} open={openKernel} />}</div>}
        </div>

        <div className="dfoot">
          <div className="sub num" style={{ color: 'var(--txt-3)', fontSize: 11 }}>SEED {fp.noiseSeed}</div>
          <div className="spacer" />
          <button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" onClick={save}><Icon.check s={15} /> 保存环境</button>
        </div>
      </div>
      {cookieOpen && profile && <CookieModal profile={{ id: profile.id, name: p.name || profile.name }} onClose={() => setCookieOpen(false)} onStaged={staged => setP(s => ({ ...s, cookies: staged }))} />}
    </div>
  );
};

const stagedN = (c?: string) => { try { const j = JSON.parse(c || '[]'); return Array.isArray(j) ? j.length : 0; } catch { return 0; } };

function blank(): Profile {
  return { id: '', name: '', group: '', tags: [], notes: '', storage: 'local', proxyId: '', inlineProxy: undefined, startUrls: [], cookies: '', createdAt: 0, lastRun: 0, cloudVersion: 0, syncedAt: 0,
    fingerprint: { deviceType: 'desktop', os: 'windows', osVersion: '11', browserVersion: '141', userAgent: '', platform: 'Win32', screenWidth: 1920, screenHeight: 1080, pixelRatio: 1, colorDepth: 24, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 0, webglVendor: '', webglRenderer: '', languages: ['en-US', 'en'], languageFromIP: true, timezone: 'America/New_York', timezoneFromIP: true, geoFromIP: true, latitude: 0, longitude: 0, canvasNoise: true, audioNoise: true, webglNoise: true, clientRectNoise: true, noiseSeed: Math.floor(Math.random() * 1e12), webrtc: 'proxy', fonts: [], doNotTrack: false, hideMediaDevices: false } as Fingerprint };
}
