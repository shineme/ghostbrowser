// 默认浏览器：GhostBrowser registers once with Windows; the starred profile
// opens external links. Star button (profile rows), settings card, and the
// picker shown when a link arrives without a usable default.
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { api, Profile, onEvent } from './api';
import { Icon, useToast, useExit, Avatar, OsIcon } from './ui';

export type DBStatus = { supported: boolean; registered: boolean; isSystemDefault: boolean; profileId: string; profileName: string; error?: string };
const A = api as any;
const EMPTY: DBStatus = { supported: false, registered: false, isSystemDefault: false, profileId: '', profileName: '' };

/** Shared status: refreshed on "defaultbrowser:changed" and on window focus
 *  (the user comes back from the system Default Apps page). */
export const useDefaultBrowser = (): [DBStatus, (s?: DBStatus) => void] => {
  const [st, setSt] = useState<DBStatus>(EMPTY);
  const refresh = useCallback((s?: DBStatus) => { if (s) { setSt(s); return; } A.DefaultBrowserStatus?.().then((x: DBStatus) => x && setSt(x)).catch(() => {}); }, []);
  useEffect(() => { refresh(); }, [refresh]);
  useEffect(() => onEvent('defaultbrowser:changed', (s: DBStatus) => setSt(s)), []);
  useEffect(() => { const f = () => refresh(); window.addEventListener('focus', f); return () => window.removeEventListener('focus', f); }, [refresh]);
  return [st, refresh];
};

const shortURL = (u: string) => u.replace(/^https?:\/\//, '').replace(/^file:\/\/\/?/, '').replace(/\/$/, '');

/** Star toggle next to a profile name. */
export const DefaultStar: React.FC<{ profile: Profile; on: boolean; onChanged: (s: DBStatus) => void }> = ({ profile, on, onChanged }) => {
  const toast = useToast();
  const [pop, setPop] = useState(false);
  const click = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      const s: DBStatus = await A.SetDefaultProfile(on ? '' : profile.id);
      onChanged(s);
      if (on) { toast(`已取消默认：外部链接打开时会询问用哪个环境`); return; }
      setPop(true); setTimeout(() => setPop(false), 520);
      if (s.supported && !s.isSystemDefault) toast(`「${profile.name}」已设为默认。还差一步：在系统「默认应用」里选择 GhostBrowser`, 'star', { label: '去设置', onClick: () => { A.RegisterDefaultBrowser().then(onChanged).catch((er: any) => toast(String(er?.message || er), 'bad')); } });
      else toast(`外部链接将在「${profile.name}」中打开`, 'star');
    } catch (er: any) { toast(String(er?.message || er), 'bad'); }
  };
  return <button className={`db-star ${on ? 'on' : ''} ${pop ? 'pop' : ''}`} onClick={click} title={on ? '当前默认浏览器（点击取消）' : '设为默认浏览器'} aria-label={on ? '当前默认浏览器' : '设为默认浏览器'} aria-pressed={on}>
    <Icon.star s={13} />
  </button>;
};

/** 设置 → 默认浏览器 card. */
export const DefaultBrowserCard: React.FC<{ index: number; profiles?: Profile[] }> = ({ index }) => {
  const toast = useToast();
  const [st, refresh] = useDefaultBrowser();
  const [busy, setBusy] = useState('');
  const run = async (key: string, fn: () => Promise<any>, ok?: string) => {
    setBusy(key);
    try { const s = await fn(); if (s && typeof s === 'object' && 'supported' in s) refresh(s); else refresh(); if (ok) toast(ok); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setBusy(''); }
  };
  return (
    <div className="panel pad setting-card spot rise db-card" style={{ ['--i' as any]: index }}>
      <div className="head"><span className="ic"><Icon.link /></span><div><h4>默认浏览器</h4><span>在微信、邮件、文档里点开的链接，直接用指定环境打开</span></div></div>
      {!st.supported ? <div className="notice warn" style={{ marginBottom: 0 }}><Icon.alert /><span className="grow">只有 Windows 版可以设为系统默认浏览器，当前系统不支持接管外部链接。</span></div> : <>
        <div className={`geo-card db-status ${st.isSystemDefault ? '' : 'warn'}`}>
          <div className="db-chips">
            <span className={`badge ${st.registered ? 'ok' : 'unknown'}`}><span className="d" />{st.registered ? '已注册' : '未注册'}</span>
            <span className={`badge ${st.isSystemDefault ? 'ok' : st.registered ? 'warn' : 'unknown'}`}><span className="d" />系统默认：{st.isSystemDefault ? '是' : '否'}</span>
          </div>
          <div className="db-profile">
            <span className={`db-star static ${st.profileId ? 'on' : ''}`}><Icon.star s={13} /></span>
            {st.profileId ? <span className="db-pn">默认环境 <b title={st.profileName}>{st.profileName}</b></span> : <span className="db-pn dim">还没有默认环境：在环境列表点名称旁的 <Icon.star s={11} /> 设置</span>}
          </div>
          {!st.isSystemDefault && <div className="sub db-hint">{st.registered ? 'Windows 不允许程序自己改默认浏览器：打开「默认应用」，在 GhostBrowser 页面把 HTTP、HTTPS 设为 GhostBrowser。' : '先注册，Windows 才会在「默认应用」里列出 GhostBrowser。只写当前用户注册表，不需要管理员权限。'}</div>}
          {st.error && <div className="sub upd-err">{st.error}</div>}
        </div>
        <div className="row db-actions">
          <button className="btn primary" disabled={!!busy} onClick={() => run('reg', () => A.RegisterDefaultBrowser(), '已注册，正在打开系统「默认应用」')}><Icon.check s={14} /> {st.registered ? '重新注册并打开系统设置' : '注册并打开系统设置'}</button>
          <button className="btn" disabled={!!busy || !st.registered} onClick={() => run('open', () => A.OpenDefaultAppsSettings())}><Icon.settings s={14} /> 打开系统默认应用</button>
          <div className="spacer" />
          {st.registered && <button className="btn ghost" disabled={!!busy} onClick={() => run('unreg', () => A.UnregisterDefaultBrowser(), '已取消注册')}><Icon.x s={14} /> 取消注册</button>}
        </div>
      </>}
    </div>
  );
};

/** Asks which profile opens a link that arrived with no usable default. */
export const LinkPicker: React.FC<{ profiles: Profile[]; running: Set<string> }> = ({ profiles, running }) => {
  const [queue, setQueue] = useState<string[]>([]);
  const add = useCallback((list: string[]) => setQueue(q => [...q, ...list.filter(u => u && !q.includes(u))]), []);
  useEffect(() => { A.PendingLinks?.().then((l: string[]) => add(l || [])).catch(() => {}); }, [add]);
  useEffect(() => onEvent('defaultbrowser:pick', (u: string) => add([u])), [add]);
  const url = queue[0];
  if (!url) return null;
  return <PickDialog key={url} url={url} more={queue.length - 1} profiles={profiles} running={running} onDone={() => setQueue(q => q.slice(1))} />;
};

const PickDialog: React.FC<{ url: string; more: number; profiles: Profile[]; running: Set<string>; onDone: () => void }> = ({ url, more, profiles, running, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onDone);
  const [q, setQ] = useState('');
  const sorted = useMemo(() => [...profiles].sort((a, b) => (+running.has(b.id) - +running.has(a.id)) || (b.lastRun || 0) - (a.lastRun || 0)), [profiles, running]);
  const list = useMemo(() => { const l = q.trim().toLowerCase(); return l ? sorted.filter(p => p.name.toLowerCase().includes(l) || (p.group || '').toLowerCase().includes(l)) : sorted; }, [sorted, q]);
  const [sel, setSel] = useState(sorted[0]?.id || '');
  useEffect(() => { if (list.length && !list.some(p => p.id === sel)) setSel(list[0].id); }, [list, sel]);
  const [remember, setRemember] = useState(true);
  const [busy, setBusy] = useState(false);
  const cancel = () => { A.DismissLink?.(url); close(); };
  const go = async () => {
    if (!sel) return;
    setBusy(true);
    try {
      if (remember) await A.SetDefaultProfile(sel);
      await A.OpenURLIn(sel, url);
      const p = profiles.find(x => x.id === sel);
      toast(running.has(sel) ? `已在「${p?.name}」中打开链接` : `正在启动「${p?.name}」打开链接`);
      close();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); setBusy(false); }
  };
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !busy) cancel(); if (e.key === 'Enter' && !busy && sel && !(e.target as HTMLElement)?.matches?.('button')) go(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });
  return <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && !busy && cancel()}>
    <div className="modal db-pick">
      <div className="dhead">
        <div className="glyph"><Icon.link s={18} /></div>
        <div className="db-pick-h"><h3>用哪个环境打开链接？</h3><div className="sub db-url mono" title={url}>{shortURL(url)}</div></div>
        <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" disabled={busy} onClick={cancel} aria-label="关闭"><Icon.x /></button></div>
      </div>
      <div className="dbody">
        {profiles.length > 6 && <div className="search db-search"><Icon.search /><input type="text" autoFocus placeholder="搜索环境" value={q} onChange={e => setQ(e.target.value)} /></div>}
        <div className="db-list" role="listbox">
          {list.length === 0 && <div className="sub db-empty">{profiles.length ? '没有匹配的环境' : '还没有环境：先新建一个环境'}</div>}
          {list.map(p => <button key={p.id} role="option" aria-selected={sel === p.id} className={`db-opt ${sel === p.id ? 'on' : ''}`} onClick={() => setSel(p.id)} onDoubleClick={() => { setSel(p.id); setTimeout(go, 0); }}>
            <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} running={running.has(p.id)} />
            <span className="db-on"><b>{p.name}</b><small>{p.group || '未分组'}</small></span>
            {running.has(p.id) ? <span className="badge ok"><span className="d" />运行中</span> : <span className="db-os"><OsIcon os={p.fingerprint.os} s={13} /></span>}
            <span className="radio" />
          </button>)}
        </div>
        <label className="db-remember" onClick={() => setRemember(r => !r)}>
          <span className={`chk ${remember ? 'on' : ''}`}><Icon.check s={11} /></span>
          <span>记住为默认<small>以后的链接直接用这个环境打开，可在环境列表点 <Icon.star s={10} /> 更改</small></span>
        </label>
      </div>
      <div className="dfoot">
        {more > 0 && <span className="sub num">还有 {more} 个链接等待选择</span>}
        <div className="spacer" />
        <button className="btn" disabled={busy} onClick={cancel}>取消</button>
        <button className="btn primary" disabled={busy || !sel} onClick={go}><Icon.play s={13} /> {sel && running.has(sel) ? '打开' : '启动并打开'}</button>
      </div>
    </div>
  </div>;
};
