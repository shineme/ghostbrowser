import React, { useEffect, useState } from 'react';
import { Combo, SearchSelect, FontPicker, CountrySelect, LanguagePicker, TimezoneSelect, countryOfTz } from './locales';
import { api, Profile, Proxy, Fingerprint } from './api';
import { Field, Toggle, Icon, flag, useToast, useExit, Segmented, Avatar } from './ui';

type FPCatalog = { osVersions: string[]; browserMajors: string[]; userAgents: string[]; platforms: string[]; vendors: string[]; gpus: { vendor: string; renderer: string; custom?: string }[]; fonts: string[]; screens: string[]; cores: string[]; memory: string[]; custom: Record<string, string[]> };
const EMPTY_CAT: FPCatalog = { osVersions: [], browserMajors: [], userAgents: [], platforms: [], vendors: [], gpus: [], fonts: [], screens: [], cores: [], memory: [], custom: {} };
// Keep the current value visible even when it is outside the catalog (e.g. an imported profile).
const opt = <T,>(list: T[], cur: T): T[] => (cur === '' || cur === undefined || cur === null || list.includes(cur)) ? list : [cur, ...list];
const osVerLabel = (os: string, v: string) => os === 'windows' ? `Windows ${v}` : os === 'macos' ? `macOS ${v}` : `Linux ${v}`;
const OSES = [['windows', 'Windows'], ['macos', 'macOS'], ['linux', 'Linux']];


export const ProfileDrawer: React.FC<{
  profile: Profile | null; proxies: Proxy[]; onClose: () => void; onSaved: () => void;
}> = ({ profile, proxies, onClose, onSaved }) => {
  const toast = useToast();
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
  const [tab, setTab] = useState<'basic' | 'fp' | 'adv'>('basic');
  const [p, setP] = useState<Profile>(() => profile ? JSON.parse(JSON.stringify(profile)) : blank());
  const [gpus, setGpus] = useState<{ vendor: string; renderer: string }[]>([]);
  const [devices, setDevices] = useState<any[]>([]);
  const [device, setDevice] = useState('');
  const [cat, setCat] = useState<FPCatalog>(EMPTY_CAT);
  const [screens, setScreens] = useState<string[]>([]);
  const fp = p.fingerprint;
  const [out, close] = useExit(onClose);
  const [flash, setFlash] = useState(0);
  const setFP = (patch: Partial<Fingerprint>) => setP(s => ({ ...s, fingerprint: { ...s.fingerprint, ...patch } as Fingerprint }));
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
        {geo?.ok && !(fp.timezoneFromIP && fp.languageFromIP && fp.geoFromIP) && <button type="button" className="btn sm primary" onClick={() => runGeo(true)}>全部跟随 IP</button>}
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

  useEffect(() => {
    api.GPUOptions(fp.os).then(setGpus);
    loadCat();
    api.DevicePresets(fp.os).then(d => setDevices(d || [])).catch(() => setDevices([]));
    api.ScreenOptions(fp.os).then(setScreens);
  }, [fp.os]);

  const native = !!(p as any).native;
  const applyDevice = (id: string) => {
    const d = devices.find(x => x.id === id); if (!d) return;
    setDevice(id);
    setFP({ osVersion: d.osVersion, webglVendor: d.webglVendor, webglRenderer: d.webglRenderer, screenWidth: d.screenWidth, screenHeight: d.screenHeight, pixelRatio: d.pixelRatio, hardwareConcurrency: d.hardwareConcurrency, deviceMemory: d.deviceMemory, maxTouchPoints: d.maxTouchPoints });
    setFlash(x => x + 1);
  };
  const loadCat = () => api.FingerprintCatalog(fp.os).then(c => setCat({ ...EMPTY_CAT, ...(c as any) })).catch(() => setCat(EMPTY_CAT));
  const libAdd = async (field: string, v: string) => { try { await api.LibraryAdd(fp.os, field, v); toast('已存入我的指纹库'); loadCat(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const libRemove = async (field: string, v: string) => { try { await api.LibraryRemove(fp.os, field, v); toast('已从指纹库删除'); loadCat(); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const lib = (field: string, options: string[]) => ({ options, custom: cat.custom?.[field] || [], onSave: (v: string) => libAdd(field, v), onRemove: (v: string) => libRemove(field, v) });
  const vendorGpus = cat.gpus.filter(g => g.vendor === fp.webglVendor);
  const [scrText, setScrText] = useState('');
  useEffect(() => { setScrText(`${fp.screenWidth}x${fp.screenHeight}@${fp.pixelRatio}`); }, [fp.screenWidth, fp.screenHeight, fp.pixelRatio]);
  // A typed UA that belongs to another OS is allowed, but flagged.
  const uaOS = /Windows NT/.test(fp.userAgent) ? 'windows' : /Macintosh|Mac OS X/.test(fp.userAgent) ? 'macos' : /Linux|X11/.test(fp.userAgent) ? 'linux' : '';
  const uaWarn = uaOS && uaOS !== fp.os ? `这个 UA 属于 ${OSES.find(o => o[0] === uaOS)?.[1]}，与所选系统不一致，容易被识别` : '';
  const savePreset = async () => {
    try {
      const d: any = await api.SaveDevicePreset({ id: device.startsWith('my-') ? device : '', name: '', os: fp.os, osVersion: fp.osVersion, webglVendor: fp.webglVendor, webglRenderer: fp.webglRenderer, screenWidth: fp.screenWidth, screenHeight: fp.screenHeight, pixelRatio: fp.pixelRatio, hardwareConcurrency: fp.hardwareConcurrency, deviceMemory: fp.deviceMemory, maxTouchPoints: fp.maxTouchPoints } as any);
      setDevice(d.id); toast(`已保存设备预设：${d.name}`);
      api.DevicePresets(fp.os).then(x => setDevices(x || []));
    } catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const delPreset = async () => { try { await api.DeleteDevicePreset(device); setDevice(''); toast('已删除设备预设'); api.DevicePresets(fp.os).then(x => setDevices(x || [])); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  // Linked edits: UA and browser major always move together; renderer always belongs to the vendor.
  const setMajor = (m: string) => setFP({ browserVersion: m, userAgent: cat.userAgents.find(u => u.includes(`Chrome/${m}.`)) || fp.userAgent });
  const setUA = (ua: string) => { const m = /Chrome\/(\d+)\./.exec(ua); setFP({ userAgent: ua, ...(m ? { browserVersion: m[1] } : {}) }); };
  const setVendor = (v: string) => { const g = cat.gpus.find(x => x.vendor === v); setFP({ webglVendor: v, webglRenderer: g ? g.renderer : fp.webglRenderer }); };
  const regen = async () => { const f = await api.NewFingerprint(fp.os); setFP(f as any); setFlash(x => x + 1); toast('已生成新指纹'); };
  const changeOS = async (os: string) => { const f = await api.NewFingerprint(os); setFP(f as any); setFlash(x => x + 1); };

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
              <div className="ua">{fp.userAgent || '保存时按所选系统自动生成 User-Agent'}</div>
              <div className="chips">
                <span>{fp.platform}</span><span className="num">{fp.screenWidth}×{fp.screenHeight} @{fp.pixelRatio}x</span>
                <span className="num">{fp.hardwareConcurrency} 核 · {fp.deviceMemory} GB</span>
                <span>{(fp.webglRenderer || 'WebGL 自动').replace(/^ANGLE \(([^,]+),\s*/, '').split(' (')[0].split(',')[0]}</span>
              </div>
            </div>
            <i className="scan" />
          </div>
          <Segmented value={tab} onChange={setTab} items={[['basic', '基本信息', <Icon.layers />], ['fp', '指纹配置', <Icon.fingerprint />], ['adv', '地区与噪声', <Icon.shield />]]} />

          {tab === 'basic' && <div className="tabpane">
            <Field label="环境名称"><input type="text" value={p.name} onChange={e => setP({ ...p, name: e.target.value })} placeholder="例如 FB-Ads-01" autoFocus /></Field>
            <div className="grid2">
              <Field label="分组"><input type="text" value={p.group} onChange={e => setP({ ...p, group: e.target.value })} placeholder="广告投放 / 电商…" /></Field>
              <Field label="标签 (逗号分隔)"><input type="text" value={(p.tags || []).join(', ')} onChange={e => setP({ ...p, tags: e.target.value.split(',').map(s => s.trim()).filter(Boolean) })} /></Field>
            </div>
            <Field label="存储方式">
              <select value={p.storage} onChange={e => setP({ ...p, storage: e.target.value })}>
                <option value="local">本地 — 只存在这台电脑</option>
                <option value="cloud">云端 — 同步到你的 VPS</option>
              </select>
            </Field>
            <Field label="代理">
              <select value={p.proxyId} onChange={e => setP({ ...p, proxyId: e.target.value })}>
                <option value="">不使用代理（直连）</option>
                {[...extraProxies.filter(x => !proxies.some(y => y.id === x.id)), ...proxies].map(px => <option key={px.id} value={px.id}>{flag(px.country)} · {px.name} · {px.type}{px.status === 'ok' ? ` · ${px.city || px.ip}` : ''}</option>)}
              </select>
              <div className="combo-row" style={{ marginTop: 6 }}>
                <input type="text" value={pxText} onChange={e => setPxText(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') quickProxy(); }} placeholder="或直接粘贴一个代理，自动识别格式并检测，如 user:pass@1.2.3.4:8080" />
                <button type="button" className="btn sm combo-btn" disabled={pxBusy || !pxText.trim()} onClick={quickProxy}>{pxBusy ? '检测中…' : '识别并使用'}</button>
              </div>
              {pxInfo && <div className="sub" style={{ marginTop: 4 }}>{pxInfo}</div>}
            </Field>
            <Field label="启动页 (每行一个)"><textarea value={(p.startUrls || []).join('\n')} onChange={e => setP({ ...p, startUrls: e.target.value.split('\n') })} placeholder="https://facebook.com" /></Field>
            <Field label="导入 Cookies (JSON, 下次启动时写入)" hint="支持 Cookie-Editor / EditThisCookie 的导出格式"><textarea value={p.cookies} onChange={e => setP({ ...p, cookies: e.target.value })} placeholder="[ { &quot;name&quot;: …, &quot;domain&quot;: … } ]" /></Field>
            <Field label="备注"><textarea value={p.notes} onChange={e => setP({ ...p, notes: e.target.value })} /></Field>
          </div>}

          {tab === 'fp' && <div className="tabpane">
            {native && <div className="sub" style={{ marginBottom: 12 }}>原生模板 {p.native?.templateKey} · 配置 {p.native?.sourceConfigId}。以下项目会写入 startInfo，高级参数通过启动 JSON 管理。</div>}
            <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 10 }}>
              <button className="btn sm" onClick={regen}><Icon.dice s={14} /> {native ? '加载默认模板值' : '重新随机'}</button>
            </div>
            {!native && devices.length > 0 && <Field label={`设备预设（${devices.length} 台 ${OSES.find(o => o[0] === fp.os)?.[1] || ''} 整机）`} hint="选择一台真实机型，显卡、分辨率、CPU、内存一次填好"><SearchSelect value={device} options={devices.map(d => [d.id, (d.custom ? '★ ' : '') + d.name])} onChange={applyDevice} placeholder="搜索机型，如 4060 / M3 / Iris" /></Field>}
            <div className="grid3">
              <Field label="操作系统"><select value={fp.os} onChange={e => changeOS(e.target.value)}>{OSES.filter(([v]) => !native || v === 'windows').map(([v, l]) => <option key={v} value={v}>{l}</option>)}</select></Field>
              <Field label="系统版本"><Combo value={fp.osVersion} {...lib('osVersions', cat.osVersions)} onChange={v => setFP({ osVersion: v })} render={v => osVerLabel(fp.os, v)} /></Field>
              <Field label="浏览器大版本"><Combo value={fp.browserVersion} {...lib('browserMajors', cat.browserMajors)} onChange={setMajor} render={v => `Chrome ${v}`} /></Field>
            </div>
            <Field label="User-Agent" hint={`可手动输入或粘贴；列表只含 ${OSES.find(o => o[0] === fp.os)?.[1]} 的 UA，★ 为你存入库的。大版本会按 UA 自动同步`}>
              <Combo multiline value={fp.userAgent} {...lib('userAgents', cat.userAgents)} onChange={setUA} placeholder="粘贴 User-Agent，或点 ▾ 选择" />
              {uaWarn && <div className="sub" style={{ color: 'var(--warn, #f5c451)', marginTop: 4 }}>⚠ {uaWarn}</div>}
            </Field>
            <div className="grid2">
              <Field label="平台 (navigator.platform)"><Combo value={fp.platform} {...lib('platforms', cat.platforms)} onChange={v => setFP({ platform: v })} /></Field>
              <Field label="分辨率" hint="格式 宽x高@缩放，如 1920x1080@1.25"><Combo value={scrText} {...lib('screens', cat.screens.length ? cat.screens : screens)} onChange={v => { setScrText(v); setScr(v); }} /></Field>
            </div>
            <div className="grid3">
              <Field label="CPU 核心数"><Combo value={String(fp.hardwareConcurrency ?? '')} {...lib('cores', cat.cores)} onChange={v => /^\d*$/.test(v) && setFP({ hardwareConcurrency: +v })} /></Field>
              <Field label="内存 (GB)" hint="Chrome 上报最高 8"><Combo value={String(fp.deviceMemory ?? '')} {...lib('memory', cat.memory)} onChange={v => /^\d*$/.test(v) && setFP({ deviceMemory: +v })} /></Field>
              <Field label={native ? '触控点（继承模板）' : '触控点'}><select disabled={native} value={fp.maxTouchPoints} onChange={e => setFP({ maxTouchPoints: +e.target.value })}>{opt(fp.os === 'windows' ? [0, 10] : [0], fp.maxTouchPoints).map(n => <option key={n} value={n}>{n === 0 ? '0（无触屏）' : `${n}（触屏）`}</option>)}</select></Field>
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
          </div>}

          {tab === 'adv' && native && <div className="tabpane">
            {geoCard}
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v, timezoneFromIP: false })} /></Field>
            <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v, languageFromIP: false })} /></Field>
            <Field label="二进制模板"><select value={p.native?.templateKey} onChange={e => setP({ ...p, native: { ...p.native!, templateKey: e.target.value } as any })}><option value="windows-intel-a">Windows Intel · A</option><option value="windows-intel-a2">Windows Intel · A2</option><option value="windows-intel-b">Windows Intel · B</option></select></Field>
            <div className="sub">字体、媒体设备、语音、WebGL 参数和未映射的噪声开关继承模板。高级配置可在「指纹库 / JSON」导出启动 JSON，修改后导入为新环境。此模式直接使用内核和 CDP，需保持 GhostBrowser 运行。</div>
          </div>}

          {tab === 'adv' && !native && <div className="tabpane">
            <div className="section-title">地区 — 跟随代理出口 IP</div>
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            {geoCard}
            <Toggle on={fp.timezoneFromIP} onChange={v => setFP({ timezoneFromIP: v })} label="时区跟随 IP" hint={fp.timezoneFromIP && geo?.ok ? `→ ${geo.timezone}` : '关闭则使用下方手动时区'} />
            {!fp.timezoneFromIP && <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v })} /></Field>}
            <Toggle on={fp.languageFromIP} onChange={v => setFP({ languageFromIP: v })} label="语言跟随 IP" hint={fp.languageFromIP && geo?.ok ? `→ ${(geo.languages || []).join(', ')}` : '按出口国家（及地区）设置 Accept-Language 和 navigator.languages'} />
            {!fp.languageFromIP && <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v })} /></Field>}
            <Toggle on={fp.geoFromIP} onChange={v => setFP({ geoFromIP: v })} label="地理定位跟随 IP" hint={fp.geoFromIP && geo?.ok ? `→ ${(+geo.lat).toFixed(4)}, ${(+geo.lon).toFixed(4)}（${geo.city || geo.country}）` : '关闭则使用下方手动坐标'} />
            {!fp.geoFromIP && <div className="grid2"><Field label="纬度"><input type="number" step="0.0001" value={fp.latitude} onChange={e => setFP({ latitude: +e.target.value })} /></Field><Field label="经度"><input type="number" step="0.0001" value={fp.longitude} onChange={e => setFP({ longitude: +e.target.value })} /></Field></div>}

            <div className="section-title">噪声 — 让每个环境的指纹都独一无二</div>
            <Toggle on={fp.canvasNoise} onChange={v => setFP({ canvasNoise: v })} label="Canvas 噪声" hint="按环境种子注入稳定噪声" />
            <Toggle on={fp.webglNoise} onChange={v => setFP({ webglNoise: v })} label="WebGL 噪声" />
            <Toggle on={fp.audioNoise} onChange={v => setFP({ audioNoise: v })} label="AudioContext 噪声" />
            <Toggle on={fp.clientRectNoise} onChange={v => setFP({ clientRectNoise: v })} label="ClientRects 噪声" />
            <Field label="噪声种子 (seed)" hint="同一种子 = 同一环境指纹永远一致"><input type="number" value={fp.noiseSeed} onChange={e => setFP({ noiseSeed: +e.target.value })} /></Field>

            <div className="section-title">其它</div>
            <Field label="WebRTC"><select value={fp.webrtc} onChange={e => setFP({ webrtc: e.target.value })}><option value="proxy">替换为代理 IP（推荐）</option><option value="disabled">完全禁用</option><option value="real">真实（不推荐）</option></select></Field>
            <Toggle on={fp.hideMediaDevices} onChange={v => setFP({ hideMediaDevices: v })} label="隐藏媒体设备" />
            <Toggle on={fp.doNotTrack} onChange={v => setFP({ doNotTrack: v })} label="Do Not Track" />
            <Field label={`字体（${(fp.fonts || []).length} / ${cat.fonts.length} 种 ${OSES.find(o => o[0] === fp.os)?.[1]} 字体）`} hint="只列出该系统自带字体">
              <FontPicker value={fp.fonts || []} all={cat.fonts} onChange={v => setFP({ fonts: v })} onSave={f => libAdd('fonts', f)} />
            </Field>
          </div>}
        </div>

        <div className="dfoot">
          <div className="sub num" style={{ color: 'var(--txt-3)', fontSize: 11 }}>SEED {fp.noiseSeed}</div>
          <div className="spacer" />
          <button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" onClick={save}><Icon.check s={15} /> 保存环境</button>
        </div>
      </div>
    </div>
  );
};

function blank(): Profile {
  return { id: '', name: '', group: '', tags: [], notes: '', storage: 'local', proxyId: '', inlineProxy: undefined, startUrls: [], cookies: '', createdAt: 0, lastRun: 0, cloudVersion: 0, syncedAt: 0,
    fingerprint: { os: 'windows', osVersion: '11', browserVersion: '141', userAgent: '', platform: 'Win32', screenWidth: 1920, screenHeight: 1080, pixelRatio: 1, colorDepth: 24, hardwareConcurrency: 8, deviceMemory: 8, maxTouchPoints: 0, webglVendor: '', webglRenderer: '', languages: ['en-US', 'en'], languageFromIP: true, timezone: 'America/New_York', timezoneFromIP: true, geoFromIP: true, latitude: 0, longitude: 0, canvasNoise: true, audioNoise: true, webglNoise: true, clientRectNoise: true, noiseSeed: Math.floor(Math.random() * 1e12), webrtc: 'proxy', fonts: [], doNotTrack: false, hideMediaDevices: false } as Fingerprint };
}
