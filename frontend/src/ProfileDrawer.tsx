import React, { useEffect, useState } from 'react';
import { CountrySelect, LanguagePicker, TimezoneSelect, countryOfTz } from './locales';
import { api, Profile, Proxy, Fingerprint } from './api';
import { Field, Toggle, Icon, flag, useToast, useExit, Segmented, Avatar } from './ui';

const OSES = [['windows', 'Windows'], ['macos', 'macOS'], ['linux', 'Linux']];


export const ProfileDrawer: React.FC<{
  profile: Profile | null; proxies: Proxy[]; onClose: () => void; onSaved: () => void;
}> = ({ profile, proxies, onClose, onSaved }) => {
  const toast = useToast();
  const [tab, setTab] = useState<'basic' | 'fp' | 'adv'>('basic');
  const [p, setP] = useState<Profile>(() => profile ? JSON.parse(JSON.stringify(profile)) : blank());
  const [gpus, setGpus] = useState<{ vendor: string; renderer: string }[]>([]);
  const [screens, setScreens] = useState<string[]>([]);
  const fp = p.fingerprint;
  const [out, close] = useExit(onClose);
  const [flash, setFlash] = useState(0);
  const setFP = (patch: Partial<Fingerprint>) => setP(s => ({ ...s, fingerprint: { ...s.fingerprint, ...patch } as Fingerprint }));

  useEffect(() => {
    api.GPUOptions(fp.os).then(setGpus);
    api.ScreenOptions(fp.os).then(setScreens);
  }, [fp.os]);

  const native = !!(p as any).native;
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
            <div className="grid3">
              <Field label="操作系统"><select value={fp.os} onChange={e => changeOS(e.target.value)}>{OSES.filter(([v]) => !native || v === 'windows').map(([v, l]) => <option key={v} value={v}>{l}</option>)}</select></Field>
              <Field label="系统版本"><input type="text" value={fp.osVersion} onChange={e => setFP({ osVersion: e.target.value })} /></Field>
              <Field label="浏览器大版本"><input type="text" value={fp.browserVersion} onChange={e => setFP({ browserVersion: e.target.value })} /></Field>
            </div>
            <Field label="User-Agent"><textarea style={{ minHeight: 54 }} value={fp.userAgent} onChange={e => setFP({ userAgent: e.target.value })} /></Field>
            <div className="grid2">
              <Field label="平台 (navigator.platform)"><input type="text" value={fp.platform} onChange={e => setFP({ platform: e.target.value })} /></Field>
              <Field label="分辨率"><select value={scr} onChange={e => setScr(e.target.value)}>{screens.map(s => <option key={s} value={s}>{s.replace('@', '  @')}</option>)}{!screens.includes(scr) && <option value={scr}>{scr}</option>}</select></Field>
            </div>
            <div className="grid3">
              <Field label="CPU 核心数"><select value={fp.hardwareConcurrency} onChange={e => setFP({ hardwareConcurrency: +e.target.value })}>{[2, 4, 6, 8, 12, 16, 24].map(n => <option key={n} value={n}>{n}</option>)}</select></Field>
              <Field label="内存 (GB)"><select value={fp.deviceMemory} onChange={e => setFP({ deviceMemory: +e.target.value })}>{[2, 4, 8, 16, 32].map(n => <option key={n} value={n}>{n}</option>)}</select></Field>
              <Field label={native ? '触控点（继承模板）' : '触控点'}><input disabled={native} type="number" value={fp.maxTouchPoints} onChange={e => setFP({ maxTouchPoints: +e.target.value })} /></Field>
            </div>
            <Field label="WebGL 显卡">
              <select value={fp.webglRenderer} onChange={e => { const g = gpus.find(x => x.renderer === e.target.value); if (g) setFP({ webglVendor: g.vendor, webglRenderer: g.renderer }); }}>
                {gpus.map(g => <option key={g.renderer} value={g.renderer}>{g.renderer}</option>)}
                {!gpus.find(g => g.renderer === fp.webglRenderer) && <option value={fp.webglRenderer}>{fp.webglRenderer}</option>}
              </select>
            </Field>
            <div className="kv"><span className="k">WebGL Vendor</span><span className="v">{fp.webglVendor}</span></div>
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
            <div className="kv"><span className="k">字体数量</span><span className="v">{(fp.fonts || []).length} 种</span></div>
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
