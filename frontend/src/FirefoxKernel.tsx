// Firefox kernel (Profile.kernel === 'firefox'): detection, one-click install
// of the official Firefox, and the honest fingerprint panes for the drawer.
// Firefox is driven by user.js prefs + WebDriver BiDi, so only what Firefox
// itself supports is shown here.
import React, { useEffect, useState } from 'react';
import { api, onEvent, Fingerprint } from './api';
import { Icon, Segmented, Field, useToast } from './ui';
import { KernelProgress, KernelProgressView } from './KernelInstall';
import { Row, Sw, NumberStepper } from './FpControls';

type FP = Fingerprint & Record<string, any>;
type SetFP = (patch: Partial<Fingerprint> & Record<string, any>) => void;

export type FirefoxInfo = { path: string; auto: string; version: string; esr: boolean; custom: boolean; missing: boolean; downloadUrl: string };

/** Generic fox glyph (not Mozilla's trademark logo). */
export const FoxIcon: React.FC<{ s?: number }> = ({ s = 14 }) => (
  <svg className="fox-ic" width={s} height={s} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinejoin="round" strokeLinecap="round" aria-hidden>
    <path d="M4 3.5 8.6 8h6.8L20 3.5l-.6 8.2c-.3 4.4-3.6 8.3-7.4 8.8-3.8-.5-7.1-4.4-7.4-8.8Z" />
    <path d="M8.2 12.6l2.2 1.3M15.8 12.6l-2.2 1.3M10.6 17h2.8L12 18.4Z" />
  </svg>
);

export function useFirefoxInfo(dep?: any): [FirefoxInfo | null, () => void] {
  const [info, setInfo] = useState<FirefoxInfo | null>(null);
  const load = () => api.FirefoxInfo().then((x: FirefoxInfo) => setInfo(x)).catch(() => setInfo(null));
  useEffect(() => { load(); }, [dep]);
  return [info, load];
}

export const ffVersionLabel = (i: FirefoxInfo | null) => !i ? '检测中…' : i.path ? `Firefox ${i.version || '版本未知'}` : 'Firefox 未安装';

const ffIdle: KernelProgress = { phase: 'idle', running: false, source: 'esr', done: 0, total: 0, pct: 0, speed: 0, eta: -1, message: '' };

export function useFirefoxInstall(onDone?: (p: KernelProgress) => void) {
  const [p, setP] = useState<KernelProgress>(ffIdle);
  useEffect(() => {
    let last = '';
    api.FirefoxInstallStatus().then((x: KernelProgress) => { if (x) { setP(x); last = x.phase; } }).catch(() => {});
    return onEvent('firefox:install', (x: KernelProgress) => { setP(x); if (x.phase === 'done' && last !== 'done') onDone?.(x); last = x.phase; });
  }, []);
  return p;
}

/** One-click install of the official Firefox (ESR / release) from Mozilla. */
export const FirefoxInstaller: React.FC<{ onInstalled?: (path: string) => void; compact?: boolean }> = ({ onInstalled, compact }) => {
  const toast = useToast();
  const [ch, setCh] = useState<'esr' | 'release'>('esr');
  const p = useFirefoxInstall(x => { if (x.path) onInstalled?.(x.path); toast(`Firefox ${x.version || ''} 安装完成`, 'ok'); });
  const start = async () => { try { await api.FirefoxInstallStart(ch); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const show = p.phase !== 'idle';
  return (
    <div className={`ki ff-ki ${compact ? 'compact' : ''}`}>
      {!p.running && !compact && <div className="ki-src">
        <Segmented<'esr' | 'release'> className="compact" value={ch} onChange={setCh} items={[['esr', '延长支持版 ESR'], ['release', '正式版']]} />
        <span className="sub" style={{ margin: 0 }}>{ch === 'esr' ? '一年一个大版本，指纹最稳定，推荐' : '每 4 周更新，和大多数 Firefox 用户一致'}</span>
      </div>}
      {show
        ? <KernelProgressView p={p} onCancel={() => api.FirefoxInstallCancel()} onRetry={start} compact={compact} product="Firefox" />
        : <div className="ki-cta">
          <div className="ki-cta-ic ff"><FoxIcon s={18} /></div>
          <div className="ki-cta-txt"><b>一键安装官方 Firefox</b><span>从 Mozilla 官方下载（HTTPS），按官方 SHA-512 校验后静默安装到程序数据目录 · 约 70 MB</span></div>
          <button className="btn primary" onClick={start}><Icon.download s={14} /> 一键安装</button>
        </div>}
    </div>
  );
};

/** Settings card: path + pick + auto-detect, version, download link, installer. */
export const FirefoxSettingsCard: React.FC<{ value: string; onChange: (path: string) => void; onInstalled: (path: string) => void; index?: number }> = ({ value, onChange, onInstalled, index = 2 }) => {
  const [info, reload] = useFirefoxInfo(value);
  const [t, setT] = useState(0);
  useEffect(() => { const h = setTimeout(() => setT(x => x + 1), 300); return () => clearTimeout(h); }, [value]);
  useEffect(() => { reload(); }, [t]);
  const pick = async () => { const f = await api.PickFirefox(); if (f) onChange(f); };
  return (
    <div id="ff-card" className="panel pad setting-card spot rise" style={{ ['--i' as any]: index }}>
      <div className="head"><span className="ic ff"><FoxIcon s={17} /></span><div><h4>Firefox 内核</h4><span>Mozilla Firefox + 偏好设置（user.js），通过 WebDriver BiDi 控制</span></div>
        <span className={`badge ${info?.path ? 'ok' : 'warn'} ff-ver`} title={info?.path || ''}><span className="d" />{info ? (info.path ? info.version || '版本未知' : '未安装') : '检测中…'}</span></div>
      <Field label="firefox.exe 路径" hint={`留空自动检测：${info?.auto || '没找到已安装的 Firefox'}`}>
        <div className="row"><input type="text" value={value} onChange={e => onChange(e.target.value)} placeholder="C:\Program Files\Mozilla Firefox\firefox.exe" />
          <button className="btn" style={{ flex: 'none' }} onClick={pick}><Icon.folder s={15} /> 选择</button>
          {value && <button className="btn ghost" style={{ flex: 'none' }} onClick={() => onChange('')}><Icon.refresh s={14} /> 自动检测</button>}</div>
      </Field>
      {info?.missing && <div className="notice warn"><Icon.alert /><span className="grow">找不到这个文件，将改用自动检测到的 Firefox。</span></div>}
      {info && !info.path && <div className="notice warn ff-miss"><Icon.alert /><span className="grow">本机没有安装 Firefox。可以一键安装官方版本，或<a className="ff-link" onClick={() => api.FirefoxDownloadPage()}>打开 Mozilla 官方下载页</a>手动安装。</span></div>}
      <FirefoxInstaller onInstalled={p => { onInstalled(p); reload(); }} />
      <div className="sub" style={{ marginTop: 12 }}>每个环境使用独立的 Firefox 配置目录。支持：代理（含账号密码，经本地中转）、语言、时区、WebRTC、DNT / GPC、地理位置、WebGL、字体可见范围、严格模式、CPU 核心数、禁止图片、启动页和窗口大小。不伪装显卡、Canvas 和系统。RPA、同步器暂不支持 Firefox。</div>
    </div>
  );
};

/** Inline notice in the drawer when no Firefox is installed. */
export const FirefoxMissingNotice: React.FC = () => {
  const [info, reload] = useFirefoxInfo();
  if (!info || info.path) return null;
  return (
    <div className="notice warn ki-notice">
      <div><b>没有找到 Firefox</b><span>这个环境需要 Firefox 才能启动。可以一键安装官方 ESR，或<a className="ff-link" onClick={() => api.FirefoxDownloadPage()}>前往官方下载页</a>。</span></div>
      <FirefoxInstaller compact onInstalled={() => reload()} />
    </div>
  );
};

const FONT_VIS: [string, string][] = [['0', '默认'], ['1', '仅系统基础'], ['2', '+语言包'], ['3', '全部']];
const fontHint = (v: number) => v === 1 ? '只暴露操作系统自带的基础字体，字体指纹最常见' : v === 2 ? '基础字体加已安装语言包的字体' : v === 3 ? '暴露本机安装的全部字体（容易被识别）' : '跟随 Firefox 默认（普通窗口显示全部字体）';

/** 「指纹配置」 for a Firefox environment. */
export const FirefoxFpPane: React.FC<{ fp: FP; setFP: SetFP; info: FirefoxInfo | null }> = ({ fp, setFP, info }) => {
  const ua = String(fp.userAgent || '');
  const notFirefoxUA = ua !== '' && !/Firefox\/\d+/.test(ua);
  const hc = fp.hardwareConcurrency || 0;
  return <>
    <div className="ff-facts">
      <div className="ff-fact"><span>浏览器</span><b className="ell">{ffVersionLabel(info)}</b></div>
      <div className="ff-fact"><span>系统 / 平台</span><b className="ell">跟随本机</b></div>
      <div className="ff-fact"><span>显卡 / Canvas</span><b className="ell">真实</b></div>
    </div>
    <div className="sub" style={{ margin: '0 0 12px' }}>Firefox 只能通过自己的偏好设置调整指纹，下面只列出它真正支持的项目；显卡、Canvas 噪声、屏幕分辨率和系统平台保持本机真实值。</div>

    <div className="section-title" style={{ marginTop: 0 }}>User-Agent</div>
    <div className="fprows">
      <Row title="覆盖 User-Agent" hint="写入 general.useragent.override：只改 UA 字符串，navigator.oscpu、平台和引擎特性仍是真实的" onClick={() => setFP({ ffUaOverride: !fp.ffUaOverride })}
        control={<Sw on={!!fp.ffUaOverride} onChange={v => setFP({ ffUaOverride: v })} />}>
        {fp.ffUaOverride && <div className="field" style={{ marginBottom: 0 }}>
          <textarea className="mono ff-ua" value={ua} onChange={e => setFP({ userAgent: e.target.value })} placeholder="Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0" spellCheck={false} />
          <div className="notice warn ff-warn"><Icon.alert /><span className="grow">{notFirefoxUA ? '这不是 Firefox 的 UA：网站能从引擎特性看出是 Firefox，伪装成 Chrome 很容易被识别。' : '覆盖后 UA 可能与真实版本和系统不一致，建议只填同版本的 Firefox UA，或保持关闭。'}</span></div>
        </div>}
      </Row>
    </div>

    <div className="section-title">硬件</div>
    <div className="fprows">
      <Row title="CPU 核心数上限" hint={hc ? `dom.maxHardwareConcurrency：navigator.hardwareConcurrency 最多报告 ${hc} 核（只能调低，不能高于本机）` : '不限制，报告本机真实核心数'}
        control={<NumberStepper value={hc} min={0} max={64} onChange={v => setFP({ hardwareConcurrency: v })} suffix={hc ? '核' : '不限'} />} />
    </div>
  </>;
};

/** Region extras for Firefox: strict mode lock + font visibility (in 「地区与字体」). */
export const FirefoxFontPane: React.FC<{ fp: FP; setFP: SetFP }> = ({ fp, setFP }) => {
  const v = String(fp.ffFontVisibility || 0);
  return <>
    <div className="section-title">字体</div>
    <div className="fprows">
      <Row title="字体可见范围" hint={fontHint(+v)} control={<Segmented className="compact" value={v} onChange={x => setFP({ ffFontVisibility: +x })} items={FONT_VIS} />} />
    </div>
  </>;
};

/** 「防护」 for a Firefox environment. */
export const FirefoxGuardPane: React.FC<{ fp: FP; setFP: SetFP }> = ({ fp, setFP }) => {
  const strict = !!fp.ffStrict;
  const setStrict = (on: boolean) => setFP(on ? { ffStrict: true, languageFromIP: false, timezoneFromIP: false } : { ffStrict: false });
  return <>
    <div className="notice fpk-ok ff-note"><FoxIcon s={15} /><div className="grow">Firefox 内核：以下项目写入该环境的 user.js，每次启动生效。</div></div>
    <div className="section-title">严格模式</div>
    <div className="fprows">
      <Row title="严格防指纹（resistFingerprinting）" hint={strict ? '已开启：时区固定为 UTC+0、语言为 en-US、屏幕与计时器取整；「语言 / 时区跟随 IP」已关闭' : 'Firefox 自带的抗指纹模式，与「语言跟随 IP」「时区跟随 IP」互斥'} onClick={() => setStrict(!strict)}
        control={<Sw on={strict} onChange={setStrict} />} />
    </div>
    <div className="section-title">媒体与 API</div>
    <div className="fprows">
      <Row title="WebRTC" hint={fp.webrtc === 'real' ? '会暴露本机真实 IP，不推荐' : fp.webrtc === 'disabled' ? 'media.peerconnection.enabled = false，音视频通话网站不可用' : '只用默认路由地址、不暴露内网 IP，有代理时只走代理（推荐）'}
        control={<Segmented className="compact" value={fp.webrtc || 'proxy'} onChange={v => setFP({ webrtc: v })} items={[['proxy', '代理 IP'], ['real', '真实'], ['disabled', '禁用']]} />} />
      <Row title="WebGL" hint={fp.ffWebglOff ? 'webgl.disabled：网站拿不到显卡信息，部分 3D 网站不可用' : '保留 WebGL（显卡为本机真实型号）'}
        control={<Segmented className="compact" value={fp.ffWebglOff ? 'off' : 'real'} onChange={x => setFP({ ffWebglOff: x === 'off' })} items={[['real', '真实'], ['off', '禁用']]} />} />
      <Row title="WebGPU" hint={fp.webgpu === 'off' ? 'dom.webgpu.enabled = false' : '跟随 Firefox 默认'}
        control={<Segmented className="compact" value={fp.webgpu === 'off' ? 'off' : 'real'} onChange={x => setFP({ webgpu: x })} items={[['real', '默认'], ['off', '关闭']]} />} />
      <Row title="隐藏语音列表" hint="media.webspeech.synth.enabled = false：speechSynthesis 不可用，不暴露本机语音" onClick={() => setFP({ speechVoices: fp.speechVoices === 'off' ? 'real' : 'off' })}
        control={<Sw on={fp.speechVoices === 'off'} onChange={v => setFP({ speechVoices: v ? 'off' : 'real' })} />} />
    </div>
    <div className="section-title">隐私信号</div>
    <div className="fprows">
      <Row title="Do Not Track" hint="navigator.doNotTrack = 1，并发送 DNT 请求头" onClick={() => setFP({ doNotTrack: !fp.doNotTrack })} control={<Sw on={!!fp.doNotTrack} onChange={v => setFP({ doNotTrack: v })} />} />
      <Row title="Global Privacy Control" hint="navigator.globalPrivacyControl = true，并发送 Sec-GPC 请求头" onClick={() => setFP({ gpc: !fp.gpc })} control={<Sw on={!!fp.gpc} onChange={v => setFP({ gpc: v })} />} />
    </div>
  </>;
};

/** 「启动」 for a Firefox environment. */
export const FirefoxLaunchPane: React.FC<{ fp: FP; setFP: SetFP }> = ({ fp, setFP }) => {
  const custom = fp.windowMode === 'custom';
  const num = (k: string, label: string, ph: string) => (
    <div className="field" style={{ marginBottom: 0 }}><label>{label}</label>
      <input type="number" min={0} value={fp[k] || ''} placeholder={ph} onChange={e => setFP({ [k]: e.target.value === '' ? 0 : Math.max(0, Math.round(+e.target.value)) })} /></div>
  );
  return <>
    <div className="notice fpk-ok ff-note"><FoxIcon s={15} /><div className="grow">Firefox 内核：独立配置目录（-profile -no-remote），启动页取自「基本信息」。</div></div>
    <div className="section-title">窗口</div>
    <div className="fprows">
      <Row title="窗口大小" hint={custom ? '按下方宽高打开第一个窗口（-width / -height）' : `按指纹分辨率自动计算（${fp.screenWidth}×${fp.screenHeight} 的 80%）`}
        control={<Segmented className="compact" value={custom ? 'custom' : 'auto'} onChange={m => setFP({ windowMode: m, ...(m === 'custom' && !fp.windowWidth ? { windowWidth: Math.round((fp.screenWidth || 1920) * 0.8), windowHeight: Math.round((fp.screenHeight || 1080) * 0.8) } : {}) })} items={[['auto', '自动'], ['custom', '自定义']]} />}>
        {custom && <div className="grid2">{num('windowWidth', '宽 W', '1280')}{num('windowHeight', '高 H', '800')}</div>}
      </Row>
    </div>
    <div className="section-title">流量</div>
    <div className="fprows">
      <Row title="禁止加载图片" hint="permissions.default.image = 2，节省流量" onClick={() => setFP({ blockImages: !fp.blockImages })} control={<Sw on={!!fp.blockImages} onChange={v => setFP({ blockImages: v })} />} />
      <Row title="静音" hint="media.volume_scale = 0，所有标签页不出声" onClick={() => setFP({ blockSound: !fp.blockSound })} control={<Sw on={!!fp.blockSound} onChange={v => setFP({ blockSound: v })} />} />
    </div>
    <div className="section-title">不支持的选项</div>
    <div className="ff-unsup">
      {['自定义启动参数', '禁止加载视频', '硬件加速开关', '扩展中心的扩展', 'RPA 自动化', '多窗口同步器'].map(x => <span key={x} title="Firefox 内核暂不支持">{x}</span>)}
    </div>
  </>;
};
