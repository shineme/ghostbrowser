import React, { useEffect, useMemo, useState, useCallback } from 'react';
import { BorderBeam } from 'border-beam';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Proxy, Settings, onEvent } from './api';
import { MetalFx } from 'metal-fx';
import { Icon, Field, flag, ago, useToast, ToastHost } from './ui';
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

  return (
    <div className="app">
      <aside className="side">
        <div className="brand">
          <MetalFx preset="chromatic" theme="dark" borderRadius={9} style={{ width: 34, height: 34 }}><div className="logo"><Icon.ghost s={20} /></div></MetalFx>
          <div><h1>GhostBrowser</h1><span>指纹浏览器 · 群控管理</span></div>
        </div>
        <nav className="nav">
          {nav.map(([t, ic, label, count]) => {
            const I = Icon[ic];
            return <button key={t} className={tab === t ? 'active' : ''} onClick={() => setTab(t)}>
              <I /> {label} {count != null && <span className="count">{count}</span>}
            </button>;
          })}
        </nav>
        <div className="spacer" />
        <div className="foot">
          <span className="dot">●</span> 本地 API :{settings?.apiPort ?? '—'}<br />
          内核 {settings?.kernelMode === 'custom' ? '自研内核' : '标准 Chrome'}
        </div>
      </aside>

      <main className="main">
        <div className="topbar">
          <div>
            <h2>{titles[tab][0]}</h2>
            <div className="sub">{titles[tab][1]}</div>
          </div>
        </div>
        <div className="content">
          {tab === 'profiles' && <ProfilesView {...{ profiles, proxyName, running, setRunning, reload }} />}
          {tab === 'proxies' && <ProxyView proxies={proxies} reload={reload} />}
          {tab === 'cloud' && <CloudView profiles={profiles} settings={settings} reload={reload} />}
          {tab === 'settings' && settings && <SettingsView settings={settings} onSaved={s => { setSettings(s); toast('设置已保存'); }} />}
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
  const [q, setQ] = useState('');
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [editing, setEditing] = useState<Profile | null>(null);
  const [drawer, setDrawer] = useState(false);
  const [bulk, setBulk] = useState(false);
  const [nativeTools,setNativeTools]=useState(false);
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
  const del = async (ids: string[]) => { if (!confirm(`确认删除 ${ids.length} 个环境？数据目录会一并删除。`)) return; await api.DeleteProfiles(ids); setSel(new Set()); toast('已删除'); reload(); };

  const startBatch = async () => { for (const id of sel) { const p = profiles.find(x => x.id === id); if (p && !running.has(id)) await start(p); } };

  if (profiles.length === 0) return (
    <div className="empty">
      <div className="orb"><ThinkingOrb state="breathing" size={64} theme="dark" /></div>
      <h3>还没有环境</h3>
      <div>创建第一个浏览器环境，它会拥有一套独立的指纹、代理和数据目录。</div>
      <div style={{ display: 'flex', gap: 10 }}>
        <button className="btn primary" onClick={() => { setEditing(null); setDrawer(true); }}><Icon.plus s={15} /> 新建环境</button>
        <button className="btn" onClick={() => setBulk(true)}><Icon.layers s={15} /> 批量创建</button>
        <button className="btn" onClick={()=>setNativeTools(true)}>指纹库 / JSON</button>
      </div>
      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={()=>setNativeTools(false)} onChanged={reload}/>}
    </div>
  );

  return (
    <>
      <div className="toolbar">
        <div className="search"><Icon.search /><input type="text" placeholder="搜索名称 / 分组 / 标签" value={q} onChange={e => setQ(e.target.value)} /></div>
        <button className="btn sm" onClick={startBatch} disabled={!sel.size}><Icon.play s={13} /> 启动所选</button>
        <button className="btn sm danger" onClick={() => del([...sel])} disabled={!sel.size}><Icon.trash s={13} /> 删除</button>
        <div className="spacer" />
        <button className="btn" onClick={() => setBulk(true)}><Icon.layers s={15} /> 批量创建</button>
        <button className="btn" onClick={()=>setNativeTools(true)}>指纹库 / JSON</button>
        <BorderBeam size="sm" colorVariant="ocean" theme="dark" borderRadius={8}>
          <button className="btn primary" onClick={() => { setEditing(null); setDrawer(true); }}><Icon.plus s={15} /> 新建环境</button>
        </BorderBeam>
      </div>

      <div className="panel" style={{ overflow: 'hidden' }}>
        <table>
          <thead><tr>
            <th style={{ width: 34 }}><Chk on={allSel} onClick={() => setSel(allSel ? new Set() : new Set(list.map(p => p.id)))} /></th>
            <th>环境</th><th>指纹</th><th>代理</th><th>存储</th><th>最近使用</th><th style={{ textAlign: 'right' }}>操作</th>
          </tr></thead>
          <tbody>
            {list.map(p => {
              const px = proxyName[p.proxyId];
              const isRun = running.has(p.id); const isBusy = busy.has(p.id);
              return <tr key={p.id} className={sel.has(p.id) ? 'sel' : ''}>
                <td><Chk on={sel.has(p.id)} onClick={() => toggleSel(p.id)} /></td>
                <td>
                  <div className="nm">{isRun && <span className="dot-run" style={{ marginRight: 7 }} />}{p.name}</div>
                  <div className="sub">{p.group || '未分组'}{(p.tags || []).length ? ' · ' : ''}{(p.tags || []).map(t => <span key={t} className="tag">{t}</span>)}</div>
                </td>
                <td><div className="nm" style={{ fontWeight: 500 }}>{osLabel(p.fingerprint.os)} · {p.fingerprint.browserVersion}</div><div className="sub">{p.fingerprint.screenWidth}×{p.fingerprint.screenHeight} · {p.fingerprint.hardwareConcurrency}核</div></td>
                <td>{px ? <><span className="flag">{flag(px.country)}</span> <span className="sub">{px.city || px.name}</span></> : <span className="sub">直连</span>}</td>
                <td><span className={`badge ${p.storage}`}>{p.storage === 'cloud' ? <><Icon.cloud s={12} /> 云端 v{p.cloudVersion}</> : <>本地</>}</span></td>
                <td><span className="sub">{ago(p.lastRun)}</span></td>
                <td>
                  <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                    {isBusy ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state={isRun ? 'breathing' : 'connecting'} size={20} theme="dark" /></span>
                      : isRun ? <button className="btn sm danger" onClick={() => stop(p)}><Icon.stop s={13} /> 停止</button>
                        : <button className="btn sm" onClick={() => start(p)}><Icon.play s={13} /> 启动</button>}
                    <button className="btn ghost icon" title="编辑" onClick={() => { setEditing(p); setDrawer(true); }}><Icon.edit s={15} /></button>
                    <button className="btn ghost icon" title="复制" onClick={() => clone(p)}><Icon.copy s={15} /></button>
                    <button className="btn ghost icon danger" title="删除" onClick={() => del([p.id])}><Icon.trash s={15} /></button>
                  </div>
                </td>
              </tr>;
            })}
          </tbody>
        </table>
      </div>

      {drawer && <ProfileDrawer profile={editing} proxies={Object.values(proxyName)} onClose={() => setDrawer(false)} onSaved={reload} />}
      {bulk && <BulkModal proxies={Object.values(proxyName)} onClose={() => setBulk(false)} onDone={reload} />}
      {nativeTools && <NativeTools profiles={profiles} onClose={()=>setNativeTools(false)} onChanged={reload}/>}
    </>
  );
};

const Chk: React.FC<{ on: boolean; onClick: () => void }> = ({ on, onClick }) =>
  <span className={`chk ${on ? 'on' : ''}`} onClick={onClick}>{on && <Icon.check s={11} />}</span>;

const BulkModal: React.FC<{ proxies: Proxy[]; onClose: () => void; onDone: () => void }> = ({ proxies, onClose, onDone }) => {
  const toast = useToast();
  const [prefix, setPrefix] = useState('Env');
  const [os, setOs] = useState('windows');
  const [count, setCount] = useState(10);
  const [group, setGroup] = useState('');
  const [proxyId, setProxyId] = useState('');
  const [busy, setBusy] = useState(false);
  const go = async () => { setBusy(true); await api.CreateProfiles(prefix, os, count, proxyId, group); toast(`已创建 ${count} 个环境`); setBusy(false); onDone(); onClose(); };
  return (
    <div className="overlay center" onMouseDown={e => e.target === e.currentTarget && onClose()}>
      <BorderBeam size="pulse-inner" colorVariant="ocean" theme="dark" strength={0.6} borderRadius={12}>
      <div className="modal">
        <div className="dhead"><h3>批量创建环境</h3><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={onClose}><Icon.x /></button></div></div>
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
            <select value={proxyId} onChange={e => setProxyId(e.target.value)}><option value="">不分配</option>{proxies.map(p => <option key={p.id} value={p.id}>{flag(p.country)} {p.name}</option>)}</select>
          </Field>
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={onClose}>取消</button><button className="btn primary" onClick={go} disabled={busy}>{busy ? '创建中…' : `创建 ${count} 个`}</button></div>
      </div>
      </BorderBeam>
    </div>
  );
};

const osLabel = (o: string) => ({ windows: 'Windows', macos: 'macOS', linux: 'Linux' } as any)[o] || o;

export default function App() { return <ToastHost><Shell /></ToastHost>; }
