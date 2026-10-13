import React, { useEffect, useState } from 'react';
import { SearchSelect, FontPicker, CountrySelect, LanguagePicker, TimezoneSelect, countryOfTz } from './locales';
import { api, Profile, Proxy, Fingerprint } from './api';
import { Field, Toggle, Icon, flag, useToast, useExit, Segmented, Avatar } from './ui';

type FPCatalog = { osVersions: string[]; browserMajors: string[]; userAgents: string[]; platforms: string[]; vendors: string[]; gpus: { vendor: string; renderer: string }[]; fonts: string[] };
const EMPTY_CAT: FPCatalog = { osVersions: [], browserMajors: [], userAgents: [], platforms: [], vendors: [], gpus: [], fonts: [] };
// Keep the current value visible even when it is outside the catalog (e.g. an imported profile).
const opt = <T,>(list: T[], cur: T): T[] => (cur === '' || cur === undefined || cur === null || list.includes(cur)) ? list : [cur, ...list];
const osVerLabel = (os: string, v: string) => os === 'windows' ? `Windows ${v}` : os === 'macos' ? `macOS ${v}` : `Linux ${v}`;
const OSES = [['windows', 'Windows'], ['macos', 'macOS'], ['linux', 'Linux']];


export const ProfileDrawer: React.FC<{
  profile: Profile | null; proxies: Proxy[]; onClose: () => void; onSaved: () => void;
}> = ({ profile, proxies, onClose, onSaved }) => {
  const toast = useToast();
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

  useEffect(() => {
    api.GPUOptions(fp.os).then(setGpus);
    api.FingerprintCatalog(fp.os).then(c => setCat({ ...EMPTY_CAT, ...(c as any) })).catch(() => setCat(EMPTY_CAT));
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
                {proxies.map(px => <option key={px.id} value={px.id}>{flag(px.country)} · {px.name} · {px.type}{px.status === 'ok' ? ` · ${px.city || px.ip}` : ''}</option>)}
              </select>
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
            {!native && devices.length > 0 && <Field label={`设备预设（${devices.length} 台 ${OSES.find(o => o[0] === fp.os)?.[1] || ''} 整机）`} hint="选择一台真实机型，显卡、分辨率、CPU、内存一次填好"><SearchSelect value={device} options={devices.map(d => [d.id, d.name])} onChange={applyDevice} placeholder="搜索机型，如 4060 / M3 / Iris" /></Field>}
            <div className="grid3">
              <Field label="操作系统"><select value={fp.os} onChange={e => changeOS(e.target.value)}>{OSES.filter(([v]) => !native || v === 'windows').map(([v, l]) => <option key={v} value={v}>{l}</option>)}</select></Field>
              <Field label="系统版本"><select value={fp.osVersion} onChange={e => setFP({ osVersion: e.target.value })}>{opt(cat.osVersions, fp.osVersion).map(v => <option key={v} value={v}>{osVerLabel(fp.os, v)}</option>)}</select></Field>
              <Field label="浏览器大版本"><select value={fp.browserVersion} onChange={e => setMajor(e.target.value)}>{opt(cat.browserMajors, fp.browserVersion).map(v => <option key={v} value={v}>Chrome {v}</option>)}</select></Field>
            </div>
            <Field label="User-Agent" hint={`只列出 ${OSES.find(o => o[0] === fp.os)?.[1]} 的 UA；选中后浏览器大版本同步`}>
              <select value={fp.userAgent} onChange={e => setUA(e.target.value)}>{opt(cat.userAgents, fp.userAgent).map(v => <option key={v} value={v}>{v}</option>)}</select>
            </Field>
            <div className="grid2">
              <Field label="平台 (navigator.platform)"><select value={fp.platform} onChange={e => setFP({ platform: e.target.value })}>{opt(cat.platforms, fp.platform).map(v => <option key={v} value={v}>{v}</option>)}</select></Field>
              <Field label="分辨率"><select value={scr} onChange={e => setScr(e.target.value)}>{screens.map(s => <option key={s} value={s}>{s.replace('@', '  @')}</option>)}{!screens.includes(scr) && <option value={scr}>{scr}</option>}</select></Field>
            </div>
            <div className="grid3">
              <Field label="CPU 核心数"><select value={fp.hardwareConcurrency} onChange={e => setFP({ hardwareConcurrency: +e.target.value })}>{opt([2, 4, 6, 8, 10, 12, 14, 16, 20, 24, 32], fp.hardwareConcurrency).map(n => <option key={n} value={n}>{n}</option>)}</select></Field>
              <Field label="内存 (GB)" hint="Chrome 上报最高 8"><select value={fp.deviceMemory} onChange={e => setFP({ deviceMemory: +e.target.value })}>{opt([2, 4, 8], fp.deviceMemory).map(n => <option key={n} value={n}>{n}</option>)}</select></Field>
              <Field label={native ? '触控点（继承模板）' : '触控点'}><select disabled={native} value={fp.maxTouchPoints} onChange={e => setFP({ maxTouchPoints: +e.target.value })}>{opt(fp.os === 'windows' ? [0, 10] : [0], fp.maxTouchPoints).map(n => <option key={n} value={n}>{n === 0 ? '0（无触屏）' : `${n}（触屏）`}</option>)}</select></Field>
            </div>
            <div className="grid2">
              <Field label="WebGL Vendor"><select value={fp.webglVendor} onChange={e => setVendor(e.target.value)}>{opt(cat.vendors, fp.webglVendor).map(v => <option key={v} value={v}>{v}</option>)}</select></Field>
              <Field label="WebGL 显卡" hint="只列出所选 Vendor 的显卡">
                <SearchSelect value={fp.webglRenderer} options={cat.gpus.filter(g => g.vendor === fp.webglVendor).map(g => [g.renderer, g.renderer] as [string, string])} onChange={v => setFP({ webglRenderer: v })} placeholder="搜索显卡，如 4060 / Iris / M3" />
              </Field>
            </div>
          </div>}

          {tab === 'adv' && native && <div className="tabpane">
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v, timezoneFromIP: false })} /></Field>
            <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v, languageFromIP: false })} /></Field>
            <Field label="二进制模板"><select value={p.native?.templateKey} onChange={e => setP({ ...p, native: { ...p.native!, templateKey: e.target.value } as any })}><option value="windows-intel-a">Windows Intel · A</option><option value="windows-intel-a2">Windows Intel · A2</option><option value="windows-intel-b">Windows Intel · B</option></select></Field>
            <div className="sub">字体、媒体设备、语音、WebGL 参数和未映射的噪声开关继承模板。高级配置可在「指纹库 / JSON」导出启动 JSON，修改后导入为新环境。此模式直接使用内核和 CDP，需保持 GhostBrowser 运行。</div>
          </div>}

          {tab === 'adv' && !native && <div className="tabpane">
            <div className="section-title">地区 — 跟随代理出口 IP</div>
            <Field label="国家/地区预设" hint="一键填入该国家的时区、语言和地理坐标"><CountrySelect value={countryOfTz(fp.timezone)} onChange={c => setFP({ timezone: c.tz, languages: c.langs, latitude: c.lat, longitude: c.lon, timezoneFromIP: false, languageFromIP: false, geoFromIP: false })} /></Field>
            <Toggle on={fp.timezoneFromIP} onChange={v => setFP({ timezoneFromIP: v })} label="时区跟随 IP" hint="关闭则使用下方手动时区" />
            {!fp.timezoneFromIP && <Field label="时区"><TimezoneSelect value={fp.timezone} onChange={v => setFP({ timezone: v })} /></Field>}
            <Toggle on={fp.languageFromIP} onChange={v => setFP({ languageFromIP: v })} label="语言跟随 IP" hint="按出口国家设置 Accept-Language" />
            {!fp.languageFromIP && <Field label="语言（按优先级）"><LanguagePicker value={fp.languages || []} onChange={v => setFP({ languages: v })} /></Field>}
            <Toggle on={fp.geoFromIP} onChange={v => setFP({ geoFromIP: v })} label="地理定位跟随 IP" />
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
              <FontPicker value={fp.fonts || []} all={cat.fonts} onChange={v => setFP({ fonts: v })} />
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
