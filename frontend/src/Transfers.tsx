import React, { useEffect, useLayoutEffect, useRef, useState, useSyncExternalStore } from 'react';
import { api, onEvent } from './api';
import { Icon } from './ui';

export type Transfer = {
  id: string; name: string; dir: 'up' | 'down';
  phase: 'scan' | 'pack' | 'upload' | 'commit' | 'download' | 'write' | 'done' | 'error';
  done: number; total: number; speed: number; eta: number; saved: number; started: number; error?: string; final: boolean;
};

// ---- tiny shared store: the dock, the profile list and the cloud list all read it ----
type Entry = Transfer & { hist: number[]; seen: number };
let items: Record<string, Entry> = {};
const subs = new Set<() => void>();
const emit = () => { items = { ...items }; subs.forEach(f => f()); };
const timers: Record<string, any> = {};

function put(t: Transfer) {
  const prev = items[t.id];
  const fresh = !prev || prev.final || t.started !== prev.started;
  const hist = fresh ? [] : prev.hist.slice(-39);
  if (!t.final && (t.phase === 'upload' || t.phase === 'download')) hist.push(t.speed || 0);
  items[t.id] = { ...t, hist, seen: Date.now() };
  clearTimeout(timers[t.id]);
  if (t.final) timers[t.id] = setTimeout(() => dismiss(t.id), t.phase === 'error' ? 9000 : 4200);
  emit();
}
export function dismiss(id: string) { clearTimeout(timers[id]); delete items[id]; emit(); }

let wired = false;
function wire() {
  if (wired) return; wired = true;
  onEvent('sync:progress', (t: Transfer) => put(t));
  (api as any).Transfers?.().then((l: Transfer[]) => (l || []).forEach(put)).catch(() => {});
}
export function useTransfers(): Record<string, Entry> {
  wire();
  return useSyncExternalStore(f => { subs.add(f); return () => { subs.delete(f); }; }, () => items);
}
export const useTransfer = (id: string) => useTransfers()[id];

// ---- formatting ----
export const fmtBytes = (b: number) => b >= 1 << 30 ? `${(b / (1 << 30)).toFixed(2)} GB` : b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : b >= 1024 ? `${Math.round(b / 1024)} KB` : `${Math.max(0, Math.round(b))} B`;
export const fmtSpeed = (b: number) => b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB/s` : `${Math.max(0, Math.round(b / 1024))} KB/s`;
const fmtEta = (s: number) => s < 0 ? '' : s < 60 ? `${s}秒` : s < 3600 ? `${Math.floor(s / 60)}分${String(s % 60).padStart(2, '0')}秒` : `${Math.floor(s / 3600)}时${Math.floor(s % 3600 / 60)}分`;
const PHASE: Record<string, [string, string]> = {
  scan: ['正在扫描文件', '比对哪些数据需要上传'],
  pack: ['正在打包', '压缩并加密环境数据'],
  upload: ['上传中', ''],
  commit: ['正在提交', '写入新版本'],
  download: ['下载中', ''],
  write: ['正在还原', '把数据写回环境目录'],
  done: ['完成', ''],
  error: ['失败', ''],
};
export const pct = (t: Transfer) => t.total > 0 ? Math.min(100, (t.done / t.total) * 100) : 0;

// ---- pieces ----
const Spark: React.FC<{ v: number[]; dir: string }> = ({ v, dir }) => {
  if (v.length < 2) return <svg className="spark" viewBox="0 0 64 20" />;
  const max = Math.max(...v, 1);
  const pts = v.map((x, i) => [(i / (v.length - 1)) * 64, 18 - (x / max) * 16]);
  const d = pts.map((p, i) => `${i ? 'L' : 'M'}${p[0].toFixed(1)} ${p[1].toFixed(1)}`).join(' ');
  const g = `sg-${dir}`;
  return (
    <svg className="spark" viewBox="0 0 64 20" preserveAspectRatio="none">
      <defs><linearGradient id={g} x1="0" x2="0" y1="0" y2="1"><stop offset="0" stopColor="currentColor" stopOpacity=".35" /><stop offset="1" stopColor="currentColor" stopOpacity="0" /></linearGradient></defs>
      <path d={`${d} L64 20 L0 20 Z`} fill={`url(#${g})`} />
      <path d={d} fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinejoin="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
};

export const Bar: React.FC<{ t: Transfer; thin?: boolean }> = ({ t, thin }) => {
  const indet = !t.final && (t.total <= 0 || t.phase === 'scan' || t.phase === 'pack' || t.phase === 'commit');
  const state = t.phase === 'error' ? 'bad' : t.phase === 'done' ? 'ok' : t.dir;
  return (
    <div className={`xbar ${state} ${indet ? 'indet' : ''} ${thin ? 'thin' : ''}`}>
      <i style={indet ? undefined : { width: `${t.final && t.phase === 'done' ? 100 : pct(t)}%` }} />
    </div>
  );
};

const Row: React.FC<{ t: Entry }> = ({ t }) => {
  const [label, hint] = PHASE[t.phase] || [t.phase, ''];
  const moving = t.phase === 'upload' || t.phase === 'download' || t.phase === 'write';
  const p = pct(t);
  const elapsed = Math.max(1, Math.round(((t.final ? t.seen : Date.now()) - t.started) / 1000));
  return (
    <div className={`xrow ${t.dir} ${t.phase}`}>
      <div className="xic">
        {t.phase === 'done' ? <Icon.check s={15} /> : t.phase === 'error' ? <Icon.alert s={15} /> : t.dir === 'up' ? <Icon.upload s={15} /> : <Icon.download s={15} />}
      </div>
      <div className="xmain">
        <div className="xtop">
          <span className="xname" title={t.name}>{t.name}</span>
          <span className="xphase">{t.phase === 'upload' || t.phase === 'download' ? label : `${t.dir === 'up' ? '上传' : '拉取'} · ${label}`}</span>
          {t.saved > 0 && t.phase !== 'error' && <span className="xsave" title="对方已有这些数据，增量同步不再传输">省 {fmtBytes(t.saved)}</span>}
          <span className="xpct num">{t.phase === 'done' ? '100%' : moving && t.total > 0 ? `${p < 10 ? p.toFixed(1) : Math.floor(p)}%` : ''}</span>
          {t.final && <button className="xx" onClick={() => dismiss(t.id)} aria-label="关闭"><Icon.x s={12} /></button>}
        </div>
        <Bar t={t} />
        <div className="xmeta">
          {t.phase === 'error'
            ? <span className="xerr" title={t.error}>{t.error}</span>
            : t.phase === 'done'
              ? <span className="num">用时 {fmtEta(elapsed)}{t.speed > 0 ? ` · 平均 ${fmtSpeed(t.speed)}` : ''}</span>
              : moving && t.total > 0
                ? <span className="num">{fmtBytes(t.done)} / {fmtBytes(t.total)}{t.phase !== 'write' && t.speed > 0 ? <> · <b>{fmtSpeed(t.speed)}</b></> : null}{t.eta > 0 ? ` · 约 ${fmtEta(t.eta)}` : ''}</span>
                : <span>{hint || '准备中'}</span>}
          {t.phase !== 'error' && <Spark v={t.hist} dir={t.dir} />}
        </div>
      </div>
    </div>
  );
};

// Bottom-right dock listing every upload/download, newest last.
export const TransferDock: React.FC = () => {
  const all = Object.values(useTransfers()).sort((a, b) => Number(a.final) - Number(b.final) || a.started - b.started);
  const [open, setOpen] = useState(true);
  const ref = useRef<HTMLDivElement>(null);
  const [, tick] = useState(0);
  const live = all.filter(t => !t.final);
  useEffect(() => { if (!live.length) return; const i = setInterval(() => tick(x => x + 1), 1000); return () => clearInterval(i); }, [live.length]);
  // Keep toasts above the dock.
  useLayoutEffect(() => {
    const el = ref.current;
    const set = () => document.documentElement.style.setProperty('--dock-h', el && all.length ? `${el.offsetHeight + 12}px` : '0px');
    set();
    if (!el) return;
    const ro = new ResizeObserver(set); ro.observe(el);
    return () => { ro.disconnect(); };
  });
  if (!all.length) return null;
  const speedUp = live.filter(t => t.dir === 'up').reduce((s, t) => s + (t.phase === 'upload' ? t.speed : 0), 0);
  const speedDown = live.filter(t => t.dir === 'down').reduce((s, t) => s + (t.phase === 'download' ? t.speed : 0), 0);
  const totalAll = live.reduce((s, t) => s + (t.total > 0 ? t.total : 0), 0);
  const doneAll = live.reduce((s, t) => s + (t.total > 0 ? t.done : 0), 0);
  return (
    <div className={`xdock ${open ? '' : 'min'}`} ref={ref}>
      <button className="xhead" onClick={() => setOpen(o => !o)}>
        <span className={`xorb ${live.length ? 'on' : ''}`}><Icon.cloud s={14} /></span>
        <span className="xtitle">{live.length ? `正在同步 ${live.length} 项` : '同步完成'}</span>
        <span className="xrates num">
          {speedUp > 0 && <span className="up"><Icon.upload s={11} />{fmtSpeed(speedUp)}</span>}
          {speedDown > 0 && <span className="down"><Icon.download s={11} />{fmtSpeed(speedDown)}</span>}
        </span>
        <span className="xchev">{open ? '收起' : '展开'}</span>
      </button>
      {!open && live.length > 0 && totalAll > 0 && <div className="xsum"><Bar t={{ ...live[0], phase: 'upload', done: doneAll, total: totalAll, final: false, dir: live[0].dir }} thin /></div>}
      {open && <div className="xlist">{all.map(t => <Row key={t.id} t={t} />)}</div>}
    </div>
  );
};

// Inline status for a table row: replaces the "同步中" badge with a live bar.
export const InlineTransfer: React.FC<{ id: string; fallback: React.ReactNode }> = ({ id, fallback }) => {
  const t = useTransfer(id);
  if (!t || t.final) return <>{fallback}</>;
  const moving = (t.phase === 'upload' || t.phase === 'download') && t.total > 0;
  return (
    <span className={`xinline ${t.dir}`} title={`${t.dir === 'up' ? '上传' : '拉取'} · ${PHASE[t.phase]?.[0] || ''}`}>
      {t.dir === 'up' ? <Icon.upload s={11} /> : <Icon.download s={11} />}
      <span className="num">{moving ? `${Math.floor(pct(t))}%` : PHASE[t.phase]?.[0].replace('正在', '')}</span>
      <Bar t={t} thin />
    </span>
  );
};
