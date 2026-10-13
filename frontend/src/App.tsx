import React, { useEffect, useMemo, useState, useCallback, useLayoutEffect, useRef } from 'react';
import { BorderBeam } from 'border-beam';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Proxy, Settings, onEvent } from './api';
import { MetalFx } from 'metal-fx';
import { Icon, Field, flag, ago, useToast, useConfirm, useExit, ToastHost, WindowControls, useMaximised, StatCard, Avatar, CC, CountUp, OsIcon, Ambient, Splash, useSpotlight } from './ui';
import { ProfileDrawer } from './ProfileDrawer';
import { ProxyView } from './ProxyView';
import { SettingsView } from './SettingsView';
import { CloudView } from './CloudView';
import { NativeTools } from './NativeTools';

type Tab = 'profiles' | 'proxies' | 'cloud' | 'settings';

function Shell() {
  const toast = useToast();
  const [tab, setTab] = useState<Tab>('profiles');
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [running, setRunning] = useState<Set<string>>(new Set());
  const [settings, setSettings] = useState<Settings | null>(null);
  const max = useMaximised();

  const reload = useCallback(async () => {
    setProfiles(await api.ListProfiles());
    setProxies(await api.ListProxies());
    const r = await api.Running();
    setRunning(new Set(r.map(x => x.profileId)));
  }, []);

  useEffect(() => { reload(); api.GetSettings().then(setSettings); }, [reload]);
  useEffect(() => onEvent('profile:exit', (id: string) => { setRunning(s => { const n = new Set(s); n.delete(id); return n; }); }), []);

  const proxyName = useMemo(() => Object.fromEntries(proxies.map(p => [p.id, p])), [proxies]);

  const titles: Record<Tab, [string, string]> = {
    profiles: ['浏览器环境', '每个环境拥有独立指纹、代理与数据目录'],
    proxies: ['代理管理', '导入、检测并为环境分配代理'],
    cloud: ['云端同步', '把环境加密同步到你自己的 VPS'],
    settings: ['设置', '内核、本地 API、同步服务器'],
  };

  const nav: [Tab, keyof typeof Icon, string, number | null][] = [
    ['profiles', 'layers', '环境', profiles.length],
    ['proxies', 'globe', '代理', proxies.length],
    ['cloud', 'cloud', '云同步', null],
    ['settings', 'settings', '设置', null],
  ];

  const navRef = useRef<HTMLDivElement>(null);
  const [pillY, setPillY] = useState(0);
  useLayoutEffect(() => {
    const el = navRef.current?.querySelector<HTMLButtonElement>(`button[data-t="${tab}"]`);
    if (el) setPillY(el.offsetTop);
  }, [tab]);
  useSpotlight();
  // Ctrl/Cmd+1..4 switches sections
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if ((e.ctrlKey || e.metaKey) && /^[1-4]$/.test(e.key)) { e.preventDefault(); setTab(nav[+e.key - 1][0]); } };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  return (
    <div className={`app ${max ? 'max' : ''}`}>
      <Ambient />
      <WindowControls max={max} />
      <aside className="side">
        <div className="brand">
          <MetalFx preset="chromatic" theme="dark" borderRadius={10} style={{ width: 36, height: 36 }}><div className="logo"><Icon.ghost s={20} /></div></MetalFx>
          <div><h1>Ghost<b>Browser</b></h1><span>指纹浏览器 · 群控管理</span></div>
        </div>
        <div className="nav-label">工作台</div>
        <nav className="nav" ref={navRef}>
          <span className="pill" style={{ transform: `translateY(${pillY}px)` }} />
          {nav.map(([t, ic, label, count]) => {
            const I = Icon[ic];
            return <button key={t} data-t={t} className={tab === t ? 'active' : ''} onClick={() => setTab(t)}>
              <I /> {label} {count != null && <span className="count num">{count}</span>}
            </button>;
          })}
        </nav>
        <div className="spacer" />
        <div className="status-card spot">
          <div className="row1"><span className="live" /> 本地 API 运行中</div>
          <div className="row2"><span>端口</span><b className="num">:{settings?.apiPort ?? '—'}</b></div>
          <div className="row2"><span>内核</span><b>{settings?.kernelMode === 'custom' ? '自研内核' : '标准 Chrome'}</b></div>
          <div className="row2"><span>运行中</span><b className="num">{running.size} / {profiles.length}</b></div>
        </div>
      </aside>

      <main className="main">
        <div className="topbar">
          <div key={tab} className="page">
            <h2>{titles[tab][0]}</h2>
            <div className="sub">{titles[tab][1]}</div>
          </div>
        </div>
        <div className="content">
          <div key={tab} className="page">
            {tab === 'profiles' && <ProfilesView {...{ profiles, proxyName, running, setRunning, reload }} />}
            {tab === 'proxies' && <ProxyView proxies={proxies} reload={reload} />}
            {tab === 'cloud' && <CloudView profiles={profiles} settings={settings} reload={reload} />}
            {tab === 'settings' && settings && <SettingsView settings={settings} onSaved={s => { setSettings(s); toast('设置已保存'); }} />}
          </div>
        </div>
      </main>
    </div>
  );
}

const ProfilesView: React.FC<{
  profiles: Profile[]; proxyName: Record<string, Proxy>; running: Set<string>;
  setRunning: React.Dispatch<React.SetStateAction<Set<string>>>; reload: () => void;
}> = ({ profiles, proxyName, running, setRunning, reload }) => {
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

  const list = useMemo(() => profiles.filter(p =>
    !q || p.name.toLowerCase().includes(q.toLowerCase()) || (p.group || '').toLowerCase().includes(q.toLowerCase()) || (p.tags || []).some(t => t.toLowerCase().includes(q.toLowerCase()))
  ), [profiles, q]);

  const toggleSel = (id: string) => setSel(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const allSel = list.length > 0 && list.every(p => sel.has(p.id));

  const setBusyFor = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });

  const start = async (p: Profile) => {
    setBusyFor(p.id, true);
    try { await api.StartProfile(p.id); setRunning(s => new Set(s).add(p.id)); toast(`已启动 ${p.name}`); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setBusyFor(p.id, false); }
  };
  const stop = async (p: Profile) => { setBusyFor(p.id, true); await api.StopProfile(p.id); setRunning(s => { const n = new Set(s); n.delete(p.id); return n; }); setBusyFor(p.id, false); };
  const clone = async (p: Profile) => { await api.CloneProfile(p.id); toast('已复制环境'); reload(); };
  const del = async (ids: string[]) => { if (!await ask({ title: `删除 ${ids.length} 个环境？`, body: '环境的数据目录（Cookie、缓存、扩展状态）会一并删除，无法恢复。云端副本不受影响。', ok: '删除', danger: true })) return; await api.DeleteProfiles(ids); setSel(new Set()); toast('已删除'); reload(); };

  const startBatch = async () => { for (const id of sel) { const p = profiles.find(x => x.id === id); if (p && !running.has(id)) await start(p); } };

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
      </div>
      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={() => setNativeTools(false)} onChanged={reload} />}
    </div>
  );

  const cloudN = profiles.filter(p => p.storage === 'cloud').length;
  const proxied = profiles.filter(p => p.proxyId).length;

  return (
    <>
      <div className="stat-row">
        <StatCard l="环境总数" n={profiles.length} ic="layers" pct={100} />
        <StatCard l="运行中" n={running.size} of={profiles.length} ic="bolt" tone="ok" pct={profiles.length ? running.size / profiles.length * 100 : 0} />
        <StatCard l="云端同步" n={cloudN} of={profiles.length} ic="cloud" tone="vi" pct={profiles.length ? cloudN / profiles.length * 100 : 0} />
        <StatCard l="已分配代理" n={proxied} of={profiles.length} ic="globe" tone="cy" pct={profiles.length ? proxied / profiles.length * 100 : 0} />
      </div>

      <div className="toolbar">
        <div className="search"><Icon.search /><input ref={searchRef} type="text" placeholder="搜索名称、分组或标签" value={q} onChange={e => setQ(e.target.value)} /><kbd>Ctrl K</kbd></div>
        {sel.size > 0 && <div className="selbar">已选 <b className="num">{sel.size}</b>
          <button className="btn sm go" onClick={startBatch}><Icon.play s={12} /> 启动</button>
          <button className="btn sm danger" onClick={() => del([...sel])}><Icon.trash s={13} /> 删除</button>
          <button className="btn sm ghost" onClick={() => setSel(new Set())}>取消</button>
        </div>}
        <div className="spacer" />
        <button className="btn" onClick={() => setBulk(true)}><Icon.layers s={15} /> 批量创建</button>
        <button className="btn" onClick={() => setNativeTools(true)}><Icon.shield s={15} /> 指纹库 / JSON</button>
        <BorderBeam size="sm" colorVariant="ocean" theme="dark" borderRadius={9}>
          <button className="btn primary" onClick={() => { setEditing(null); setDrawer(true); }}><Icon.plus s={15} /> 新建环境</button>
        </BorderBeam>
      </div>

      <div className="panel list">
        <table>
          <thead><tr>
            <th style={{ width: 44 }}><Chk on={allSel} onClick={() => setSel(allSel ? new Set() : new Set(list.map(p => p.id)))} /></th>
            <th>环境</th><th>指纹</th><th>代理出口</th><th>存储</th><th>最近使用</th><th style={{ textAlign: 'right' }}>操作</th>
          </tr></thead>
          <tbody>
            {list.map((p, i) => {
              const px = proxyName[p.proxyId];
              const isRun = running.has(p.id); const isBusy = busy.has(p.id);
              return <tr key={p.id} className={`rise ${sel.has(p.id) ? 'sel' : ''} ${isRun ? 'running' : ''}`} style={{ ['--i' as any]: i }}>
                <td><Chk on={sel.has(p.id)} onClick={() => toggleSel(p.id)} /></td>
                <td>
                  <div className="cell-id">
                    <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} running={isRun} />
                    <div>
                      <div className="nm">{p.name}</div>
                      <div className="sub">{p.group || '未分组'}{(p.tags || []).length ? <span style={{ marginLeft: 6 }}>{(p.tags || []).map(t => <span key={t} className="tag">{t}</span>)}</span> : null}</div>
                    </div>
                  </div>
                </td>
                <td><div className="os" style={{ fontWeight: 550 }}><OsIcon os={p.fingerprint.os} s={13} />{osLabel(p.fingerprint.os)} <span className="sub" style={{ margin: 0 }}>Chrome {p.fingerprint.browserVersion}</span></div><div className="sub num">{p.fingerprint.screenWidth}×{p.fingerprint.screenHeight} · {p.fingerprint.hardwareConcurrency}C/{p.fingerprint.deviceMemory}G</div></td>
                <td>{px ? <div><CC cc={px.country} /><span>{px.city || px.name}</span><div className="sub num">{px.ip || px.host}</div></div> : <span className="sub">直连</span>}</td>
                <td><span className={`badge ${p.storage}`}>{p.storage === 'cloud' ? <><Icon.cloud s={12} /> 云端 <span className="num">v{p.cloudVersion}</span></> : <><Icon.laptop s={12} /> 本地</>}</span></td>
                <td><span className="sub">{ago(p.lastRun)}</span></td>
                <td>
                  <div style={{ display: 'flex', gap: 4, justifyContent: 'flex-end', alignItems: 'center' }}>
                    {isBusy ? <span style={{ width: 74, display: 'grid', placeItems: 'center' }}><ThinkingOrb state={isRun ? 'breathing' : 'connecting'} size={20} theme="dark" /></span>
                      : isRun ? <button className="btn sm stop" style={{ width: 74 }} onClick={() => stop(p)}><Icon.stop s={12} /> 停止</button>
                        : <button className="btn sm go" style={{ width: 74 }} onClick={() => start(p)}><Icon.play s={12} /> 启动</button>}
                    <span className="acts" style={{ display: 'flex', gap: 2 }}>
                      <button className="btn ghost icon" title="编辑" onClick={() => { setEditing(p); setDrawer(true); }}><Icon.edit s={15} /></button>
                      <button className="btn ghost icon" title="复制" onClick={() => clone(p)}><Icon.copy s={15} /></button>
                      <button className="btn ghost icon danger" title="删除" onClick={() => del([p.id])}><Icon.trash s={15} /></button>
                    </span>
                  </div>
                </td>
              </tr>;
            })}
          </tbody>
        </table>
        {list.length === 0 && <div className="empty" style={{ padding: 50 }}><Icon.search s={28} /><div>没有匹配「{q}」的环境</div></div>}
      </div>

      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={() => setNativeTools(false)} onChanged={reload} />}
    </>
  );
};

const Chk: React.FC<{ on: boolean; onClick: () => void }> = ({ on, onClick }) =>
  <span className={`chk ${on ? 'on' : ''}`} onClick={onClick}><Icon.check s={11} /></span>;

const BulkModal: React.FC<{ proxies: Proxy[]; onClose: () => void; onDone: () => void }> = ({ proxies, onClose, onDone }) => {
  const toast = useToast();
  const [prefix, setPrefix] = useState('Env');
  const [os, setOs] = useState('windows');
  const [count, setCount] = useState(10);
  const [group, setGroup] = useState('');
  const [proxyId, setProxyId] = useState('');
  const [busy, setBusy] = useState(false);
  const [out, close] = useExit(onClose);
  const go = async () => { setBusy(true); await api.CreateProfiles(prefix, os, count, proxyId, group); toast(`已创建 ${count} 个环境`); setBusy(false); onDone(); close(); };
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
            <Field label="操作系统"><select value={os} onChange={e => setOs(e.target.value)}><option value="windows">Windows</option><option value="macos">macOS</option><option value="linux">Linux</option></select></Field>
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

const osLabel = (o: string) => ({ windows: 'Windows', macos: 'macOS', linux: 'Linux' } as any)[o] || o;

export default function App() { return <ToastHost><Shell /><Splash /></ToastHost>; }
