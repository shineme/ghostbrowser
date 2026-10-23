import ghostMark from './assets/images/ghost-mark.svg';
import React, { useEffect, useMemo, useState, useCallback, useLayoutEffect, useRef } from 'react';
import { BorderBeam } from 'border-beam';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Proxy, Settings, onEvent } from './api';
import { DeleteModal, DelTarget } from './DeleteModal';
import { TrashModal, useTrashCount } from './TrashView';
import { Icon, Field, flag, ago, useToast, useConfirm, useExit, ToastHost, WindowControls, useMaximised, StatCard, Avatar, CC, CountUp, OsIcon, Ambient, Splash, useSpotlight, Pick, Segmented, DevIcon, MoreMenu } from './ui';
import { DEVICE_TYPES, devType, devName, osName as osNameOf, needsOpen } from './devices';
import { ProfileDrawer } from './ProfileDrawer';
import { useSubs, groupKey, groupOptions, isCore, protoOf } from './Subscriptions';
import { ProxyView } from './ProxyView';
import { SettingsView } from './SettingsView';
import { CloudView, syncState } from './CloudView';
import { syncErrorToast } from './leftover';
import { SyncPrompt } from './SyncPrompt';
import { EnvSetup } from './EnvSetup';
import { NativeTools } from './NativeTools';
import { TransferDock, InlineTransfer } from './Transfers';
import { RpaView } from './RpaView';
import { ExtensionsView } from './ExtensionsView';
import { AuditView } from './AuditView';
import { WindowTools } from './WindowTools';
import { KernelMissingModal } from './KernelInstall';
import { EnvCheckModal } from './EnvCheckModal';
import { TerminalView } from './TerminalView';
import { FoxIcon, useFirefoxInfo, ffVersionLabel } from './FirefoxKernel';
import { ShareDialog, ShIc, shareLeft } from './SharingView'; // share
import { TopbarExtras, usePendingRestart, RestartTag, useRestart, SiteLaunch } from './AppShell';
import { DefaultStar, LinkPicker, useDefaultBrowser, DBStatus } from './DefaultBrowser';

type Tab = 'profiles' | 'proxies' | 'rpa' | 'extensions' | 'terminal' | 'cloud' | 'audit' | 'settings';

function Shell() {
  const toast = useToast();
  const [tab, setTab] = useState<Tab>('profiles');
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [running, setRunning] = useState<Set<string>>(new Set());
  const [settings, setSettings] = useState<Settings | null>(null);
  const [settingsRev, setSettingsRev] = useState(0);
  const max = useMaximised();

  const reload = useCallback(async () => {
    setProfiles(await api.ListProfiles());
    setProxies(await api.ListProxies());
    const r = await api.Running();
    setRunning(new Set(r.map(x => x.profileId)));
  }, []);

  useEffect(() => { reload(); api.GetSettings().then(setSettings); }, [reload]);
  // Auto sync: refresh the list whenever a profile is pulled/pushed; surface failures.
  useEffect(() => onEvent('sync:event', (e: any) => {
    if (e.phase === 'item' && e.id === '_global' && e.action === 'pull' && !e.error) { api.GetSettings().then(st => { setSettings(st); setSettingsRev(r => r + 1); }); }
    if (e.phase === 'item') { reload(); if (e.message) toast(`${e.name || e.id}：${e.message}`, 'bad'); else if (e.error) toast(`${e.name || e.id} ${e.action === 'pull' ? '下载' : '上传'}失败：${e.error}`, 'bad'); else if (e.action === 'push') toast(`已同步 ${e.name} 到服务器`); else if (e.action === 'pull') toast(`已从云端拉取 ${e.name}`); }
    if (e.phase === 'done' && (e.pulled || e.pushed || e.failed)) { reload(); toast(e.message, e.failed ? 'bad' : undefined); }
    if (e.phase === 'error') toast(e.message, 'bad');
    if (e.phase === 'deleted' || e.phase === 'deleted-pending' || e.phase === 'deleted-cloud') { reload(); if (e.message) toast(e.message, e.phase === 'deleted-pending' ? 'bad' : undefined); }
  }), [reload]);
  // share: ended shares removed in the background; jump to 云端同步 → 设备与密钥
  useEffect(() => onEvent('shares:changed', () => { reload(); toast('有分享已撤销或过期，本机副本已删除'); }), [reload]);
  // 订阅 → 「去设置下载」内核: open settings and scroll to the core card
  useEffect(() => { const f = (e: any) => { setTab('settings'); setTimeout(() => document.getElementById(e.detail || '')?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 300); }; window.addEventListener('gb:goto-settings', f); return () => window.removeEventListener('gb:goto-settings', f); }, []);
  useEffect(() => { const f = (e: any) => { setTab('cloud'); setTimeout(() => window.dispatchEvent(new CustomEvent('gb:cloud-tab', { detail: e.detail })), 60); }; window.addEventListener('gb:goto-cloud', f); return () => window.removeEventListener('gb:goto-cloud', f); }, []);
  useEffect(() => onEvent('profile:exit', (id: string) => { setRunning(s => { const n = new Set(s); n.delete(id); return n; }); }), []);
  // default browser: a link launched a profile, or failed to open
  const [dbStatus, setDbStatus] = useDefaultBrowser();
  useEffect(() => onEvent('profile:started', (id: string) => { setRunning(s => new Set(s).add(id)); reload(); }), [reload]);
  useEffect(() => onEvent('defaultbrowser:opened', (e: any) => { if (e?.error) toast(`链接没有打开（${e.name || '环境'}）：${e.error}`, 'bad'); }), []);

  const proxyName = useMemo(() => Object.fromEntries(proxies.map(p => [p.id, p])), [proxies]);

  const titles: Record<Tab, [string, string]> = {
    profiles: ['浏览器环境', '每个环境拥有独立指纹、代理与数据目录'],
    proxies: ['代理管理', '导入、检测并为环境分配代理'],
    rpa: ['自动化', '可视化流程，批量在多个环境中运行'],
    extensions: ['扩展中心', '下载 Chrome 扩展并分配给环境'],
    terminal: ['终端', '用环境的时区、语言和代理打开 Claude Code / Codex / Gemini CLI'],
    cloud: ['云端同步', '把环境加密同步到你自己的 VPS'],
    audit: ['接口与审计', 'MCP、本地 API 与操作日志'],
    settings: ['设置', '内核、本地 API、同步服务器'],
  };

  const nav: [Tab, keyof typeof Icon, string, number | null][] = [
    ['profiles', 'layers', '环境', profiles.length],
    ['proxies', 'globe', '代理', proxies.length],
    ['rpa', 'bolt', '自动化', null],
    ['extensions', 'grid', '扩展', null],
    ['terminal', 'console', '终端', null],
    ['cloud', 'cloud', '云同步', null],
    ['audit', 'terminal', '接口审计', null],
    ['settings', 'settings', '设置', null],
  ];

  const navRef = useRef<HTMLDivElement>(null);
  const [pillY, setPillY] = useState(0);
  // Sidebar collapse (Ctrl+B): icon-only rail, remembered across launches.
  const [collapsed, setCollapsed] = useState(() => { try { return localStorage.getItem('gb.sideCollapsed') === '1'; } catch { return false; } });
  const [tip, setTip] = useState<{ text: string; y: number } | null>(null);
  const toggleSide = useCallback(() => setCollapsed(c => { try { localStorage.setItem('gb.sideCollapsed', c ? '0' : '1'); } catch { /* private mode */ } return !c; }), []);
  const placePill = useCallback(() => {
    const el = navRef.current?.querySelector<HTMLButtonElement>(`button[data-t="${tab}"]`);
    if (el) setPillY(el.offsetTop);
  }, [tab]);
  useLayoutEffect(placePill, [placePill, collapsed]);
  useEffect(() => { window.addEventListener('resize', placePill); return () => window.removeEventListener('resize', placePill); }, [placePill]);
  useEffect(() => { setTip(null); }, [collapsed]);
  const tipOn = (text: string) => (e: React.MouseEvent<HTMLElement>) => { if (!collapsed) return; const r = e.currentTarget.getBoundingClientRect(); setTip({ text, y: r.top + r.height / 2 }); };
  const tipOff = () => setTip(null);
  useSpotlight();
  // Ctrl/Cmd+1..4 switches sections
  useEffect(() => {
    const k = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && /^[1-8]$/.test(e.key)) { e.preventDefault(); setTab(nav[+e.key - 1][0]); }
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && !e.altKey && e.key.toLowerCase() === 'b' && !(e.target as HTMLElement)?.isContentEditable) { e.preventDefault(); toggleSide(); }
    };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  return (
    <div className={`app ${max ? 'max' : ''} ${collapsed ? 'side-collapsed' : ''}`}>
      <Ambient />
      <WindowControls max={max} />
      <aside className="side" onTransitionEnd={e => { if (e.target === e.currentTarget || (e.target as HTMLElement).classList?.contains('side')) placePill(); }}>
        <div className="brand">
          <div className="logo"><img src={ghostMark} alt="" draggable={false} /></div>
          <div className="brand-txt"><h1>Ghost<b>Browser</b></h1><span>指纹浏览器 · 群控管理</span></div>
        </div>
        <div className="nav-label"><span>工作台</span></div>
        <nav className="nav" ref={navRef}>
          <span className="pill" style={{ transform: `translateY(${pillY}px)` }} />
          {nav.map(([t, ic, label, count], i) => {
            const I = Icon[ic];
            return <button key={t} data-t={t} className={tab === t ? 'active' : ''} onClick={() => setTab(t)} aria-label={label} onMouseEnter={tipOn(count ? `${label} · ${count}　Ctrl+${i + 1}` : `${label}　Ctrl+${i + 1}`)} onMouseLeave={tipOff}>
              <I /> <span className="nl">{label}</span> {count != null && <span className={`count num ${count ? '' : 'zero'}`}>{count}</span>}
            </button>;
          })}
        </nav>
        <div className="spacer" />
        <button className="side-toggle" onClick={toggleSide} aria-label={collapsed ? '展开侧栏' : '收起侧栏'} aria-expanded={!collapsed} onMouseEnter={tipOn('展开侧栏　Ctrl+B')} onMouseLeave={tipOff}>
          <Icon.sidebar s={17} /><span className="nl">收起侧栏</span><kbd className="nl">Ctrl B</kbd>
        </button>
        <div className="status-mini" onMouseEnter={tipOn(`本地 API :${settings?.apiPort ?? '—'} · 运行中 ${running.size} / ${profiles.length}`)} onMouseLeave={tipOff}><span className="live" /><b className="num">{running.size}</b></div>
        <div className="status-card spot">
          <div className="row1"><span className="live" /> 本地 API 运行中</div>
          <div className="row2"><span>端口</span><b className="num">:{settings?.apiPort ?? '—'}</b></div>
          <div className="row2"><span>内核</span><b>定制 Chromium</b></div>
          <div className="row2"><span>运行中</span><b className="num">{running.size} / {profiles.length}</b></div>
        </div>
      </aside>
      {tip && collapsed && <div className="side-tip" style={{ top: tip.y }}>{tip.text.split('　')[0]}{tip.text.includes('　') && <kbd>{tip.text.split('　')[1]}</kbd>}</div>}

      <main className="main">
        <div className="topbar">
          <div key={tab} className="page">
            <h2>{titles[tab][0]}</h2>
            <div className="sub">{titles[tab][1]}</div>
          </div>
          <TopbarExtras />
        </div>
        <div className="content">
          <div key={tab} className="page">
            {tab === 'profiles' && <ProfilesView {...{ profiles, proxyName, running, setRunning, reload, dbStatus, setDbStatus }} />}
            {tab === 'proxies' && <ProxyView proxies={proxies} profiles={profiles} reload={reload} />}
            {tab === 'rpa' && <RpaView profiles={profiles} reload={reload} />}
            {tab === 'extensions' && <ExtensionsView profiles={profiles} reload={reload} />}
            {tab === 'terminal' && <TerminalView profiles={profiles} proxyName={proxyName} />}
            {tab === 'audit' && <AuditView profiles={profiles} />}
            {tab === 'cloud' && <CloudView profiles={profiles} settings={settings} reload={reload} />}
            {tab === 'settings' && settings && <SettingsView key={settingsRev} settings={settings} onSaved={s => { setSettings(s); toast('设置已保存'); }} />}
          </div>
        </div>
      </main>
      <LinkPicker profiles={profiles} running={running} />
    </div>
  );
}

const ProfilesView: React.FC<{
  profiles: Profile[]; proxyName: Record<string, Proxy>; running: Set<string>;
  setRunning: React.Dispatch<React.SetStateAction<Set<string>>>; reload: () => void;
  dbStatus: DBStatus; setDbStatus: (s?: DBStatus) => void;
}> = ({ profiles, proxyName, running, setRunning, reload, dbStatus, setDbStatus }) => {
  const toast = useToast();
  const ask = useConfirm();
  const searchRef = useRef<HTMLInputElement>(null);
  useEffect(() => { const k = (e: KeyboardEvent) => { if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); searchRef.current?.focus(); } }; window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k); }, []);
  const [q, setQ] = useState('');
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [editing, setEditing] = useState<Profile | null>(null);
  const [drawer, setDrawer] = useState(false);
  const [bulk, setBulk] = useState(false);
  const [nativeTools, setNativeTools] = useState(false);
  const [busy, setBusy] = useState<Set<string>>(new Set());

  const [fGroup, setFGroup] = useState('\u0000');
  const [fTag, setFTag] = useState('');
  const [sort, setSort] = useState<'recent' | 'name' | 'created'>('recent');
  const [modal, setModal] = useState<'' | 'group' | 'assign'>('');
  const [shareP, setShareP] = useState<Profile | null>(null); // share
  const [needKernel, setNeedKernel] = useState<Profile | null>(null);
  const [envChk, setEnvChk] = useState<Profile | null>(null); // 环境检测
  const [syncBusy, setSyncBusy] = useState<Set<string>>(new Set());
  const [runs, setRuns] = useState<Record<string, number>>({});
  const [pending, refreshPending] = usePendingRestart([profiles, proxyName, running]);
  const { restart, restarting } = useRestart(useCallback(() => { reload(); refreshPending(); }, [reload, refreshPending]));
  const [, tick] = useState(0);
  useEffect(() => { api.SyncBusy().then((b: any) => setSyncBusy(new Set(b || []))).catch(() => {}); }, []);
  useEffect(() => onEvent('sync:event', (e: any) => {
    if (e.phase === 'busy') setSyncBusy(s => new Set(s).add(e.id));
    if (e.phase === 'idle') setSyncBusy(s => { const n = new Set(s); n.delete(e.id); return n; });
  }), []);
  useEffect(() => { api.Running().then((l: any) => setRuns(Object.fromEntries((l || []).map((r: any) => [r.profileId, r.startedAt])))).catch(() => {}); }, [running]);
  useEffect(() => { if (!running.size) return; const t = setInterval(() => tick(x => x + 1), 30000); return () => clearInterval(t); }, [running.size]);
  const groups = useMemo(() => [...new Set(profiles.map(p => p.group || ''))].sort(), [profiles]);
  const tags = useMemo(() => [...new Set(profiles.flatMap(p => p.tags || []))].sort(), [profiles]);
  // Profiles that share an exit IP with another profile are easy to link.
  const ipShare = useMemo(() => {
    const by: Record<string, number> = {};
    for (const p of profiles) { const ip = proxyName[p.proxyId]?.ip; if (ip) by[ip] = (by[ip] || 0) + 1; }
    return by;
  }, [profiles, proxyName]);
  const list = useMemo(() => {
    const ql = q.toLowerCase();
    const out = profiles.filter(p =>
      (!q || p.name.toLowerCase().includes(ql) || (p.group || '').toLowerCase().includes(ql) || (p.tags || []).some(t => t.toLowerCase().includes(ql)) || (proxyName[p.proxyId]?.ip || '').includes(q))
      && (fGroup === '\u0000' || (p.group || '') === fGroup) && (!fTag || (p.tags || []).includes(fTag)));
    out.sort(sort === 'name' ? (a, b) => a.name.localeCompare(b.name, 'zh') : sort === 'created' ? (a, b) => (b.createdAt || 0) - (a.createdAt || 0) : (a, b) => (b.lastRun || 0) - (a.lastRun || 0));
    return out;
  }, [profiles, q, fGroup, fTag, sort, proxyName]);

  const toggleSel = (id: string) => setSel(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const allSel = list.length > 0 && list.every(p => sel.has(p.id));

  const setBusyFor = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });

  const start = async (p: Profile) => {
    setBusyFor(p.id, true);
    try { await api.StartProfile(p.id); setRunning(s => new Set(s).add(p.id)); toast(`已启动 ${p.name}`); }
    catch (e: any) { const m = String(e?.message || e); if (m.includes('没有找到开源内核')) setNeedKernel(p); else toast(m, 'bad'); }
    finally { setBusyFor(p.id, false); }
  };
  const stop = async (p: Profile) => { setBusyFor(p.id, true); await api.StopProfile(p.id); setRunning(s => { const n = new Set(s); n.delete(p.id); return n; }); setBusyFor(p.id, false); };
  const clone = async (p: Profile) => { await api.CloneProfile(p.id); toast('已复制环境'); reload(); };
  // 修复云端: force re-upload from this device (pull elsewhere reported missing chunks)
  const repairCloud = async (p: Profile) => {
    if (!await ask({ title: `修复「${p.name}」的云端数据？`, body: '用本机的这份环境数据强制重新上传：补传服务器缺少的全部数据分块，并提交一个新的云端版本。请在上传它的那台设备上操作。', ok: '修复云端' })) return;
    setSyncBusy(s => new Set(s).add(p.id));
    try { const r: any = await api.RepairCloud(p.id); toast(r?.message || '已修复', 'ok'); reload(); }
    catch (e: any) { syncErrorToast(p.id, e, toast, ask); }
    finally { setSyncBusy(s => { const n = new Set(s); n.delete(p.id); return n; }); }
  };
  const [delT, setDelT] = useState<DelTarget[] | null>(null);
  const [trashOpen, setTrashOpen] = useState(false); // 最近删除
  const [trashN, refreshTrash] = useTrashCount(profiles.length);
  const trashBtn = <button className={`btn icon trash-btn ${trashN ? '' : 'none'}`} onClick={() => setTrashOpen(true)} title={trashN ? `最近删除 · ${trashN} 个` : '最近删除'} aria-label="最近删除"><Icon.trash s={15} />{trashN > 0 && <span className="trash-n num">{trashN > 99 ? '99+' : trashN}</span>}</button>;
  const trashModal = trashOpen && <TrashModal onClose={() => { setTrashOpen(false); refreshTrash(); }} onChanged={() => { reload(); refreshTrash(); }} />;
  const del = (ids: string[]) => setDelT(ids.map(id => profiles.find(p => p.id === id)).filter(Boolean).map(p => ({ id: p!.id, name: p!.name, inCloud: p!.storage === 'cloud' || p!.cloudVersion > 0, running: running.has(p!.id) })));

  const startBatch = async () => { for (const id of sel) { const p = profiles.find(x => x.id === id); if (p && !running.has(id)) await start(p); } };
  const stopBatch = async () => { const ids = [...sel].filter(id => running.has(id)); if (!ids.length) return; await api.StopProfiles(ids); setRunning(s => { const n = new Set(s); ids.forEach(id => n.delete(id)); return n; }); toast(`已停止 ${ids.length} 个环境`); };
  const pushBatch = async () => { const r: any = await api.PushProfiles([...sel]); toast(r.failed ? `上传 ${r.ok} 个，失败 ${r.failed} 个：${(r.errors || [])[0] || ''}` : `已上传 ${r.ok} 个环境`, r.failed ? 'bad' : 'ok'); reload(); };
  const focus = async (id: string) => { try { await api.FocusProfile(id); } catch (e: any) { toast(String(e?.message || e), 'bad'); } };
  const runningList = profiles.filter(p => running.has(p.id));

  if (profiles.length === 0) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="breathing" size={64} theme="dark" /></div>
      <h3>还没有环境</h3>
      <div>创建第一个浏览器环境，它会拥有一套独立的指纹、代理和数据目录。</div>
      <div className="actions">
        <BorderBeam size="sm" colorVariant="ocean" theme="dark" borderRadius={9}>
          <button className="btn primary" onClick={() => { setEditing(null); setDrawer(true); }}><Icon.plus s={15} /> 新建环境</button>
        </BorderBeam>
        <button className="btn" onClick={() => setBulk(true)}><Icon.layers s={15} /> 批量创建</button>
        <button className="btn" onClick={() => setNativeTools(true)}><Icon.shield s={15} /> 指纹库 / JSON</button>
        {trashN > 0 && <button className="btn" onClick={() => setTrashOpen(true)}><Icon.trash s={15} /> 最近删除 <span className="trash-n inline num">{trashN}</span></button>}
      </div>
      {trashModal}
      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={() => setNativeTools(false)} onChanged={reload} />}
    </div>
  );

  const cloudN = profiles.filter(p => p.storage === 'cloud').length;
  const proxied = profiles.filter(p => p.proxyId).length;

  return (
    <div className={sel.size ? 'has-bulk' : ''} style={{ display: 'contents' }}>
      <div className="stat-row">
        <StatCard l="环境总数" n={profiles.length} ic="layers" pct={100} />
        <StatCard l="运行中" n={running.size} of={profiles.length} ic="bolt" tone="ok" pct={profiles.length ? running.size / profiles.length * 100 : 0} />
        <StatCard l="云端同步" n={cloudN} of={profiles.length} ic="cloud" tone="vi" pct={profiles.length ? cloudN / profiles.length * 100 : 0} />
        <StatCard l="已分配代理" n={proxied} of={profiles.length} ic="globe" tone="cy" pct={profiles.length ? proxied / profiles.length * 100 : 0} />
      </div>

      <div className="toolbar">
        <div className="search"><Icon.search /><input ref={searchRef} type="text" placeholder="搜索名称、分组、标签或 IP" value={q} onChange={e => setQ(e.target.value)} /><kbd>Ctrl K</kbd></div>
        <Pick label="分组" value={fGroup} onChange={setFGroup} icon={<Icon.folder />}
          options={[['\u0000', '全部分组', profiles.length] as [string, string, number], ...groups.map(g => [g, g || '未分组', profiles.filter(p => (p.group || '') === g).length] as [string, string, number])]} />
        {tags.length > 0 && <Pick label="标签" value={fTag} onChange={setFTag} icon={<Icon.grid />}
          options={[['', '全部标签'] as [string, string], ...tags.map(t => [t, t, profiles.filter(p => (p.tags || []).includes(t)).length] as [string, string, number])]} />}
        <Pick label="排序" value={sort} onChange={setSort as any} icon={<Icon.layers />}
          options={[['recent', '最近运行'], ['name', '名称'], ['created', '创建时间']]} />
        {(fGroup !== '\u0000' || fTag) && <><span className="tb-count num">{list.length} / {profiles.length}</span><button className="tb-clear" onClick={() => { setFGroup('\u0000'); setFTag(''); }}>清除</button></>}
        <div className="spacer" />
        {trashBtn}
        <button className="btn" onClick={() => setBulk(true)}><Icon.layers s={15} /> 批量创建</button>
        <button className="btn" onClick={() => setNativeTools(true)}><Icon.shield s={15} /> 指纹库 / JSON</button>
        <BorderBeam size="sm" colorVariant="ocean" theme="dark" borderRadius={9}>
          <button className="btn primary" onClick={() => { setEditing(null); setDrawer(true); }}><Icon.plus s={15} /> 新建环境</button>
        </BorderBeam>
      </div>

      {runningList.length > 0 && <div className="run-strip">
        <span className="lbl"><Icon.bolt /> 运行中 {runningList.length}</span>
        <div className="run-chips">{runningList.map(p => { const px = proxyName[p.proxyId]; const st = runs[p.id]; return <div key={p.id} className="run-chip">
          <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} running />
          <div className="t"><b>{p.name}{pending.has(p.id) && <RestartTag compact />}</b><span className="num">{st ? dur(st) : '启动中'} · {px ? (px.ip || px.host) : '直连'}</span></div>
          {pending.has(p.id) && <button className="ib restart" title="待重启：点击重启使新设置生效" disabled={restarting.has(p.id)} onClick={() => restart(p)}><Icon.refresh /></button>}
          <button className="ib" title="切到前台" onClick={() => focus(p.id)}><Icon.laptop /></button>
          <button className="ib stop" title="停止" onClick={() => stop(p)}><Icon.stop /></button>
        </div>; })}</div>
        <WindowTools running={runningList.filter(p => (p as any).kernel !== 'firefox')} selected={[...sel].filter(id => running.has(id))} bulk={sel.size > 0} />
      </div>}

      <div className="panel list">
        <table>
          <thead><tr>
            <th style={{ width: 44 }}><Chk on={allSel} onClick={() => setSel(allSel ? new Set() : new Set(list.map(p => p.id)))} /></th>
            <th>环境</th><th>指纹</th><th>代理出口</th><th>存储</th><th>最近使用</th><th style={{ textAlign: 'right' }}>操作</th>
          </tr></thead>
          <tbody>
            {list.map((p, i) => {
              const px = proxyName[p.proxyId];
              const isRun = running.has(p.id); const isBusy = busy.has(p.id) || restarting.has(p.id); const needRestart = isRun && pending.has(p.id);
              return <tr key={p.id} className={`rise ${sel.has(p.id) ? 'sel' : ''} ${isRun ? 'running' : ''}`} style={{ ['--i' as any]: i }}>
                <td><Chk on={sel.has(p.id)} onClick={() => toggleSel(p.id)} /></td>
                <td>
                  <div className="cell-id">
                    <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} running={isRun} />
                    <div>
                      <div className="nm"><span className="nm-line"><span className="nm-t">{p.name}</span><DefaultStar profile={p} on={dbStatus.profileId === p.id} onChanged={setDbStatus} /></span>{needRestart && <RestartTag />}</div>
                      {(p as any).shareId ? <div className="sub sh-rowtag"><span className="badge cloud" title={`由 ${(p as any).sharedFrom} 共享，配置只读`}><ShIc.share s={11} /> 共享 · {(p as any).sharedFrom}</span><span className={`sh-left ${shareLeft((p as any).shareExpires)[1]}`}>{shareLeft((p as any).shareExpires)[0]}</span></div> :
                      <div className="sub">{p.group || '未分组'}{(p.tags || []).length ? <span style={{ marginLeft: 6 }}>{(p.tags || []).map(t => <span key={t} className="tag">{t}</span>)}</span> : null}</div>}
                    </div>
                  </div>
                </td>
                <td><div className="os" style={{ fontWeight: 550 }}><OsIcon os={p.fingerprint.os} s={13} />{osLabel(p.fingerprint.os)} <span className="sub" style={{ margin: 0 }}>{(p as any).kernel === 'firefox' ? 'Firefox' : `Chrome ${p.fingerprint.browserVersion}`}</span><span className={`devtag ${(p as any).kernel === 'firefox' ? 'ff' : ''} ${(p as any).kernel === 'chromium' || needsOpen(p.fingerprint.os, (p.fingerprint as any).deviceType) ? 'open' : ''}`} title={(p as any).kernel === 'firefox' ? 'Firefox 内核' : (p as any).kernel === 'chromium' || needsOpen(p.fingerprint.os, (p.fingerprint as any).deviceType) ? '开源内核' : '定制内核'}>{(p as any).kernel === 'firefox' ? <FoxIcon s={11} /> : null}<DevIcon t={devType((p.fingerprint as any).deviceType)} s={11} />{(p.fingerprint as any).deviceName || devName((p.fingerprint as any).deviceType)}</span></div><div className="sub num">{p.fingerprint.screenWidth}×{p.fingerprint.screenHeight} · {p.fingerprint.hardwareConcurrency}C/{p.fingerprint.deviceMemory}G</div></td>
                <td>{px ? <div><CC cc={px.country} /><span>{px.city || px.name}</span>{px.status === 'fail' && <span className="badge fail" style={{ marginLeft: 6 }}>失效</span>}<div className="sub num">{px.ip || px.host}{px.ip && ipShare[px.ip] > 1 && <span className="ip-dup" title="多个环境使用同一出口 IP，容易被平台关联"> ⚠ {ipShare[px.ip]} 个环境共用此 IP</span>}</div></div> : <span className="sub">直连</span>}</td>
                <td>{(() => { const [txt, tone] = syncState(p, syncBusy); return <InlineTransfer id={p.id} fallback={<span className={`badge ${tone}`}>{tone === 'local' ? <Icon.laptop s={12} /> : <Icon.cloud s={12} />} {txt}{p.storage === 'cloud' && <span className="num" style={{ marginLeft: 3 }}>v{p.cloudVersion}</span>}</span>} />; })()}</td>
                <td><span className="sub">{ago(p.lastRun)}</span></td>
                <td>
                  <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end', alignItems: 'center' }}>
                    {needRestart && !isBusy && <button className="btn sm icon restart-btn" onClick={() => restart(p)} title="重启：代理、指纹或启动参数已修改，重启后生效" aria-label="重启"><Icon.refresh s={13} /></button>}
                    {isBusy ? <span style={{ width: 98, display: 'grid', placeItems: 'center' }}><ThinkingOrb state={isRun ? 'breathing' : 'connecting'} size={20} theme="dark" /></span>
                      : <span className="split-btn">{isRun ? <button className="btn sm stop" style={{ width: 70 }} onClick={() => stop(p)}><Icon.stop s={12} /> 停止</button>
                        : <button className="btn sm go" style={{ width: 70 }} onClick={() => start(p)}><Icon.play s={12} /> 启动</button>}
                        <SiteLaunch profile={p} running={isRun} onLaunched={() => setRunning(s => new Set(s).add(p.id))} /></span>}
                    <span className="acts" style={{ display: 'flex', gap: 2 }}>
                      {(p as any).shareId ? <button className="btn ghost icon" title="共享环境的配置为只读" disabled><Icon.edit s={15} /></button>
                        : <button className="btn ghost icon" title="编辑" onClick={() => { setEditing(p); setDrawer(true); }}><Icon.edit s={15} /></button>}
                      <MoreMenu title="更多操作" items={[
                        !(p as any).shareId && { label: '环境检测 / 一键修复', icon: <Icon.shield s={14} />, onClick: () => setEnvChk(p) },
                        !(p as any).shareId && { label: '分享给其他用户', icon: <ShIc.share s={14} />, onClick: () => setShareP(p) },
                        !(p as any).shareId && { label: '复制环境', icon: <Icon.copy s={14} />, onClick: () => clone(p) },
                        !(p as any).shareId && p.storage === 'cloud' && { label: '修复云端（强制重新上传）', icon: <Icon.upload s={14} />, onClick: () => repairCloud(p) },
                        !(p as any).shareId && 'sep',
                        { label: '删除', icon: <Icon.trash s={14} />, danger: true, onClick: () => del([p.id]) },
                      ]} />
                    </span>
                  </div>
                </td>
              </tr>;
            })}
          </tbody>
        </table>
        {list.length === 0 && <div className="empty" style={{ padding: 50 }}><Icon.search s={28} /><div>没有匹配「{q}」的环境</div></div>}
      </div>

      {sel.size > 0 && <div className="bulkbar">
        <span className="cnt">已选 <b className="num">{sel.size}</b> 个</span>
        <button className="btn sm go" onClick={startBatch}><Icon.play s={12} /> 启动</button>
        <button className="btn sm" onClick={stopBatch}><Icon.stop s={12} /> 停止</button>
        <span className="sep" />
        <button className="btn sm ghost" onClick={() => setModal('group')}><Icon.folder s={13} /> 分组</button>
        <button className="btn sm ghost" onClick={() => setModal('assign')}><Icon.globe s={13} /> 代理</button>
        <button className="btn sm ghost" onClick={pushBatch}><Icon.upload s={13} /> 同步</button>
        <span className="sep" />
        <button className="btn sm ghost danger" onClick={() => del([...sel])}><Icon.trash s={13} /></button>
        <button className="btn ghost icon" title="取消选择" onClick={() => setSel(new Set())}><Icon.x /></button>
      </div>}
      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload}
        runningNow={!!editing && running.has(editing.id)} pendingRestart={!!editing && pending.has(editing.id)} restarting={!!editing && restarting.has(editing.id)} onRestart={editing ? () => restart(editing) : undefined} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={() => setNativeTools(false)} onChanged={reload} />}
      {delT && delT.length > 0 && <DeleteModal targets={delT} onClose={() => setDelT(null)} onDone={() => { setSel(new Set()); reload(); refreshTrash(); }} />}
      {trashModal}
      {modal === 'group' && <GroupModal ids={[...sel]} groups={groups} onClose={() => setModal('')} onDone={reload} />}
      {envChk && <EnvCheckModal profileId={envChk.id} name={envChk.name} onClose={() => setEnvChk(null)} onChanged={reload} />}
      {needKernel && <KernelMissingModal name={needKernel.name} onClose={() => setNeedKernel(null)} onLaunch={() => start(needKernel)} />}
      {shareP && <ShareDialog profile={shareP} running={running.has(shareP.id)} onClose={() => setShareP(null)} onSecurity={() => window.dispatchEvent(new CustomEvent('gb:goto-cloud', { detail: 'security' }))} />}
      {modal === 'assign' && <AssignModal ids={[...sel]} proxies={Object.values(proxyName)} profiles={profiles} onClose={() => setModal('')} onDone={reload} />}
    </div>
  );
};

const dur = (since: number) => { const m = Math.max(0, Math.floor((Date.now() / 1000 - since) / 60)); return m < 60 ? `${m} 分钟` : `${Math.floor(m / 60)} 小时 ${m % 60} 分`; };

const GroupModal: React.FC<{ ids: string[]; groups: string[]; onClose: () => void; onDone: () => void }> = ({ ids, groups, onClose, onDone }) => {
  const toast = useToast();
  const [group, setGroup] = useState('');
  const [keep, setKeep] = useState(false);
  const [tag, setTag] = useState('');
  const [out, close] = useExit(onClose);
  const go = async () => { const n = await api.SetProfilesGroup(ids, keep ? '\u0000' : group, tag); toast(`已更新 ${n} 个环境`); onDone(); close(); };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal">
        <div className="dhead"><div className="glyph"><Icon.folder s={18} /></div><div><h3>分组与标签</h3><div className="sub">应用到选中的 {ids.length} 个环境</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <Field label="移动到分组" hint="留空 = 未分组">
            <input type="text" list="gb-groups" value={group} disabled={keep} onChange={e => setGroup(e.target.value)} placeholder="输入或选择分组" />
            <datalist id="gb-groups">{groups.filter(Boolean).map(g => <option key={g} value={g} />)}</datalist>
          </Field>
          <div className="assign-filters" style={{ marginTop: -4, marginBottom: 12 }}><button className={`chip-tg ${keep ? 'on' : ''}`} onClick={() => setKeep(v => !v)}>不修改分组，只加标签</button></div>
          <Field label="添加标签（可选）"><input type="text" value={tag} onChange={e => setTag(e.target.value)} /></Field>
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" onClick={go}><Icon.check s={15} /> 应用</button></div>
      </div>
    </div>
  );
};

const AssignModal: React.FC<{ ids: string[]; proxies: Proxy[]; profiles: Profile[]; onClose: () => void; onDone: () => void }> = ({ ids, proxies, profiles, onClose, onDone }) => {
  const toast = useToast();
  const [q, setQ] = useState('');
  const [onlyOk, setOnlyOk] = useState(true);
  const [onlyFree, setOnlyFree] = useState(false);
  const [pick, setPick] = useState<Set<string>>(new Set());
  const [mode, setMode] = useState<'order' | 'random' | 'one'>('order');
  const [group, setGroup] = useState('*');
  const [subs] = useSubs();
  const [out, close] = useExit(onClose);
  const used = useMemo(() => new Set(profiles.filter(p => !ids.includes(p.id)).map(p => p.proxyId).filter(Boolean)), [profiles, ids]);
  const shown = proxies.filter(p => (group === '*' || groupKey(p) === group) && (!onlyOk || p.status === 'ok') && (!onlyFree || !used.has(p.id)) && (!q || `${p.name} ${p.host} ${p.ip} ${p.country} ${p.tag} ${p.subName || ''}`.toLowerCase().includes(q.toLowerCase())));
  const toggle = (id: string) => setPick(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const go = async (direct = false) => {
    try { const n = await api.AssignProxies(ids, direct ? [] : shown.filter(p => pick.has(p.id)).map(p => p.id), mode); toast(direct ? `${n} 个环境改为直连` : `已为 ${n} 个环境分配代理`); onDone(); close(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const nPick = shown.filter(p => pick.has(p.id)).length;
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 640 }}>
        <div className="dhead"><div className="glyph"><Icon.globe s={18} /></div><div><h3>批量分配代理</h3><div className="sub">选中的 {ids.length} 个环境</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="row" style={{ gap: 8, marginBottom: 8 }}>
            <input type="text" placeholder="搜索名称、IP、国家、标签" value={q} onChange={e => setQ(e.target.value)} />
            <button className="btn sm ghost" style={{ flex: 'none' }} onClick={() => setPick(new Set(shown.map(p => p.id)))}>全选</button>
            <button className="btn sm ghost" style={{ flex: 'none' }} onClick={() => setPick(new Set())}>清空</button>
          </div>
          <div className="assign-filters">
            <button className={`chip-tg ${onlyOk ? 'on' : ''}`} onClick={() => setOnlyOk(v => !v)}>仅可用</button>
            <button className={`chip-tg ${onlyFree ? 'on' : ''}`} onClick={() => setOnlyFree(v => !v)}>仅未被占用</button>
            {(subs.length > 0 || proxies.some(p => p.subId)) && <><span className="spacer" /><Pick label="分组" value={group} onChange={g => { setGroup(g); if (g !== '*') setPick(new Set(proxies.filter(p => groupKey(p) === g && (!onlyOk || p.status === 'ok') && (!onlyFree || !used.has(p.id))).map(p => p.id))); }} icon={<Icon.layers />} options={groupOptions(proxies, subs)} align="right" /></>}
          </div>
          <div className="assign-list">
            {shown.map(p => <div key={p.id} className={`assign-row ${pick.has(p.id) ? 'on' : ''}`} onClick={() => toggle(p.id)}>
              <Chk on={pick.has(p.id)} onClick={() => {}} />
              <CC cc={p.country} /><span className="nm" style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.name}</span>
              <span className={`px-pick-proto ${isCore(p) ? 'core' : ''}`}>{protoOf(p)}</span>
              <span className="sub num">{p.ip || `${p.host}:${p.port}`}</span>
              {used.has(p.id) && <span className="badge warn">已被使用</span>}
            </div>)}
            {shown.length === 0 && <div className="sub" style={{ padding: 16, textAlign: 'center' }}>没有符合条件的代理</div>}
          </div>
          <Field label="分配方式" hint={mode === 'order' ? (group !== '*' && group ? '按所选分组的节点依次轮流：第 1 个环境用第 1 个节点' : '第 1 个环境用第 1 个代理，依次轮流') : mode === 'random' ? '随机分配，代理数量足够时不重复' : '所有环境都用第一个选中的代理'}>
            <Segmented className="block" items={[['order', '顺序轮流'], ['random', '随机'], ['one', '统一一个']]} value={mode} onChange={setMode} />
          </Field>
          {nPick > 0 && nPick < ids.length && mode !== 'one' && <div className="sub" style={{ color: '#ffd98a' }}>⚠ 代理数（{nPick}）少于环境数（{ids.length}），会有环境共用同一个代理。</div>}
        </div>
        <div className="dfoot"><button className="btn ghost" onClick={() => go(true)}>改为直连</button><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" disabled={!nPick} onClick={() => go()}><Icon.check s={15} /> 分配 {nPick} 个代理</button></div>
      </div>
    </div>
  );
};

const Chk: React.FC<{ on: boolean; onClick: () => void }> = ({ on, onClick }) =>
  <span className={`chk ${on ? 'on' : ''}`} onClick={onClick}><Icon.check s={11} /></span>;

const BulkModal: React.FC<{ proxies: Proxy[]; onClose: () => void; onDone: () => void }> = ({ proxies, onClose, onDone }) => {
  const toast = useToast();
  const [prefix, setPrefix] = useState('Env');
  const [os, setOs] = useState('windows');
  const [dt, setDt] = useState('desktop');
  const dtInfo = DEVICE_TYPES.find(d => d.id === dt)!;
  const pickType = (t: string) => { setDt(t); const i = DEVICE_TYPES.find(d => d.id === t)!; if (!i.oses.includes(os)) setOs(i.oses[0]); };
  const [count, setCount] = useState(10);
  const [group, setGroup] = useState('');
  const [proxyId, setProxyId] = useState('');
  const [busy, setBusy] = useState(false);
  const [kernel, setKernel] = useState<'' | 'firefox'>('');
  const [ffInfo] = useFirefoxInfo();
  const ffOK = dt === 'desktop' || dt === 'laptop';
  const [out, close] = useExit(onClose);
  const go = async () => { setBusy(true); await api.CreateTypedProfilesKernel(prefix, dt, os, count, proxyId, group, ffOK ? kernel : ''); toast(`已创建 ${count} 个${kernel === 'firefox' && ffOK ? ' Firefox ' : ''}环境`); setBusy(false); onDone(); close(); };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <BorderBeam size="pulse-inner" colorVariant="ocean" theme="dark" strength={0.6} borderRadius={20}>
      <div className="modal">
        <div className="dhead"><div className="glyph"><Icon.layers s={18} /></div><div><h3>批量创建环境</h3><div className="sub">每个环境都会生成一套独立且自洽的指纹</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="grid2">
            <Field label="名称前缀"><input type="text" value={prefix} onChange={e => setPrefix(e.target.value)} /></Field>
            <Field label="数量"><input type="number" min={1} max={500} value={count} onChange={e => setCount(+e.target.value)} /></Field>
          </div>
          <div className="grid2">
            <Field label="设备类型"><select value={dt} onChange={e => pickType(e.target.value)}>{DEVICE_TYPES.map(d => <option key={d.id} value={d.id}>{d.name}</option>)}</select></Field>
            <Field label="操作系统"><select value={os} onChange={e => setOs(e.target.value)}>{dtInfo.oses.map(o => <option key={o} value={o}>{osNameOf(o)}</option>)}</select></Field>
          </div>
          <Field label="内核">
            <div className="bk-kernel">
              <button type="button" className={`bk-k ${kernel === '' || !ffOK ? 'on' : ''}`} onClick={() => setKernel('')}><Icon.cpu s={16} /><span><b>Chromium</b><small>{needsOpen(os, dt) ? '开源内核（自动）' : '定制内核 / 开源内核（自动）'}</small></span><i className="radio" /></button>
              <button type="button" className={`bk-k ${kernel === 'firefox' && ffOK ? 'on' : ''} ${ffOK ? '' : 'off'}`} disabled={!ffOK} title={ffOK ? '' : 'Firefox 内核只支持桌面电脑和笔记本'} onClick={() => setKernel('firefox')}><FoxIcon s={16} /><span><b>Firefox</b><small className={ffInfo && !ffInfo.path ? 'warn' : ''}>{ffOK ? (ffInfo && !ffInfo.path ? '未安装，启动前请在设置中安装' : ffVersionLabel(ffInfo)) : '只支持桌面设备'}</small></span><i className="radio" /></button>
            </div>
          </Field>
          {kernel !== 'firefox' && needsOpen(os, dt) && <div className="sub" style={{ marginBottom: 10 }}>这些环境会用开源内核启动{dt === 'tv' ? '；电视可信度低，仅建议实验' : ''}。</div>}
          <div className="grid2">
            <Field label="分组"><input type="text" value={group} onChange={e => setGroup(e.target.value)} /></Field>
          </div>
          <Field label="统一代理" hint="每个环境仍会生成各自独立的指纹">
            <select value={proxyId} onChange={e => setProxyId(e.target.value)}><option value="">不分配</option>{proxies.map(p => <option key={p.id} value={p.id}>{flag(p.country)} · {p.name}</option>)}</select>
          </Field>
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" onClick={go} disabled={busy}><Icon.sparkles s={15} /> {busy ? '创建中…' : `创建 ${count} 个`}</button></div>
      </div>
      </BorderBeam>
    </div>
  );
};

const osLabel = (o: string) => osNameOf(o);

export default function App() { return <ToastHost><Shell /><SyncPrompt /><TransferDock /><EnvSetup onReady={() => {}} /><Splash /></ToastHost>; }
