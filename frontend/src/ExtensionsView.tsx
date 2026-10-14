import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile } from './api';
import { Icon, Pick, StatCard, ago, useConfirm, useExit, useToast, Avatar } from './ui';

export type Ext = { id: string; name: string; version: string; description: string; source: string; origin: string; dir: string; icon: string; manifestVersion: number; size: number; addedAt: number; profiles: number };

export const SOURCE: Record<string, [string, React.FC<{ s?: number }>]> = { store: ['应用商店', Icon.globe], url: ['下载链接', Icon.download], file: ['本地文件', Icon.folder] };
export const fmtSize = (b: number) => !b ? '—' : b >= 1 << 20 ? `${(b / (1 << 20)).toFixed(1)} MB` : `${Math.max(1, Math.round(b / 1024))} KB`;
const isStoreRef = (s: string) => /^[a-p]{32}$/i.test(s) || /chrome(webstore)?\.google\.com/i.test(s);

/** Extension icon with a lettered fallback when the package has none. */
export const ExtIcon: React.FC<{ e: { icon: string; name: string }; size?: number }> = ({ e, size = 44 }) => (
  <span className="ext-ic" style={{ width: size, height: size }}>
    {e.icon ? <img src={e.icon} alt="" draggable={false} /> : <b>{(e.name || '?').slice(0, 1).toUpperCase()}</b>}
  </span>
);

export function ExtensionsView({ profiles, reload }: { profiles: Profile[]; reload?: () => void }) {
  const toast = useToast();
  const ask = useConfirm();
  const [list, setList] = useState<Ext[] | null>(null);
  const [loadErr, setLoadErr] = useState('');
  const [src, setSrc] = useState('');
  const [adding, setAdding] = useState(false);
  const [addErr, setAddErr] = useState('');
  const [q, setQ] = useState('');
  const [fsrc, setFsrc] = useState('');
  const [assign, setAssign] = useState<Ext | null>(null);
  const [removing, setRemoving] = useState('');

  const load = useCallback(async () => {
    try { setList(((await api.ListExtensions()) || []) as Ext[]); setLoadErr(''); }
    catch (e: any) { setLoadErr(String(e?.message || e)); setList(l => l || []); }
  }, []);
  useEffect(() => { load(); }, [load, profiles]);

  const add = async () => {
    const s = src.trim(); if (!s) return;
    setAdding(true); setAddErr('');
    try {
      const e = (isStoreRef(s) ? await api.AddExtensionByID(s) : await api.AddExtensionFromURL(s)) as Ext;
      toast(`已安装 ${e.name} ${e.version}`); setSrc(''); await load();
      setAssign(e);
    } catch (e: any) { setAddErr(String(e?.message || e)); }
    finally { setAdding(false); }
  };
  const remove = async (e: Ext) => {
    if (!await ask({ title: `删除扩展「${e.name}」？`, body: e.profiles ? `它会从 ${e.profiles} 个环境中移除，并删除本地文件。环境下次启动时生效；扩展在浏览器里保存的数据不会被清理。` : '会删除本地文件。可以随时重新添加。', ok: '删除', danger: true })) return;
    setRemoving(e.id);
    try { await api.RemoveExtension(e.id); toast(`已删除 ${e.name}`); await load(); reload?.(); }
    catch (er: any) { toast(String(er?.message || er), 'bad'); }
    finally { setRemoving(''); }
  };

  const all = list || [];
  const shown = useMemo(() => {
    const s = q.toLowerCase();
    return all.filter(e => (!fsrc || e.source === fsrc) && (!s || [e.name, e.description, e.id].some(v => (v || '').toLowerCase().includes(s))));
  }, [all, q, fsrc]);
  const usedBy = profiles.filter(p => (p.extensionIds || []).length).length;
  const mv3 = all.filter(e => e.manifestVersion >= 3).length;
  const total = all.reduce((n, e) => n + (e.size || 0), 0);

  const addBar = (
    <div className={`ext-add panel ${addErr ? 'err' : ''}`}>
      <div className="ext-add-row">
        <span className="ext-add-ic"><Icon.plus s={16} /></span>
        <input type="text" value={src} onChange={e => { setSrc(e.target.value); setAddErr(''); }} onKeyDown={e => { if (e.key === 'Enter' && !adding) add(); }} disabled={adding}
          placeholder="粘贴扩展 ID、Chrome 应用商店链接，或 .crx / .zip 的下载链接 / 本地路径" spellCheck={false} />
        <button className="btn primary" onClick={add} disabled={adding || !src.trim()}>
          {adding ? <><ThinkingOrb state="connecting" size={20} theme="dark" /> 正在下载…</> : <><Icon.download s={15} /> 添加扩展</>}
        </button>
      </div>
      <div className="ext-add-hint">
        {src.trim() ? <span className="ext-kind">{isStoreRef(src.trim()) ? <><Icon.globe s={12} /> 从 Chrome 应用商店下载</> : <><Icon.download s={12} /> 从链接 / 本地文件安装</>}</span> : null}
        <span>例：<code>cjpalhdlnbpafiamejdnhcphjbkeiagm</code> · <code>https://chromewebstore.google.com/detail/…</code> · <code>D:\ext\tool.crx</code></span>
      </div>
      {addErr && <div className="notice bad" style={{ margin: '10px 0 0' }}><Icon.alert /><span className="grow">{addErr}</span><button className="btn sm" onClick={add}>重试</button></div>}
    </div>
  );

  if (list === null) return <div className="empty"><ThinkingOrb state="searching" size={32} theme="dark" /><div>正在读取扩展…</div></div>;

  if (all.length === 0) return (
    <>
      {addBar}
      {loadErr && <div className="notice bad"><Icon.alert /><span className="grow">读取扩展失败：{loadErr}</span><button className="btn sm" onClick={load}>重试</button></div>}
      <div className="empty">
        <div className="halo"><ThinkingOrb state="searching" size={64} theme="dark" /></div>
        <h3>还没有扩展</h3>
        <div style={{ maxWidth: 520 }}>在上方粘贴 Chrome 应用商店的扩展 ID 或链接，一键下载安装，再分配给需要的环境。<br />定制内核（Undetectable）和开源内核（Chromium）都会在启动时加载已分配的扩展。</div>
        <div className="sub" style={{ fontSize: 12 }}>品牌版 Chrome 137 及以上版本可能会忽略命令行加载的扩展。</div>
      </div>
    </>
  );

  return (
    <>
      <div className="stat-row">
        <StatCard l="已安装扩展" n={all.length} ic="layers" pct={100} />
        <StatCard l="使用扩展的环境" n={usedBy} of={profiles.length} ic="fingerprint" tone="ok" pct={profiles.length ? usedBy / profiles.length * 100 : 0} />
        <StatCard l="Manifest V3" n={mv3} of={all.length} ic="shield" tone="cy" pct={mv3 / all.length * 100} />
        <StatCard l="占用空间 (MB)" n={Math.round(total / (1 << 20))} ic="server" tone="vi" pct={Math.min(100, total / (200 << 20) * 100)} />
      </div>
      {addBar}
      {loadErr && <div className="notice bad"><Icon.alert /><span className="grow">刷新失败：{loadErr}</span><button className="btn sm" onClick={load}>重试</button></div>}
      <div className="toolbar">
        <div className="search"><Icon.search /><input type="text" placeholder="搜索名称、描述或扩展 ID" value={q} onChange={e => setQ(e.target.value)} /></div>
        <Pick label="来源" value={fsrc} onChange={setFsrc} icon={<Icon.download />}
          options={[['', '全部来源', all.length], ...Object.entries(SOURCE).map(([k, [l]]) => [k, l, all.filter(e => e.source === k).length] as [string, string, number])]} />
        <div className="spacer" />
        <span className="sub" style={{ fontSize: 12 }}>定制内核与开源内核都会加载 · 品牌版 Chrome 137+ 可能忽略</span>
      </div>

      <div className="ext-grid">
        {shown.map((e, i) => {
          const [sl, SI] = SOURCE[e.source] || SOURCE.file;
          return <div key={e.id} className="ext-card panel spot rise" style={{ ['--i' as any]: Math.min(i, 20) }}>
            <div className="ext-top">
              <ExtIcon e={e} />
              <div className="ext-meta">
                <div className="ext-name" title={e.name}>{e.name}</div>
                <div className="ext-ver"><span className="num">v{e.version}</span><span className={`mv ${e.manifestVersion >= 3 ? 'v3' : 'v2'}`} title={e.manifestVersion >= 3 ? 'Manifest V3' : 'Manifest V2：新版 Chrome 已停止支持，定制内核仍可加载'}>MV{e.manifestVersion}</span></div>
              </div>
              <button className="btn ghost icon danger" title="删除扩展" disabled={removing === e.id} onClick={() => remove(e)}>{removing === e.id ? <ThinkingOrb state="breathing" size={20} theme="dark" /> : <Icon.trash s={15} />}</button>
            </div>
            <div className="ext-desc" title={e.description}>{e.description || '没有描述'}</div>
            <div className="ext-info">
              <span className="ext-src" title={e.origin}><SI s={12} />{sl}</span>
              <span className="num">{fmtSize(e.size)}</span>
              <span>{ago(e.addedAt)}添加</span>
            </div>
            <div className="ext-foot">
              <button className={`ext-users ${e.profiles ? 'on' : ''}`} onClick={() => setAssign(e)} title="查看 / 修改使用它的环境">
                <span className="ext-av">{profiles.filter(p => (p.extensionIds || []).includes(e.id)).slice(0, 3).map(p => <Avatar key={p.id} seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} />)}</span>
                {e.profiles ? <><b className="num">{e.profiles}</b> 个环境使用</> : '未分配给任何环境'}
              </button>
              <button className="btn sm" onClick={() => setAssign(e)}><Icon.layers s={13} /> 分配到环境</button>
            </div>
          </div>;
        })}
      </div>
      {shown.length === 0 && <div className="sub" style={{ padding: 32, textAlign: 'center' }}>没有符合条件的扩展</div>}
      {assign && <ExtAssignModal ext={assign} profiles={profiles} onClose={() => setAssign(null)} onDone={() => { load(); reload?.(); }} />}
    </>
  );
}

const ExtAssignModal: React.FC<{ ext: Ext; profiles: Profile[]; onClose: () => void; onDone: () => void }> = ({ ext, profiles, onClose, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const initial = useMemo(() => new Set(profiles.filter(p => (p.extensionIds || []).includes(ext.id)).map(p => p.id)), [profiles, ext.id]);
  const [pick, setPick] = useState<Set<string>>(() => new Set(initial));
  const [q, setQ] = useState('');
  const [group, setGroup] = useState('\u0000');
  const [busy, setBusy] = useState(false);
  const groups = useMemo(() => [...new Set(profiles.map(p => p.group || ''))].sort(), [profiles]);
  const shown = profiles.filter(p => (group === '\u0000' || (p.group || '') === group) && (!q || `${p.name} ${p.group} ${(p.tags || []).join(' ')}`.toLowerCase().includes(q.toLowerCase())));
  const toggle = (id: string) => setPick(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const changed = profiles.filter(p => pick.has(p.id) !== initial.has(p.id));
  const adds = changed.filter(p => pick.has(p.id)).length;
  const save = async () => {
    setBusy(true);
    try {
      for (const p of changed) {
        const cur = (p.extensionIds || []).filter(x => x !== ext.id);
        await api.SetProfileExtensions(p.id, pick.has(p.id) ? [...cur, ext.id] : cur);
      }
      toast(changed.length ? `已更新 ${changed.length} 个环境，下次启动生效` : '没有变化');
      onDone(); close();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); setBusy(false); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 600 }}>
        <div className="dhead"><ExtIcon e={ext} size={38} /><div style={{ minWidth: 0 }}><h3 className="ellip">分配「{ext.name}」</h3><div className="sub">勾选要加载这个扩展的环境，下次启动时生效</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="row" style={{ gap: 8, marginBottom: 8, alignItems: 'center' }}>
            <div className="search grow-search"><Icon.search /><input type="text" placeholder="搜索环境名称、分组、标签" value={q} onChange={e => setQ(e.target.value)} autoFocus /></div>
            {groups.length > 1 && <Pick label="分组" value={group} onChange={setGroup} icon={<Icon.folder />} align="right"
              options={[['\u0000', '全部分组', profiles.length] as [string, string, number], ...groups.map(g => [g, g || '未分组', profiles.filter(p => (p.group || '') === g).length] as [string, string, number])]} />}
          </div>
          <div className="assign-filters" style={{ justifyContent: 'space-between', alignItems: 'center' }}>
            <span className="sub" style={{ fontSize: 12 }}>已选 <b className="num" style={{ color: 'var(--txt)' }}>{pick.size}</b> / {profiles.length}</span>
            <span style={{ display: 'flex', gap: 6 }}>
              <button className="chip-tg" onClick={() => setPick(s => new Set([...s, ...shown.map(p => p.id)]))}>全选{shown.length !== profiles.length ? '筛选结果' : ''}</button>
              <button className="chip-tg" onClick={() => setPick(s => { const n = new Set(s); shown.forEach(p => n.delete(p.id)); return n; })}>清空{shown.length !== profiles.length ? '筛选结果' : ''}</button>
            </span>
          </div>
          <div className="assign-list" style={{ maxHeight: 320 }}>
            {shown.map(p => {
              const n = (p.extensionIds || []).filter(x => x !== ext.id).length;
              return <div key={p.id} className={`assign-row ${pick.has(p.id) ? 'on' : ''}`} onClick={() => toggle(p.id)}>
                <span className={`chk ${pick.has(p.id) ? 'on' : ''}`}><Icon.check s={11} /></span>
                <Avatar seed={p.fingerprint.noiseSeed || p.id} os={p.fingerprint.os} />
                <span style={{ flex: 1, minWidth: 0 }}><div className="nm ellip">{p.name}</div><div className="sub ellip" style={{ fontSize: 11 }}>{p.group || '未分组'}{n ? ` · 另有 ${n} 个扩展` : ''}</div></span>
                {pick.has(p.id) !== initial.has(p.id) && <span className={`badge ${pick.has(p.id) ? 'ok' : 'warn'}`}>{pick.has(p.id) ? '将添加' : '将移除'}</span>}
              </div>;
            })}
            {shown.length === 0 && <div className="sub" style={{ padding: 16, textAlign: 'center' }}>{profiles.length ? '没有符合条件的环境' : '还没有环境，先去「环境」页创建一个'}</div>}
          </div>
        </div>
        <div className="dfoot">
          <span className="sub" style={{ fontSize: 12 }}>{changed.length ? `添加 ${adds} · 移除 ${changed.length - adds}` : '未修改'}</span>
          <div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" disabled={busy || !changed.length} onClick={save}>{busy ? '保存中…' : <><Icon.check s={15} /> 保存</>}</button>
        </div>
      </div>
    </div>
  );
};
