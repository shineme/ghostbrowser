import React, { createContext, useContext, useState, useCallback } from 'react';

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
  ghost: (p: IP) => <svg {...S(p)}><path d="M9 10h.01M15 10h.01M12 2a8 8 0 0 0-8 8v12l3-3 2.5 2.5L12 21l2.5 2.5L17 19l3 3V10a8 8 0 0 0-8-8Z" /></svg>,
};

// ---- toast ----
type Toast = { id: number; msg: string; kind: string };
const ToastCtx = createContext<(msg: string, kind?: string) => void>(() => {});
export const useToast = () => useContext(ToastCtx);
export const ToastHost: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [list, setList] = useState<Toast[]>([]);
  const push = useCallback((msg: string, kind = 'ok') => {
    const id = Date.now() + Math.random();
    setList(l => [...l, { id, msg, kind }]);
    setTimeout(() => setList(l => l.filter(t => t.id !== id)), 3200);
  }, []);
  return (
    <ToastCtx.Provider value={push}>
      {children}
      <div className="toasts">{list.map(t => <div key={t.id} className={`toast ${t.kind}`}>{t.msg}</div>)}</div>
    </ToastCtx.Provider>
  );
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

export const flag = (cc: string) => {
  if (!cc || cc.length !== 2) return '🌐';
  return String.fromCodePoint(...[...cc.toUpperCase()].map(c => 0x1f1a5 + c.charCodeAt(0)));
};
export const ago = (ts: number) => {
  if (!ts) return '—';
  const s = Date.now() / 1000 - ts;
  if (s < 60) return '刚刚';
  if (s < 3600) return `${Math.floor(s / 60)} 分钟前`;
  if (s < 86400) return `${Math.floor(s / 3600)} 小时前`;
  return `${Math.floor(s / 86400)} 天前`;
};
