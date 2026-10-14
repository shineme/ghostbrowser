import React, { useEffect, useState } from 'react';
import { api, onEvent } from './api';
import { Icon, Segmented, useToast, useConfirm, useExit } from './ui';
import { fmtBytes, fmtSpeed } from './Transfers';

// One-click install of an open-source Chromium kernel (Windows x64 snapshot).
export type KernelProgress = {
  phase: 'idle' | 'resolve' | 'download' | 'extract' | 'verify' | 'done' | 'error' | 'cancelled';
  running: boolean; source: string; revision?: string; version?: string; path?: string; url?: string;
  done: number; total: number; pct: number; speed: number; eta: number; resumed?: boolean;
  message: string; hint?: string; error?: string;
};
export type InstalledKernel = { name: string; version: string; path: string; dir: string; size: number; installed: number; active: boolean };
type Source = 'mirror' | 'official' | 'custom';

const STEPS: [KernelProgress['phase'], string][] = [['resolve', '解析版本'], ['download', '下载'], ['extract', '解压'], ['verify', '校验'], ['done', '完成']];
const fmtEta = (s: number) => s < 0 ? '' : s < 60 ? `${Math.max(1, s)} 秒` : s < 3600 ? `${Math.floor(s / 60)} 分 ${String(s % 60).padStart(2, '0')} 秒` : `${Math.floor(s / 3600)} 小时 ${Math.floor(s % 3600 / 60)} 分`;
const idle: KernelProgress = { phase: 'idle', running: false, source: 'mirror', done: 0, total: 0, pct: 0, speed: 0, eta: -1, message: '' };

export function useKernelInstall(onDone?: (p: KernelProgress) => void) {
  const [p, setP] = useState<KernelProgress>(idle);
  useEffect(() => {
    let last = '';
    (api as any).KernelInstallStatus().then((x: KernelProgress) => { if (x) { setP(x); last = x.phase; } }).catch(() => {});
    return onEvent('kernel:install', (x: KernelProgress) => { setP(x); if (x.phase === 'done' && last !== 'done') onDone?.(x); last = x.phase; });
  }, []);
  return p;
}

const FF_STEPS: [KernelProgress['phase'], string][] = [['resolve', '获取地址'], ['download', '下载'], ['verify', '校验'], ['extract', '安装'], ['done', '完成']];
const Stepper: React.FC<{ p: KernelProgress; steps?: [KernelProgress['phase'], string][] }> = ({ p, steps = STEPS }) => {
  const failed = p.phase === 'error' || p.phase === 'cancelled';
  // On failure, mark the step that was running when it stopped.
  const [stopAt, setStopAt] = useState(0);
  const cur = steps.findIndex(s => s[0] === p.phase);
  useEffect(() => { if (cur >= 0) setStopAt(cur); }, [cur]);
  const at = failed ? stopAt : cur;
  return (
    <div className="ki-steps">
      {steps.map(([k, label], i) => {
        const st = p.phase === 'done' || i < at ? 'ok' : i === at ? (failed ? 'bad' : 'cur') : '';
        return <React.Fragment key={k}>
          {i > 0 && <i className={`ki-line ${i <= at || p.phase === 'done' ? 'on' : ''}`} />}
          <span className={`ki-step ${st}`}><b>{st === 'ok' ? <Icon.check s={10} /> : st === 'bad' ? <Icon.x s={10} /> : i + 1}</b>{label}</span>
        </React.Fragment>;
      })}
    </div>
  );
};

export const KernelProgressView: React.FC<{ p: KernelProgress; onCancel: () => void; onRetry: () => void; compact?: boolean; product?: string }> = ({ p, onCancel, onRetry, compact, product }) => {
  const fx = product === 'Firefox';
  const indet = p.running && (p.phase === 'resolve' || p.phase === 'verify' || p.total <= 0);
  const state = p.phase === 'error' ? 'bad' : p.phase === 'done' ? 'ok' : p.phase === 'cancelled' ? 'muted' : 'down';
  const pct = p.phase === 'done' ? 100 : Math.max(0, Math.min(100, p.pct || 0));
  const title = p.phase === 'done' ? `已安装 ${product || 'Chromium'} ${p.version || ''}` : p.phase === 'error' ? '安装失败' : p.phase === 'cancelled' ? '已取消' : p.message || '准备中…';
  return (
    <div className={`ki-prog ${state} ${compact ? 'compact' : ''}`}>
      {!compact && <Stepper p={p} steps={fx ? FF_STEPS : STEPS} />}
      <div className="ki-top">
        <span className="ki-ic">{p.phase === 'done' ? <Icon.check s={15} /> : p.phase === 'error' ? <Icon.alert s={15} /> : p.phase === 'cancelled' ? <Icon.x s={14} /> : <Icon.download s={15} />}</span>
        <div className="ki-title">
          <b title={title}>{title}</b>
          <span title={p.phase === 'error' ? p.error : undefined}>
            {p.phase === 'error' ? (p.message || p.error)
              : p.phase === 'done' ? (p.path || '')
              : p.phase === 'cancelled' ? (p.hint || '')
              : fx && p.version ? `Firefox ${p.version} · 官方安装包${p.resumed ? ' · 断点续传' : ''}` : p.revision ? `快照 r${p.revision}${p.resumed ? ' · 断点续传' : ''}` : '正在连接下载源…'}
          </span>
        </div>
        {(p.phase === 'download' || p.phase === 'extract') && p.total > 0 && <span className="ki-pct num">{pct < 10 ? pct.toFixed(1) : Math.floor(pct)}%</span>}
        {p.running
          ? <button className="btn sm ghost" onClick={onCancel}>取消</button>
          : (p.phase === 'error' || p.phase === 'cancelled') && <button className="btn sm" onClick={onRetry}><Icon.refresh s={13} /> {p.phase === 'cancelled' ? '继续安装' : '重试'}</button>}
      </div>
      <div className={`xbar ${state === 'muted' ? '' : state} ${indet ? 'indet' : ''}`}><i style={indet ? undefined : { width: `${pct}%` }} /></div>
      <div className="ki-meta">
        {p.phase === 'download' && p.total > 0
          ? <><span className="num">{fmtBytes(p.done)} / {fmtBytes(p.total)}</span><span>{p.speed > 0 ? <><b className="num">{fmtSpeed(p.speed)}</b>{p.eta > 0 ? ` · 剩余约 ${fmtEta(p.eta)}` : ''}</> : '正在测速…'}</span></>
          : fx && p.phase === 'extract' ? <><span>正在静默安装到程序数据目录…</span><span>请勿关闭程序</span></>
          : p.phase === 'extract' ? <><span>正在解压 <span className="num">{p.total > 0 ? `${fmtBytes(p.done)} / ${fmtBytes(p.total)}` : ''}</span></span><span>请勿关闭程序</span></>
          : p.phase === 'error' && p.hint ? <span className="ki-hint"><Icon.alert s={12} /> {p.hint}</span>
          : p.phase === 'done' && fx ? <span>已设为 Firefox 内核，Firefox 环境将用它启动</span>
          : p.phase === 'done' ? <span>已设为默认开源内核，手机 / 平板 / Mac / Linux 环境将用它启动</span>
          : p.running ? <span>{p.message}</span> : null}
      </div>
    </div>
  );
};

// Full installer for Settings → 开源内核.
export const KernelInstaller: React.FC<{ source?: string; url?: string; onInstalled?: (path: string) => void; compact?: boolean }> = ({ source, url, onInstalled, compact }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [src, setSrc] = useState<Source>((source as Source) || 'mirror');
  const [custom, setCustom] = useState(url || '');
  const [list, setList] = useState<InstalledKernel[]>([]);
  const refresh = () => (api as any).KernelList().then((x: InstalledKernel[]) => setList(x || [])).catch(() => {});
  const p = useKernelInstall(x => { refresh(); if (x.path) onInstalled?.(x.path); toast(`开源内核安装完成 · Chromium ${x.version || ''}`, 'ok'); });
  useEffect(() => { refresh(); if (!source) api.GetSettings().then((st: any) => { if (st?.kernelSource) setSrc(st.kernelSource); if (st?.kernelUrl) setCustom(st.kernelUrl); }).catch(() => {}); }, []);

  const start = async () => {
    if (src === 'custom' && !/^https?:\/\//i.test(custom.trim())) { toast('请填写以 http:// 或 https:// 开头的下载地址', 'bad'); return; }
    try { await (api as any).KernelInstallStart(src, custom.trim()); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const cancel = () => (api as any).KernelInstallCancel();
  const del = async (k: InstalledKernel) => {
    if (!await ask({ title: `删除 Chromium ${k.version}？`, body: `将删除 ${fmtBytes(k.size)} 的内核文件。${k.active ? '它是当前使用的开源内核，删除后会改为自动检测。' : ''}正在使用它的环境需要先关闭。`, ok: '删除', danger: true })) return;
    try { await (api as any).KernelDelete(k.dir); toast('已删除内核'); refresh(); if (k.active) onInstalled?.(''); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const show = p.phase !== 'idle';

  return (
    <div className={`ki ${compact ? 'compact' : ''}`}>
      {!p.running && <>
        {!compact && <div className="ki-src">
          <Segmented<Source> className="compact" value={src} onChange={v => setSrc(v)} items={[['mirror', '国内镜像'], ['official', '官方源'], ['custom', '自定义地址']]} />
          <span className="sub" style={{ margin: 0 }}>{src === 'mirror' ? 'npmmirror 同步的 Chromium 官方快照，国内推荐' : src === 'official' ? 'Google 官方快照服务器，国内可能无法访问' : '快照目录或 .zip 直链，如 ungoogled-chromium 便携版'}</span>
        </div>}
        {!compact && src === 'custom' && <input type="text" className="mono" style={{ marginTop: 10 }} value={custom} onChange={e => setCustom(e.target.value)} placeholder="https://registry.npmmirror.com/-/binary/chromium-browser-snapshots/Win_x64/" />}
      </>}
      {show
        ? <KernelProgressView p={p} onCancel={cancel} onRetry={start} compact={compact} />
        : <div className="ki-cta">
          <div className="ki-cta-ic"><Icon.download s={18} /></div>
          <div className="ki-cta-txt"><b>一键安装开源 Chromium</b><span>{src === 'custom' ? '从自定义地址下载 .zip，自动解压并校验 chrome.exe · 支持断点续传' : 'Windows x64 官方快照，约 360 MB，解压后约 860 MB · 支持断点续传'}</span></div>
          <button className="btn primary" onClick={start}><Icon.download s={14} /> 一键安装</button>
        </div>}
      {show && !p.running && p.phase !== 'error' && p.phase !== 'cancelled' && !compact && <div className="ki-again"><button className="btn sm ghost" onClick={start}><Icon.refresh s={13} /> 安装最新版本</button></div>}
      {!compact && list.length > 0 && <div className="ki-list">
        <div className="ki-list-h">已安装的内核 <span className="num">{list.length}</span></div>
        {list.map(k => <div key={k.dir} className={`ki-row ${k.active ? 'on' : ''}`}>
          <span className="ki-ver"><Icon.cpu s={14} /></span>
          <div className="ki-row-main"><b className="num">Chromium {k.version}</b><span className="mono" title={k.path}>{k.path}</span></div>
          <span className="ki-size num">{fmtBytes(k.size)}</span>
          {k.active ? <span className="badge ok"><span className="d" />使用中</span> : <button className="btn sm" onClick={() => onInstalled?.(k.path)}>使用</button>}
          <button className="btn sm ghost icon-only" title="删除" aria-label="删除" disabled={p.running} onClick={() => del(k)}><Icon.trash s={14} /></button>
        </div>)}
      </div>}
    </div>
  );
};

// Shown where a chromium-kernel profile has no kernel to launch with.
export const KernelMissingNotice: React.FC<{ onInstalled?: () => void }> = ({ onInstalled }) => {
  const [missing, setMissing] = useState(false);
  const check = () => api.ChromiumPath().then(x => setMissing(!x)).catch(() => {});
  useEffect(() => { check(); }, []);
  if (!missing) return null;
  return (
    <div className="notice warn ki-notice">
      <div><b>没有找到开源内核</b><span>这个环境需要 Chromium 才能启动，可以一键下载安装（国内镜像）。</span></div>
      <KernelInstaller compact onInstalled={() => { check(); onInstalled?.(); }} />
    </div>
  );
};

// Opened when launching a chromium-kernel profile fails because no kernel exists.
export const KernelMissingModal: React.FC<{ name: string; onClose: () => void; onLaunch: () => void }> = ({ name, onClose, onLaunch }) => {
  const [out, close] = useExit(onClose);
  const [ready, setReady] = useState(false);
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 600 }}>
        <div className="dhead"><div className="glyph"><Icon.cpu s={18} /></div><div><h3>需要开源内核</h3><div className="sub ki-ell">「{name}」使用开源内核（Chromium）启动，本机还没有安装</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody"><KernelInstaller onInstalled={p => setReady(!!p)} /></div>
        <div className="dfoot"><span className="sub" style={{ margin: 0 }}>也可以在「设置 → 开源内核」手动选择 chrome.exe</span><div className="spacer" /><button className="btn" onClick={() => close()}>{ready ? '关闭' : '稍后再说'}</button><button className="btn primary" disabled={!ready} onClick={() => { onLaunch(); close(); }}><Icon.play s={14} /> 立即启动</button></div>
      </div>
    </div>
  );
};
