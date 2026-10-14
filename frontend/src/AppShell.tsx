// App shell: 待重启 badge, update pill + modal, announcements bell, site
// shortcuts launcher and the Settings 应用与更新 card.
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Profile, Settings } from './api';
import { Icon, Field, Toggle, useToast, useExit } from './ui';

const A = api as any;

// ---------- 待重启 ----------

/** Running profiles whose launch settings changed since launch. Re-checked
 *  whenever the profile list, proxies or running set change. */
export function usePendingRestart(deps: any[]): [Set<string>, () => void] {
  const [ids, setIds] = useState<Set<string>>(new Set());
  const refresh = useCallback(() => { A.PendingRestart?.().then((l: string[]) => setIds(new Set(l || []))).catch(() => {}); }, []);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(refresh, deps);
  useEffect(() => onEvent('profile:exit', refresh), [refresh]);
  return [ids, refresh];
}

export const RestartTag: React.FC<{ compact?: boolean }> = ({ compact }) => (
  <span className={`badge warn restart-tag ${compact ? 'compact' : ''}`} title="代理、指纹、内核或启动参数已修改，重启环境后生效">
    <Icon.refresh s={11} />{!compact && '待重启'}
  </span>
);

/** Restart action shared by the list, the run strip and the drawer. */
export function useRestart(onDone: () => void) {
  const toast = useToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const restart = useCallback(async (p: Pick<Profile, 'id' | 'name'>) => {
    setBusy(s => new Set(s).add(p.id));
    try { await A.RestartProfile(p.id); toast(`已重启 ${p.name}，新设置已生效`); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setBusy(s => { const n = new Set(s); n.delete(p.id); return n; }); onDone(); }
  }, [onDone, toast]);
  return { restart, restarting: busy };
}

// ---------- safe markdown (no raw HTML, http/https links only) ----------

const openURL = (href: string) => { const rt = (window as any).runtime; if (rt?.BrowserOpenURL) rt.BrowserOpenURL(href); else window.open(href, '_blank', 'noopener'); };

function inline(text: string, key = 'i'): React.ReactNode[] {
  const out: React.ReactNode[] = [];
  const re = /(\*\*([^*]+)\*\*|`([^`]+)`|\[([^\]]+)\]\(([^)\s]+)\)|\*([^*]+)\*)/g;
  let last = 0, m: RegExpExecArray | null, n = 0;
  while ((m = re.exec(text))) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const k = `${key}-${n++}`;
    if (m[2]) out.push(<b key={k}>{m[2]}</b>);
    else if (m[3]) out.push(<code key={k}>{m[3]}</code>);
    else if (m[4]) out.push(/^https?:\/\//i.test(m[5]) ? <a key={k} href={m[5]} onClick={e => { e.preventDefault(); openURL(m![5]); }}>{m[4]}</a> : <span key={k}>{m[4]}</span>);
    else if (m[6]) out.push(<i key={k}>{m[6]}</i>);
    last = re.lastIndex;
  }
  if (last < text.length) out.push(text.slice(last));
  return out;
}

export const Markdown: React.FC<{ text: string }> = ({ text }) => {
  const blocks: React.ReactNode[] = [];
  const lines = String(text || '').replace(/\r/g, '').split('\n');
  let list: { ordered: boolean; items: string[] } | null = null;
  let para: string[] = [];
  const flushPara = () => { if (para.length) { blocks.push(<p key={blocks.length}>{inline(para.join(' '), 'p' + blocks.length)}</p>); para = []; } };
  const flushList = () => {
    if (!list) return;
    const items = list.items.map((it, i) => <li key={i}>{inline(it, `l${blocks.length}-${i}`)}</li>);
    blocks.push(list.ordered ? <ol key={blocks.length}>{items}</ol> : <ul key={blocks.length}>{items}</ul>);
    list = null;
  };
  for (const raw of lines) {
    const l = raw.trimEnd();
    const h = /^(#{1,3})\s+(.*)$/.exec(l);
    const ul = /^\s*[-*]\s+(.*)$/.exec(l);
    const ol = /^\s*\d+[.)]\s+(.*)$/.exec(l);
    if (!l.trim()) { flushPara(); flushList(); continue; }
    if (h) { flushPara(); flushList(); blocks.push(<h5 key={blocks.length} className={`h${h[1].length}`}>{inline(h[2], 'h' + blocks.length)}</h5>); continue; }
    if (ul || ol) {
      flushPara();
      const ordered = !!ol && !ul;
      if (list && list.ordered !== ordered) flushList();
      if (!list) list = { ordered, items: [] };
      list.items.push((ul || ol)![1]);
      continue;
    }
    flushList(); para.push(l.trim());
  }
  flushPara(); flushList();
  return <div className="md">{blocks}</div>;
};

// ---------- shared catalog store ----------

type Site = { id: string; name: string; url: string; icon: string };
type Group = { id: string; name: string; sites: Site[] };
type Ann = { id: string; title: string; body: string; level: string; start: number; end: number };
export type CatalogState = { catalog: { revision: number; groups: Group[]; announcements: Ann[] }; fetchedAt: number; offline: boolean; error?: string; read: string[]; source: string };

let catState: CatalogState | null = null;
let catLoading = false;
const catSubs = new Set<() => void>();
const catSet = (s: CatalogState) => { catState = s; catSubs.forEach(f => f()); };
let catInit = false;
const catRefresh = async () => {
  catLoading = true; catSubs.forEach(f => f());
  try { catSet(await A.RefreshCatalog()); } catch { /* keep cache */ } finally { catLoading = false; catSubs.forEach(f => f()); }
};
function useCatalog() {
  const [, tick] = useState(0);
  useEffect(() => {
    const f = () => tick(x => x + 1);
    catSubs.add(f);
    if (!catInit) {
      catInit = true;
      A.GetCatalog?.().then((s: CatalogState) => { if (!catState || !catState.fetchedAt) catSet(s); }).catch(() => {});
      onEvent('catalog:changed', (s: CatalogState) => catSet(s));
    }
    return () => { catSubs.delete(f); };
  }, []);
  return { state: catState, loading: catLoading, refresh: catRefresh, markRead: async (ids: string[]) => { try { catSet(await A.MarkAnnouncementsRead(ids)); } catch { /* ignore */ } } };
}

const when = (t: number) => {
  if (!t) return '';
  const d = new Date(t * 1000);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};
const agoShort = (t: number) => {
  if (!t) return '从未';
  const m = Math.max(0, Math.round((Date.now() / 1000 - t) / 60));
  return m < 1 ? '刚刚' : m < 60 ? `${m} 分钟前` : m < 1440 ? `${Math.floor(m / 60)} 小时前` : `${Math.floor(m / 1440)} 天前`;
};

// click-outside for popovers
function useOutside(ref: React.RefObject<HTMLElement | null>, on: boolean, close: () => void) {
  useEffect(() => {
    if (!on) return;
    const d = (e: MouseEvent) => { if (ref.current && !ref.current.contains(e.target as Node)) close(); };
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape') close(); };
    document.addEventListener('mousedown', d); window.addEventListener('keydown', k);
    return () => { document.removeEventListener('mousedown', d); window.removeEventListener('keydown', k); };
  }, [on, close, ref]);
}

// ---------- announcements ----------

const LEVEL: Record<string, [string, string]> = { info: ['通知', 'cy'], warn: ['重要', 'warn'], critical: ['紧急', 'fail'] };

const AnnouncementBell: React.FC = () => {
  const { state, loading, refresh, markRead } = useCatalog();
  const [open, setOpen] = useState(false);
  const [expanded, setExpanded] = useState<string>('');
  const ref = useRef<HTMLDivElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutside(ref, open, close);
  const list = state?.catalog.announcements || [];
  const read = new Set(state?.read || []);
  const unread = list.filter(a => !read.has(a.id));
  const toggle = () => { const next = !open; setOpen(next); if (next) setExpanded(unread[0]?.id || list[0]?.id || ''); };
  // Mark everything shown as read when the panel closes (however it closes).
  const wasOpen = useRef(false);
  useEffect(() => {
    if (wasOpen.current && !open && unread.length) markRead(unread.map(a => a.id));
    wasOpen.current = open;
  }, [open]); // eslint-disable-line react-hooks/exhaustive-deps
  return (
    <div className="ash-pop-wrap" ref={ref}>
      <button className={`ash-ib ${open ? 'on' : ''}`} onClick={toggle} title="公告" aria-label={`公告${unread.length ? `，${unread.length} 条未读` : ''}`}>
        <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" /><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" /></svg>
        {unread.length > 0 && <span className="ash-dot num">{unread.length > 9 ? '9+' : unread.length}</span>}
      </button>
      {open && <div className="ash-pop ann-pop">
        <div className="ash-pop-head">
          <b>公告</b>
          {unread.length > 0 && <span className="badge cy">{unread.length} 条未读</span>}
          <span className="spacer" />
          {state?.offline && <span className="ash-off" title={state.error || ''}><span className="d" />离线缓存</span>}
          <button className="btn ghost icon sm" title="刷新" disabled={loading} onClick={refresh}>{loading ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.refresh s={14} />}</button>
        </div>
        <div className="ann-list">
          {!state && <div className="ash-empty"><ThinkingOrb state="connecting" size={32} theme="dark" /><span>正在加载公告…</span></div>}
          {state && list.length === 0 && <div className="ash-empty">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"><path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9" /><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" /></svg>
            <b>暂无公告</b><span>{state.offline ? (state.fetchedAt ? '无法连接服务器，显示的是上次缓存' : '没有连接到同步服务器，配置后会自动获取公告') : '有新公告时这里会显示红点'}</span>
          </div>}
          {list.map(a => {
            const [lv, tone] = LEVEL[a.level] || LEVEL.info; const isOpen = expanded === a.id;
            return <div key={a.id} className={`ann ${isOpen ? 'open' : ''} ${read.has(a.id) ? '' : 'unread'} lv-${a.level}`}>
              <button className="ann-h" onClick={() => setExpanded(isOpen ? '' : a.id)}>
                <span className={`badge ${tone}`}>{lv}</span>
                <span className="ann-t">{a.title || '公告'}</span>
                <span className="ann-d num">{when(a.start)}</span>
                <svg className="chev" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="m6 9 6 6 6-6" /></svg>
              </button>
              {isOpen && <div className="ann-b"><Markdown text={a.body} />{a.end > 0 && <div className="ann-end num">有效期至 {when(a.end)}</div>}</div>}
            </div>;
          })}
        </div>
        {state && <div className="ash-pop-foot">{state.fetchedAt ? `更新于 ${agoShort(state.fetchedAt)}` : '尚未从服务器获取'}{state.offline && state.error ? ` · ${state.error}` : ''}</div>}
      </div>}
    </div>
  );
};

// ---------- updater ----------

type UpdateInfo = { current: string; latest: string; notes: string; available: boolean; source: string; checkedAt: number; error?: string };
type UpdateProgress = { phase: string; got: number; total: number; pct: number; error?: string };
const mb = (b: number) => `${(b / 1048576).toFixed(b > 1e8 ? 0 : 1)} MB`;
const SRC: Record<string, string> = { server: '同步服务器', custom: '自定义地址', github: 'GitHub Releases' };

let updState: { info: UpdateInfo | null; prog: UpdateProgress } = { info: null, prog: { phase: 'idle', got: 0, total: 0, pct: 0 } };
const updSubs = new Set<() => void>();
const updPub = () => updSubs.forEach(f => f());
let updInit = false;
function useUpdater() {
  const [, tick] = useState(0);
  useEffect(() => {
    const f = () => tick(x => x + 1);
    updSubs.add(f);
    if (!updInit) {
      updInit = true;
      A.UpdateInfo?.().then((i: UpdateInfo) => { updState = { ...updState, info: i }; updPub(); }).catch(() => {});
      A.UpdateProgress?.().then((p: UpdateProgress) => { updState = { ...updState, prog: p }; updPub(); }).catch(() => {});
      onEvent('update:available', (i: UpdateInfo) => { updState = { ...updState, info: i }; updPub(); });
      onEvent('update:progress', (p: UpdateProgress) => { updState = { ...updState, prog: p }; updPub(); });
    }
    return () => { updSubs.delete(f); };
  }, []);
  const check = async () => { const i: UpdateInfo = await A.CheckUpdate(); updState = { ...updState, info: i }; updPub(); return i; };
  const install = async () => { updState = { ...updState, prog: { phase: 'downloading', got: 0, total: 0, pct: 0 } }; updPub(); try { await A.InstallUpdate(); } catch (e: any) { updState = { ...updState, prog: { phase: 'error', got: 0, total: 0, pct: 0, error: String(e?.message || e) } }; updPub(); } };
  return { ...updState, check, install };
}

const PHASE: Record<string, string> = { downloading: '正在下载', verifying: '正在校验签名', installing: '正在安装', restarting: '正在重启 GhostBrowser…' };

export const UpdateModal: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const { info, prog, install } = useUpdater();
  const [out, close] = useExit(onClose);
  const busy = ['downloading', 'verifying', 'installing', 'restarting'].includes(prog.phase);
  const pct = prog.phase === 'downloading' ? prog.pct : busy ? 100 : 0;
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !busy && close()}>
      <div className="modal upd-modal">
        <div className="dhead">
          <div className="glyph"><Icon.download s={18} /></div>
          <div><h3>发现新版本 {info?.latest ? `v${info.latest}` : ''}</h3><div className="sub">当前 v{info?.current || '—'}{info?.source ? ` · 来自${SRC[info.source] || info.source}` : ''}</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={busy} onClick={() => close()}><Icon.x /></button></div>
        </div>
        <div className="dbody">
          <div className="upd-notes">{info?.notes?.trim() ? <Markdown text={info.notes} /> : <span className="sub">这个版本没有填写更新说明。</span>}</div>
          {busy && <div className="upd-prog">
            <div className="upd-prog-row"><span>{PHASE[prog.phase]}</span><span className="num">{prog.phase === 'downloading' && prog.total > 0 ? `${mb(prog.got)} / ${mb(prog.total)} · ${Math.floor(prog.pct)}%` : prog.phase === 'downloading' && prog.got > 0 ? mb(prog.got) : ''}</span></div>
            <div className={`env-bar ${prog.phase === 'downloading' && !prog.total ? 'indet' : ''}`}><i style={{ width: `${Math.max(pct, 2)}%` }} /></div>
          </div>}
          {prog.phase === 'error' && <div className="notice bad" style={{ marginTop: 12, marginBottom: 0 }}><Icon.alert /><span className="grow">{prog.error || '更新失败'}</span></div>}
          <div className="sub upd-hint"><Icon.shield s={12} /> 安装前校验 SHA-256 与 Ed25519 签名，签名不符会拒绝安装。更新时会先关闭所有运行中的环境并完成同步，然后自动重启。</div>
        </div>
        <div className="dfoot"><div className="spacer" />
          <button className="btn" disabled={busy} onClick={() => close()}>稍后</button>
          <button className="btn primary" disabled={busy || !info?.available} onClick={install}>{busy ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.download s={15} />} {prog.phase === 'error' ? '重试' : busy ? PHASE[prog.phase] : '立即更新'}</button>
        </div>
      </div>
    </div>
  );
};

const UpdatePill: React.FC = () => {
  const { info, prog } = useUpdater();
  const [open, setOpen] = useState(false);
  const busy = ['downloading', 'verifying', 'installing', 'restarting'].includes(prog.phase);
  if (!info?.available && !busy) return null;
  return <>
    <button className={`upd-pill ${busy ? 'busy' : ''}`} onClick={() => setOpen(true)} title="查看更新说明">
      {busy ? <><span className="upd-ring" style={{ ['--p' as any]: `${prog.phase === 'downloading' ? prog.pct : 100}%` }} />{prog.phase === 'downloading' ? `下载中 ${Math.floor(prog.pct)}%` : PHASE[prog.phase]?.replace('…', '')}</>
        : <><span className="spark" />新版本 v{info!.latest}</>}
    </button>
    {open && <UpdateModal onClose={() => setOpen(false)} />}
  </>;
};

/** Right side of the topbar: update pill + announcements bell. */
export const TopbarExtras: React.FC = () => (
  <div className="ash-top"><UpdatePill /><AnnouncementBell /></div>
);

// ---------- 网站快捷入口 ----------

const SiteIcon: React.FC<{ s: Site }> = ({ s }) => {
  const [bad, setBad] = useState(false);
  const letter = (s.name || '?').trim().charAt(0).toUpperCase();
  const hue = [...(s.id || s.name)].reduce((h, c) => (h * 31 + c.charCodeAt(0)) % 360, 7);
  if (s.icon && !bad) return <span className="site-ic"><img src={s.icon} alt="" onError={() => setBad(true)} /></span>;
  return <span className="site-ic letter" style={{ ['--h' as any]: hue }}>{letter}</span>;
};

/** Caret next to 启动: open the profile straight to a site. */
export const SiteLaunch: React.FC<{ profile: Profile; running: boolean; onLaunched: () => void }> = ({ profile, running, onLaunched }) => {
  const toast = useToast();
  const { state, loading, refresh } = useCatalog();
  const [open, setOpen] = useState(false);
  const [g, setG] = useState('');
  const [q, setQ] = useState('');
  const [custom, setCustom] = useState('');
  const [busy, setBusy] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const close = useCallback(() => setOpen(false), []);
  useOutside(ref, open, close);
  const groups = state?.catalog.groups || [];
  const cur = groups.find(x => x.id === g) || groups[0];
  const sites = useMemo(() => {
    if (!q.trim()) return cur?.sites || [];
    const ql = q.toLowerCase();
    return groups.flatMap(x => x.sites).filter(s => `${s.name} ${s.url}`.toLowerCase().includes(ql));
  }, [q, cur, groups]);
  const go = async (url: string, id: string) => {
    let u = url.trim();
    if (u && !/^https?:\/\//i.test(u)) u = 'https://' + u;
    if (!/^https?:\/\/[^\s/]+\.[^\s]+/i.test(u)) { toast('请输入有效的网址，例如 https://example.com', 'bad'); return; }
    setBusy(id);
    try { await A.StartProfileAt(profile.id, u); toast(running ? `已在 ${profile.name} 中打开 ${u.replace(/^https?:\/\//, '').replace(/\/$/, '')}` : `已启动 ${profile.name}`); setOpen(false); setCustom(''); onLaunched(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setBusy(''); }
  };
  return (
    <div className="ash-pop-wrap" ref={ref}>
      <button className={`btn sm ${running ? '' : 'go'} site-caret ${open ? 'on' : ''}`} title={running ? '在此环境打开网站' : '启动并打开网站'} onClick={() => { setOpen(o => !o); if (!state?.fetchedAt && !loading) refresh(); }}>
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="m6 9 6 6 6-6" /></svg>
      </button>
      {open && <div className="ash-pop site-pop">
        <div className="ash-pop-head">
          <b>网站快捷入口</b><span className="sub ellip">{running ? `在「${profile.name}」中新标签页打开` : `启动「${profile.name}」并直接打开`}</span>
          <span className="spacer" />
          {state?.offline && state.fetchedAt > 0 && <span className="ash-off" title={state.error || ''}><span className="d" />离线缓存</span>}
          <button className="btn ghost icon sm" title="刷新快捷入口" disabled={loading} onClick={refresh}>{loading ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.refresh s={14} />}</button>
        </div>
        <div className="site-search"><Icon.search s={14} /><input autoFocus type="text" placeholder="搜索网站" value={q} onChange={e => setQ(e.target.value)} /></div>
        {!q && groups.length > 1 && <div className="site-tabs">{groups.map(x => <button key={x.id} className={x.id === cur?.id ? 'on' : ''} onClick={() => setG(x.id)}>{x.name}<span className="num">{x.sites.length}</span></button>)}</div>}
        <div className="site-grid">
          {!state && <div className="ash-empty"><ThinkingOrb state="connecting" size={32} theme="dark" /><span>正在加载快捷入口…</span></div>}
          {state && groups.length === 0 && <div className="ash-empty"><Icon.grid s={26} /><b>还没有快捷入口</b><span>{state.offline && !state.fetchedAt ? '连接同步服务器后自动获取（服务器上的 catalog.json）。也可以在下方直接输入网址。' : '服务器的 catalog.json 里没有网站，可以在下方直接输入网址。'}</span></div>}
          {state && groups.length > 0 && sites.length === 0 && <div className="ash-empty"><Icon.search s={24} /><span>没有匹配「{q}」的网站</span></div>}
          {sites.map(s => <button key={s.id} className="site" disabled={!!busy} onClick={() => go(s.url, s.id)} title={s.url}>
            {busy === s.id ? <span className="site-ic"><ThinkingOrb state="connecting" size={20} theme="dark" /></span> : <SiteIcon s={s} />}
            <span className="site-t"><b>{s.name}</b><small>{s.url.replace(/^https?:\/\//, '').replace(/\/$/, '')}</small></span>
          </button>)}
        </div>
        <form className="site-custom" onSubmit={e => { e.preventDefault(); go(custom, '_custom'); }}>
          <Icon.globe s={14} /><input type="text" placeholder="或输入网址，例如 example.com" value={custom} onChange={e => setCustom(e.target.value)} />
          <button className="btn sm primary" disabled={!custom.trim() || !!busy}>{busy === '_custom' ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : '打开'}</button>
        </form>
      </div>}
    </div>
  );
};

// ---------- Settings: 应用与更新 ----------

export const AppShellSettings: React.FC<{ s: Settings; set: (patch: Partial<Settings>) => void; index?: number }> = ({ s, set, index = 4 }) => {
  const toast = useToast();
  const { info, prog, check } = useUpdater();
  const [checking, setChecking] = useState(false);
  const [tray, setTray] = useState(true);
  const [open, setOpen] = useState(false);
  const [ver, setVer] = useState('');
  useEffect(() => { A.TrayAvailable?.().then(setTray).catch(() => {}); A.AppVersion?.().then(setVer).catch(() => {}); }, []);
  const x = s as any;
  const doCheck = async () => {
    setChecking(true);
    try { const i = await check(); if (i.error && !i.available) toast(`检查更新失败：${i.error}`, 'bad'); else if (!i.available) toast(`已是最新版本 v${i.current}`); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setChecking(false); }
  };
  const busy = ['downloading', 'verifying', 'installing', 'restarting'].includes(prog.phase);
  const tone = info?.available ? 'warn' : info?.error ? 'bad' : '';
  return (
    <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: index }}>
      <div className="head"><span className="ic"><Icon.download /></span><div><h4>应用与更新</h4><span>托盘常驻、在线更新、快捷入口与公告</span></div></div>
      <div className={`geo-card upd-card ${tone}`}>
        <div className="upd-card-row">
          <div style={{ minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span className={`badge ${info?.available ? 'warn' : info?.error ? 'fail' : info?.checkedAt ? 'ok' : 'unknown'}`}><span className="d" />{info?.available ? `可更新到 v${info.latest}` : info?.error ? '检查失败' : info?.checkedAt ? '已是最新' : '尚未检查'}</span>
              <b style={{ fontSize: 12.5 }}>当前版本 v{info?.current || ver || '—'}</b>
            </div>
            <div className="sub" style={{ marginTop: 5 }}>{info?.checkedAt ? `上次检查 ${agoShort(info.checkedAt)}${info.source ? ` · ${SRC[info.source] || info.source}` : ''}` : '启动 15 秒后自动检查，之后每 6 小时一次'}</div>
            {info?.error && <div className="sub upd-err">{info.error}</div>}
          </div>
          <div className="row" style={{ gap: 8, flex: 'none', width: 'auto' }}>
            <button className="btn" disabled={checking || busy} onClick={doCheck}>{checking ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <Icon.refresh s={14} />} {checking ? '检查中…' : '检查更新'}</button>
            {(info?.available || busy) && <button className="btn primary" onClick={() => setOpen(true)}><Icon.download s={14} /> {busy ? '查看进度' : '立即更新'}</button>}
          </div>
        </div>
      </div>
      <Toggle on={!x.updateAutoOff} onChange={v => set({ updateAutoOff: !v } as any)} label="自动检查更新" hint="启动时和每 6 小时检查一次，有新版本时在顶栏提示，不会自动安装" />
      <Toggle on={tray && !x.closeToTrayOff} onChange={v => { if (!tray) { toast('当前系统不支持托盘，关闭窗口会直接退出', 'bad'); return; } set({ closeToTrayOff: !v } as any); }} label="关闭窗口时最小化到托盘" hint={tray ? '运行中的环境和代理保持运行；从托盘图标恢复窗口或选择「退出」真正关闭' : '仅 Windows 支持托盘，当前系统关闭窗口会直接退出'} />
      <div className="grid2">
        <Field label="更新地址（可选）" hint="留空 = 同步服务器的 update/latest.json"><input type="text" value={x.updateUrl || ''} onChange={e => set({ updateUrl: e.target.value } as any)} placeholder="https://你的域名/latest.json" /></Field>
        <Field label="GitHub 公开仓库（备用）" hint="owner/repo，最新 Release 需附带 latest.json；私有仓库不可用"><input type="text" value={x.updateGithub || ''} onChange={e => set({ updateGithub: e.target.value } as any)} placeholder="owner/repo" /></Field>
      </div>
      <Field label="快捷入口与公告地址（可选）" hint="留空 = 同步服务器的 catalog.json；离线时使用上次缓存"><input type="text" value={x.catalogUrl || ''} onChange={e => set({ catalogUrl: e.target.value } as any)} placeholder="https://你的域名/catalog.json" /></Field>
      {open && <UpdateModal onClose={() => setOpen(false)} />}
    </div>
  );
};
