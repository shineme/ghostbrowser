import React, { createContext, useContext, useState, useCallback, useEffect, useLayoutEffect, useRef } from 'react';
import { createPortal } from 'react-dom';

// ---- icons (lucide-style, stroke) ----
type IP = { s?: number };
const S = (p: any): React.SVGProps<SVGSVGElement> => ({ xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const, width: p.s || 18, height: p.s || 18 });
export const Icon = {
  grid: (p: IP) => <svg {...S(p)}><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /></svg>,
  layers: (p: IP) => <svg {...S(p)}><path d="M12 2 2 7l10 5 10-5-10-5Z" /><path d="m2 17 10 5 10-5" /><path d="m2 12 10 5 10-5" /></svg>,
  globe: (p: IP) => <svg {...S(p)}><circle cx="12" cy="12" r="10" /><path d="M2 12h20" /><path d="M12 2a15 15 0 0 1 0 20 15 15 0 0 1 0-20Z" /></svg>,
  cloud: (p: IP) => <svg {...S(p)}><path d="M17.5 19a4.5 4.5 0 0 0 .5-9 6 6 0 0 0-11.5 1.5A4 4 0 0 0 6 19Z" /></svg>,
  settings: (p: IP) => <svg {...S(p)}><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.6 1.6 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.6 1.6 0 0 0-2.7 1.1V21a2 2 0 1 1-4 0v-.1A1.6 1.6 0 0 0 9 19.4a1.6 1.6 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.6 1.6 0 0 0-1.1-2.7H3a2 2 0 1 1 0-4h.1A1.6 1.6 0 0 0 4.6 9a1.6 1.6 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.6 1.6 0 0 0 1.8.3H9a1.6 1.6 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.6 1.6 0 0 0 2.7 1.1 1.6 1.6 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.6 1.6 0 0 0-.3 1.8V9a1.6 1.6 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.6 1.6 0 0 0-1.5 1Z" /></svg>,
  play: (p: IP) => <svg {...S(p)}><polygon points="6 3 20 12 6 21 6 3" /></svg>,
  stop: (p: IP) => <svg {...S(p)}><rect x="5" y="5" width="14" height="14" rx="2" /></svg>,
  plus: (p: IP) => <svg {...S(p)}><path d="M12 5v14M5 12h14" /></svg>,
  search: (p: IP) => <svg {...S(p)}><circle cx="11" cy="11" r="8" /><path d="m21 21-4.3-4.3" /></svg>,
  trash: (p: IP) => <svg {...S(p)}><path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m2 0v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" /></svg>,
  copy: (p: IP) => <svg {...S(p)}><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></svg>,
  edit: (p: IP) => <svg {...S(p)}><path d="M12 20h9" /><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" /></svg>,
  refresh: (p: IP) => <svg {...S(p)}><path d="M3 12a9 9 0 0 1 15-6.7L21 8" /><path d="M21 3v5h-5" /><path d="M21 12a9 9 0 0 1-15 6.7L3 16" /><path d="M3 21v-5h5" /></svg>,
  check: (p: IP) => <svg {...S(p)}><path d="M20 6 9 17l-5-5" /></svg>,
  x: (p: IP) => <svg {...S(p)}><path d="M18 6 6 18M6 6l12 12" /></svg>,
  dice: (p: IP) => <svg {...S(p)}><rect x="3" y="3" width="18" height="18" rx="3" /><circle cx="8.5" cy="8.5" r="1.3" fill="currentColor" /><circle cx="15.5" cy="8.5" r="1.3" fill="currentColor" /><circle cx="12" cy="12" r="1.3" fill="currentColor" /><circle cx="8.5" cy="15.5" r="1.3" fill="currentColor" /><circle cx="15.5" cy="15.5" r="1.3" fill="currentColor" /></svg>,
  upload: (p: IP) => <svg {...S(p)}><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><path d="M17 8l-5-5-5 5" /><path d="M12 3v12" /></svg>,
  download: (p: IP) => <svg {...S(p)}><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><path d="M7 10l5 5 5-5" /><path d="M12 15V3" /></svg>,
  folder: (p: IP) => <svg {...S(p)}><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" /></svg>,
  shield: (p: IP) => <svg {...S(p)}><path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67 0C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1Z" /></svg>,
  cpu: (p: IP) => <svg {...S(p)}><rect x="4" y="4" width="16" height="16" rx="2" /><rect x="9" y="9" width="6" height="6" /><path d="M9 2v2M15 2v2M9 20v2M15 20v2M2 9h2M2 15h2M20 9h2M20 15h2" /></svg>,
  terminal: (p: IP) => <svg {...S(p)}><path d="m4 17 6-6-6-6" /><path d="M12 19h8" /></svg>,
  console: (p: IP) => <svg {...S(p)}><rect x="3" y="4" width="18" height="16" rx="3" /><path d="m7.5 10 3 2.5-3 2.5" /><path d="M13 15.5h4" /></svg>,
  android: (p: IP) => <svg {...S(p)}><path d="M5 16V10a7 7 0 0 1 14 0v6Z" /><path d="M5 13h14M8 3.5l1.5 2.5M16 3.5 14.5 6" /><path d="M9.5 9h.01M14.5 9h.01" /></svg>,
  chromeos: (p: IP) => <svg {...S(p)}><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="4" /><path d="M12 8h9M8.5 14 4 6.3M15.5 14 11 21.8" /></svg>,
  monitor: (p: IP) => <svg {...S(p)}><rect x="2" y="3" width="20" height="14" rx="2" /><path d="M8 21h8M12 17v4" /></svg>,
  phone: (p: IP) => <svg {...S(p)}><rect x="6.5" y="2" width="11" height="20" rx="2.5" /><path d="M11 18.5h2" /></svg>,
  tablet: (p: IP) => <svg {...S(p)}><rect x="4" y="2" width="16" height="20" rx="2.5" /><path d="M11 18.5h2" /></svg>,
  fold: (p: IP) => <svg {...S(p)}><rect x="3" y="4" width="18" height="16" rx="2.5" /><path d="M12 4v16" strokeDasharray="2 2" /></svg>,
  chromebook: (p: IP) => <svg {...S(p)}><rect x="4" y="4" width="16" height="11" rx="1.5" /><path d="M2 19h20l-2-4H4Z" /><circle cx="12" cy="9.5" r="2.5" /></svg>,
  tv: (p: IP) => <svg {...S(p)}><rect x="2" y="5" width="20" height="13" rx="2" /><path d="M7 21h10M9 2l3 3 3-3" /></svg>,
  windows: (p: IP) => <svg {...S(p)} fill="currentColor" stroke="none"><path d="M3 5.5 10.5 4.4v7.1H3ZM11.5 4.3 21 3v8.5h-9.5ZM3 12.5h7.5v7.1L3 18.5ZM11.5 12.5H21V21l-9.5-1.3Z" /></svg>,
  apple: (p: IP) => <svg {...S(p)} fill="currentColor" stroke="none"><path d="M16.4 12.6c0-2.4 2-3.6 2.1-3.6-1.1-1.7-2.9-1.9-3.5-1.9-1.5-.2-2.9.9-3.7.9-.8 0-1.9-.9-3.2-.8-1.6 0-3.2 1-4 2.4-1.7 3-.4 7.4 1.2 9.8.8 1.2 1.8 2.5 3 2.4 1.2 0 1.7-.8 3.1-.8 1.5 0 1.9.8 3.2.8 1.3 0 2.2-1.2 3-2.4.9-1.4 1.3-2.7 1.3-2.8-.1 0-2.5-1-2.5-4ZM14 5.4c.7-.8 1.1-1.9 1-3-1 0-2.1.7-2.8 1.5-.6.7-1.2 1.8-1 2.9 1 .1 2.1-.6 2.8-1.4Z" /></svg>,
  linux: (p: IP) => <svg {...S(p)}><path d="M12 3c-2 0-3 1.7-3 4 0 1.6-.5 2.6-1.6 4.2C6 13.3 5 15 5.5 17c.4 1.6 2 2 3.5 2.6.9.4 1.8 1.4 3 1.4s2.1-1 3-1.4c1.5-.6 3.1-1 3.5-2.6.5-2-.5-3.7-1.9-5.8C15.5 9.6 15 8.6 15 7c0-2.3-1-4-3-4Z" /><path d="M10.5 8h.01M13.5 8h.01" /></svg>,
  bolt: (p: IP) => <svg {...S(p)}><path d="M13 2 4 14h7l-1 8 9-12h-7Z" /></svg>,
  server: (p: IP) => <svg {...S(p)}><rect x="3" y="3" width="18" height="7" rx="2" /><rect x="3" y="14" width="18" height="7" rx="2" /><path d="M7 6.5h.01M7 17.5h.01" /></svg>,
  lock: (p: IP) => <svg {...S(p)}><rect x="4" y="11" width="16" height="10" rx="2" /><path d="M8 11V7a4 4 0 0 1 8 0v4" /></svg>,
  laptop: (p: IP) => <svg {...S(p)}><rect x="4" y="4" width="16" height="11" rx="2" /><path d="M2 19h20" /></svg>,
  sparkles: (p: IP) => <svg {...S(p)}><path d="M12 3l1.8 4.6L18.5 9l-4.7 1.4L12 15l-1.8-4.6L5.5 9l4.7-1.4Z" /><path d="M19 15l.8 2 2.2.8-2.2.8-.8 2-.8-2-2.2-.8 2.2-.8Z" /></svg>,
  fingerprint: (p: IP) => <svg {...S(p)}><path d="M12 10a2 2 0 0 0-2 2c0 1.02-.1 2.51-.26 4" /><path d="M14 13.12c0 2.38 0 6.38-1 8.88" /><path d="M17.29 21.02c.12-.6.43-2.3.5-3.02" /><path d="M2 12a10 10 0 0 1 18-6" /><path d="M2 16h.01" /><path d="M21.8 16c.2-2 .131-5.354 0-6" /><path d="M5 19.5C5.5 18 6 15 6 12a6 6 0 0 1 .34-2" /><path d="M8.65 22c.21-.66.45-1.32.57-2" /><path d="M9 6.8a6 6 0 0 1 9 5.2v2" /></svg>,
  alert: (p: IP) => <svg {...S(p)}><path d="M12 9v4M12 17h.01" /><path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" /></svg>,
  sidebar: (p: IP) => <svg {...S(p)}><rect x="3" y="4" width="18" height="16" rx="3" /><path d="M9 4v16" /></svg>,
  ghost: (p: IP) => <svg {...S(p)}><path d="M9 10h.01M15 10h.01M12 2a8 8 0 0 0-8 8v12l3-3 2.5 2.5L12 21l2.5 2.5L17 19l3 3V10a8 8 0 0 0-8-8Z" /></svg>,
  star: (p: IP) => <svg {...S(p)}><path d="M11.5 2.8a.56.56 0 0 1 1 0l2.3 4.7a2.1 2.1 0 0 0 1.6 1.2l5.2.8a.56.56 0 0 1 .3 1l-3.8 3.6a2.1 2.1 0 0 0-.6 1.9l.9 5.2a.56.56 0 0 1-.8.6l-4.6-2.5a2.1 2.1 0 0 0-2 0l-4.6 2.5a.56.56 0 0 1-.8-.6l.9-5.2a2.1 2.1 0 0 0-.6-1.9L1.9 10.5a.56.56 0 0 1 .3-1l5.2-.8a2.1 2.1 0 0 0 1.6-1.2Z" /></svg>,
  link: (p: IP) => <svg {...S(p)}><path d="M10 13a5 5 0 0 0 7.5.5l3-3a5 5 0 0 0-7-7l-1.7 1.7" /><path d="M14 11a5 5 0 0 0-7.5-.5l-3 3a5 5 0 0 0 7 7l1.7-1.7" /></svg>,
  more: (p: IP) => <svg {...S(p)}><circle cx="5" cy="12" r="1.3" /><circle cx="12" cy="12" r="1.3" /><circle cx="19" cy="12" r="1.3" /></svg>,
};

// ---- toast ----
export type ToastAction = { label: string; onClick: () => void };
type Toast = { id: number; msg: string; kind: string; out?: boolean; action?: ToastAction; life: number };
const ToastCtx = createContext<(msg: string, kind?: string, action?: ToastAction) => void>(() => {});
export const useToast = () => useContext(ToastCtx);

// ---- confirm (replaces window.confirm, which looks foreign in a designed app) ----
type ConfirmOpts = { title: string; body?: React.ReactNode; ok?: string; danger?: boolean };
const ConfirmCtx = createContext<(o: ConfirmOpts) => Promise<boolean>>(async () => false);
export const useConfirm = () => useContext(ConfirmCtx);

export const ToastHost: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [list, setList] = useState<Toast[]>([]);
  const [ask, setAsk] = useState<(ConfirmOpts & { resolve: (v: boolean) => void }) | null>(null);
  const push = useCallback((msg: string, kind = 'ok', action?: ToastAction) => {
    const id = Date.now() + Math.random();
    const life = action ? 7000 : 3200; // a toast with a button stays long enough to use it
    setList(l => [...l.slice(-3), { id, msg, kind, action, life }]);
    setTimeout(() => setList(l => l.map(t => t.id === id ? { ...t, out: true } : t)), life);
    setTimeout(() => setList(l => l.filter(t => t.id !== id)), life + 280);
  }, []);
  const dismiss = (id: number) => { setList(l => l.map(t => t.id === id ? { ...t, out: true } : t)); setTimeout(() => setList(l => l.filter(t => t.id !== id)), 280); };
  const confirm = useCallback((o: ConfirmOpts) => new Promise<boolean>(resolve => setAsk({ ...o, resolve })), []);
  return (
    <ToastCtx.Provider value={push}>
      <ConfirmCtx.Provider value={confirm}>
        {children}
        {ask && <ConfirmDialog o={ask} done={v => { ask.resolve(v); setAsk(null); }} />}
        <div className="toasts">{list.map(t =>
          <div key={t.id} className={`toast ${t.kind} ${t.out ? 'out' : ''} ${t.action ? 'has-act' : ''}`}>
            <span className="ti">{t.kind === 'bad' ? <Icon.alert s={14} /> : t.kind === 'star' ? <Icon.star s={14} /> : <Icon.check s={14} />}</span>
            <span className="tm">{t.msg}</span>
            {t.action && <button className="toast-act" onClick={() => { t.action!.onClick(); dismiss(t.id); }}>{t.action.label}</button>}
            <i className="life" style={{ animationDuration: `${t.life}ms` }} />
          </div>)}
        </div>
      </ConfirmCtx.Provider>
    </ToastCtx.Provider>
  );
};

const ConfirmDialog: React.FC<{ o: ConfirmOpts; done: (v: boolean) => void }> = ({ o, done }) => {
  const [out, close] = useExit(() => {});
  const finish = (v: boolean) => close(() => done(v));
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') finish(false); if (e.key === 'Enter') finish(true); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && finish(false)}>
      <div className="modal sm">
        <div className="dhead">
          <div className={`glyph ${o.danger ? 'bad' : ''}`}>{o.danger ? <Icon.trash s={18} /> : <Icon.alert s={18} />}</div>
          <div><h3>{o.title}</h3></div>
        </div>
        {o.body && <div className="dbody"><p>{o.body}</p></div>}
        <div className="dfoot"><div className="spacer" />
          <button className="btn" onClick={() => finish(false)}>取消</button>
          <button className={`btn ${o.danger ? 'solid-bad' : 'primary'}`} onClick={() => finish(true)} autoFocus>{o.ok || '确定'}</button>
        </div>
      </div>
    </div>
  );
};

/** Plays the exit animation (adds .out) before actually unmounting. */
export const useExit = (onClose: () => void, ms = 210): [boolean, (after?: () => void) => void] => {
  const [out, setOut] = useState(false);
  const close = useCallback((after?: () => void) => {
    setOut(true);
    setTimeout(() => (after || onClose)(), ms);
  }, [onClose, ms]);
  return [out, close];
};

export const Toggle: React.FC<{ on: boolean; onChange: (v: boolean) => void; label?: string; hint?: string }> = ({ on, onChange, label, hint }) => (
  <div className="toggle" onClick={() => onChange(!on)}>
    <span>{label}{hint && <small>{hint}</small>}</span>
    <div className={`sw ${on ? 'on' : ''}`} />
  </div>
);

export const Field: React.FC<{ label?: string; hint?: string; children: React.ReactNode }> = ({ label, hint, children }) => (
  <div className="field">{label && <label>{label}</label>}{children}{hint && <span className="hint">{hint}</span>}</div>
);

/** Segmented control with a thumb that glides to the active item. */
export function Segmented<T extends string>({ items, value, onChange, className = '' }: { items: [T, string, React.ReactNode?][]; value: T; onChange: (v: T) => void; className?: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const [thumb, setThumb] = useState({ x: 0, w: 0 });
  useLayoutEffect(() => {
    const el = ref.current?.querySelector<HTMLButtonElement>(`button[data-v="${value}"]`);
    if (el) setThumb({ x: el.offsetLeft, w: el.offsetWidth });
  }, [value, items.length]);
  return (
    <div className={`seg ${className}`} ref={ref}>
      <span className="thumb" style={{ width: thumb.w, transform: `translateX(${thumb.x}px)` }} />
      {items.map(([v, label, ic]) => <button key={v} data-v={v} className={v === value ? 'active' : ''} onClick={() => onChange(v)}>{ic}{label}</button>)}
    </div>
  );
}

/** Compact filter pill with a popover list: replaces native <select> in toolbars. */
export function Pick<T extends string>({ label, value, options, onChange, icon, align = 'left' }: {
  label: string; value: T; options: [T, string, number?][]; onChange: (v: T) => void; icon?: React.ReactNode; align?: 'left' | 'right';
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setOpen(false); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key); };
  }, [open]);
  const cur = options.find(o => o[0] === value);
  const active = options.length > 0 && value !== options[0][0];
  return (
    <div className={`pick ${open ? 'open' : ''} ${active ? 'active' : ''}`} ref={ref}>
      <button type="button" className="pick-btn" onClick={() => setOpen(o => !o)}>
        {icon}<span className="pick-l">{label}</span>{active && <span className="pick-v">{cur?.[1]}</span>}
        <svg className="pick-c" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg>
      </button>
      {open && <div className={`pick-menu ${align}`}>
        {options.map(([v, l, n]) => <button type="button" key={v} className={v === value ? 'on' : ''} onClick={() => { onChange(v); setOpen(false); }}>
          <span className="pick-tick">{v === value && <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>}</span>
          <span className="pick-o">{l}</span>{n != null && <span className="pick-n num">{n}</span>}
        </button>)}
      </div>}
    </div>
  );
}

/** Row "更多" menu: a ⋯ icon button with a Pick-style popover of actions.
 * Rendered in a portal with fixed position so table panels never clip it. */
export type MoreItem = { label: string; icon?: React.ReactNode; onClick: () => void; danger?: boolean; disabled?: boolean; hint?: string } | 'sep' | null | false;
export const MoreMenu: React.FC<{ items: MoreItem[]; title?: string }> = ({ items, title = '更多' }) => {
  const [pos, setPos] = useState<{ top?: number; bottom?: number; right: number; up: boolean } | null>(null);
  const btn = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!pos) return;
    const down = (e: MouseEvent) => { if (!menu.current?.contains(e.target as Node) && !btn.current?.contains(e.target as Node)) setPos(null); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') setPos(null); };
    const close = () => setPos(null);
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key); window.addEventListener('resize', close); window.addEventListener('scroll', close, true);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key); window.removeEventListener('resize', close); window.removeEventListener('scroll', close, true); };
  }, [pos]);
  const toggle = () => {
    if (pos) return setPos(null);
    const r = btn.current!.getBoundingClientRect();
    const up = window.innerHeight - r.bottom < 220;
    setPos(up ? { bottom: window.innerHeight - r.top + 6, right: window.innerWidth - r.right, up } : { top: r.bottom + 6, right: window.innerWidth - r.right, up });
  };
  const list = items.filter(Boolean) as Exclude<MoreItem, null | false>[];
  return <>
    <button ref={btn} type="button" className={`btn ghost icon more-btn ${pos ? 'on' : ''}`} title={title} aria-label={title} onClick={toggle}><Icon.more s={15} /></button>
    {pos && createPortal(<div ref={menu} className={`pick-menu more-menu ${pos.up ? 'up' : ''}`} style={{ position: 'fixed', top: pos.top ?? 'auto', bottom: pos.bottom ?? 'auto', right: pos.right, left: 'auto' }}>
      {list.map((it, i) => it === 'sep' ? <div key={i} className="more-sep" /> :
        <button type="button" key={i} className={it.danger ? 'danger' : ''} disabled={it.disabled} title={it.hint} onClick={() => { setPos(null); it.onClick(); }}>
          <span className="more-ic">{it.icon}</span><span className="pick-o">{it.label}</span>
        </button>)}
    </div>, document.body)}
  </>;
};

/** Animated number that eases from its previous value. */
/** Full-width form dropdown in the Pick style (replaces native <select> in forms). */
export function SelectField<T extends string>({ value, options, onChange, disabled }: { value: T; options: [T, string][]; onChange: (v: T) => void; disabled?: boolean }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setOpen(false); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key); };
  }, [open]);
  const cur = options.find(o => o[0] === value);
  return (
    <div className={`pick pickfield selfield ${open ? 'open' : ''} ${disabled ? 'disabled' : ''}`} ref={ref}>
      <button type="button" className="pick-btn" disabled={disabled} onClick={() => setOpen(o => !o)}><span className="pick-o">{cur ? cur[1] : value}</span>
        <svg className="pick-c" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg></button>
      {open && <div className="pick-menu left">{options.map(([v, l]) => <button type="button" key={v} className={v === value ? 'on' : ''} title={l} onClick={() => { onChange(v); setOpen(false); }}>
        <span className="pick-tick">{v === value && <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>}</span><span className="pick-o">{l}</span></button>)}</div>}
    </div>
  );
}

export const CountUp: React.FC<{ n: number; ms?: number }> = ({ n, ms = 900 }) => {
  const [v, setV] = useState(0);
  const from = useRef(0);
  useEffect(() => {
    const a = from.current, t0 = performance.now(); let raf = 0;
    const tick = (t: number) => {
      const k = Math.min(1, (t - t0) / ms); const e = 1 - Math.pow(1 - k, 4);
      setV(Math.round(a + (n - a) * e));
      if (k < 1) raf = requestAnimationFrame(tick); else from.current = n;
    };
    raf = requestAnimationFrame(tick); return () => cancelAnimationFrame(raf);
  }, [n, ms]);
  return <span className="num">{v}</span>;
};

/** Each profile gets a stable gradient identity derived from its seed. */
export const avatarStyle = (seed: number | string): React.CSSProperties => {
  const s = String(seed); let h = 0; for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0;
  const P = [['#7c8cff', '#b07cff'], ['#4fe0d0', '#3b82f6'], ['#ff8a65', '#ff4d8d'], ['#34d399', '#0ea5a4'], ['#fbbf24', '#f97316'], ['#a78bfa', '#ec4899'], ['#60a5fa', '#818cf8'], ['#f472b6', '#c084fc'], ['#2dd4bf', '#a3e635']];
  const [a, b] = P[h % P.length];
  return { ['--av' as any]: `linear-gradient(135deg, ${a}, ${b})`, ['--av-glow' as any]: a };
};
export const OsIcon: React.FC<{ os: string; s?: number }> = ({ os, s = 16 }) =>
  os === 'macos' ? <Icon.apple s={s} /> : os === 'linux' ? <Icon.linux s={s} /> : os === 'android' ? <Icon.android s={s} /> : os === 'chromeos' ? <Icon.chromeos s={s} /> : os === 'tizen' || os === 'webos' ? <Icon.tv s={s} /> : <Icon.windows s={s} />;
export const DevIcon: React.FC<{ t?: string; s?: number }> = ({ t, s = 16 }) => {
  const I = ({ laptop: Icon.laptop, phone: Icon.phone, tablet: Icon.tablet, foldable: Icon.fold, chromebook: Icon.chromebook, tv: Icon.tv } as any)[t || 'desktop'] || Icon.monitor;
  return <I s={s} />;
};
export const Avatar: React.FC<{ seed: number | string; os: string; running?: boolean }> = ({ seed, os, running }) => (
  <span className="avatar" style={avatarStyle(seed)}><OsIcon os={os} />{running !== undefined && <i className={`st ${running ? 'on' : ''}`} />}</span>
);

/** Country code chip. Windows ships no flag emoji glyphs, so text chips render identically everywhere. */
export const CC: React.FC<{ cc?: string }> = ({ cc }) => cc && cc.length === 2 ? <span className="cc">{cc.toUpperCase()}</span> : <span className="cc none">—</span>;
export const flag = (cc: string) => (cc && cc.length === 2 ? cc.toUpperCase() : '··');

export const Latency: React.FC<{ ms?: number }> = ({ ms }) => {
  if (!ms) return <span className="sub">—</span>;
  const c = ms < 150 ? 'g' : ms < 350 ? 'm' : 'b';
  return <span className={`lat ${c}`}><span className="bars"><i /><i /><i /></span><span className="num" style={{ fontSize: 12 }}>{ms} ms</span></span>;
};

export const ago = (ts: number) => {
  if (!ts) return '—';
  const s = Date.now() / 1000 - ts;
  if (s < 60) return '刚刚';
  if (s < 3600) return `${Math.floor(s / 60)} 分钟前`;
  if (s < 86400) return `${Math.floor(s / 3600)} 小时前`;
  return `${Math.floor(s / 86400)} 天前`;
};

/** One pointer listener feeds --mx/--my to whichever .spot is under the cursor. */
export const useSpotlight = () => {
  useEffect(() => {
    const move = (e: PointerEvent) => {
      const el = (e.target as HTMLElement)?.closest?.('.spot') as HTMLElement | null;
      if (!el) return;
      const r = el.getBoundingClientRect();
      el.style.setProperty('--mx', `${e.clientX - r.left}px`);
      el.style.setProperty('--my', `${e.clientY - r.top}px`);
    };
    window.addEventListener('pointermove', move, { passive: true });
    return () => window.removeEventListener('pointermove', move);
  }, []);
};

export const Ambient: React.FC = () => <><div className="aurora"><i /><i /><i /></div><div className="grain" /></>;

export const Splash: React.FC = () => {
  const [gone, setGone] = useState(false);
  useEffect(() => { const t = setTimeout(() => setGone(true), 1800); return () => clearTimeout(t); }, []);
  if (gone) return null;
  return (
    <div className="splash"><div className="mark">
      <div className="logo"><Icon.ghost s={38} /></div>
      <div className="word">Ghost<b>Browser</b></div>
      <div className="line"><i /></div>
    </div></div>
  );
};

export const StatCard: React.FC<{ l: string; n: number; of?: number; ic: keyof typeof Icon; tone?: string; pct: number }> = ({ l, n, of, ic, tone = '', pct }) => {
  const I = Icon[ic];
  return <div className={`stat spot ${tone}`}>
    <div className="top"><span className="l">{l}</span><span className="ic"><I /></span></div>
    <div className="n"><CountUp n={n} />{of != null && <small className="num">/ {of}</small>}</div>
    <div className="bar"><i style={{ width: `${Math.max(pct, 2)}%` }} /></div>
  </div>;
};

// ---- frameless window chrome ----
const rt = () => (window as any).runtime;
export const useMaximised = () => {
  const [max, setMax] = useState(false);
  React.useEffect(() => {
    const check = async () => { try { const m = await rt()?.WindowIsMaximised?.(); setMax(!!m); } catch { /* browser preview */ } };
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, []);
  React.useEffect(() => { document.documentElement.classList.toggle('maximised', max); }, [max]);
  return max;
};

export const WindowControls: React.FC<{ max: boolean }> = ({ max }) => (
  <div className="winctl">
    <button title="最小化" onClick={() => rt()?.WindowMinimise?.()}>
      <svg viewBox="0 0 12 12" width="12" height="12"><path d="M2 6h8" stroke="currentColor" strokeWidth="1.2" /></svg>
    </button>
    <button title={max ? '还原' : '最大化'} onClick={() => rt()?.WindowToggleMaximise?.()}>
      {max
        ? <svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="1.1"><rect x="2" y="3.5" width="6.5" height="6.5" rx="1.2" /><path d="M4 3.5V3a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v4a1 1 0 0 1-1 1h-.5" /></svg>
        : <svg viewBox="0 0 12 12" width="12" height="12" fill="none" stroke="currentColor" strokeWidth="1.1"><rect x="2" y="2" width="8" height="8" rx="1.5" /></svg>}
    </button>
    <button className="close" title="关闭" onClick={() => rt()?.Quit?.()}>
      <svg viewBox="0 0 12 12" width="12" height="12"><path d="M2.5 2.5l7 7M9.5 2.5l-7 7" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" /></svg>
    </button>
  </div>
);
