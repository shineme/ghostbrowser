import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Proxy, Profile } from './api';
import { Icon, Field, Toggle, ago, useToast, useConfirm, useExit, CC, Latency, StatCard, Pick, Segmented } from './ui';
import { COUNTRIES } from './locales';
import { MihomoModal } from './MihomoModal';

const cname = (cc?: string) => (cc && COUNTRIES.find(c => c.cc === cc.toUpperCase())?.name) || cc || '';
const IPTYPE: Record<string, [string, string]> = { residential: ['住宅', 'ok'], hosting: ['机房', 'warn'], mobile: ['移动', 'cy'] };
const speedText = (k?: number) => !k ? '—' : k >= 1000 ? `${(k / 1000).toFixed(k >= 10000 ? 0 : 1)} Mbps` : `${k} Kbps`;
const speedTone = (k?: number) => !k ? '' : k >= 20000 ? 'g' : k >= 5000 ? 'm' : 'b';
// Looks like at least one proxy line: host:port somewhere.
const looksLikeProxies = (t: string) => /(\d{1,3}(\.\d{1,3}){3}|[a-z0-9-]+\.[a-z]{2,})[\s:,|]+\d{2,5}/i.test(t);

export const ProxyView: React.FC<{ proxies: Proxy[]; profiles?: Profile[]; reload: () => void }> = ({ proxies, profiles = [], reload }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [q, setQ] = useState('');
  const [status, setStatus] = useState<'all' | 'ok' | 'fail' | 'unknown'>('all');
  const [country, setCountry] = useState('');
  const [importText, setImportText] = useState<string | null>(null);
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [bulk, setBulk] = useState('');
  const [sel, setSel] = useState<Set<string>>(new Set());
  const [live, setLive] = useState<Record<string, Proxy>>({});
  const [mihomo, setMihomo] = useState(false);

  // Rows update one by one while a batch check runs.
  useEffect(() => onEvent('proxy:checked', (p: Proxy) => {
    setLive(m => ({ ...m, [p.id]: p }));
    setBusy(s => { if (!s.has(p.id)) return s; const n = new Set(s); n.delete(p.id); return n; });
  }), []);
  useEffect(() => setLive({}), [proxies]);
  const all = useMemo(() => proxies.map(p => live[p.id] || p), [proxies, live]);

  // Paste anywhere on this page (outside a text box) opens the importer.
  useEffect(() => {
    const h = (e: ClipboardEvent) => {
      const el = document.activeElement as HTMLElement | null;
      if (el && (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable)) return;
      if (document.querySelector('.overlay')) return;
      const t = e.clipboardData?.getData('text') || '';
      if (looksLikeProxies(t)) { e.preventDefault(); setImportText(t); }
    };
    window.addEventListener('paste', h);
    return () => window.removeEventListener('paste', h);
  }, []);

  const countries = useMemo(() => [...new Set(all.filter(p => p.country).map(p => p.country))].sort(), [all]);
  const list = useMemo(() => {
    const s = q.toLowerCase();
    return all.filter(p => (status === 'all' || (p.status || 'unknown') === status) && (!country || p.country === country) &&
      (!s || [p.name, p.host, p.ip, p.country, cname(p.country), p.city, p.region, p.isp, p.tag].some(v => (v || '').toLowerCase().includes(s))));
  }, [all, q, status, country]);
  const target = sel.size ? [...sel].filter(id => all.some(p => p.id === id)) : list.map(p => p.id);
  const scope = sel.size ? `所选 ${target.length} 个` : (list.length === all.length ? '全部' : `筛选出的 ${list.length} 个`);

  const mark = (ids: string[], on: boolean) => setBusy(s => { const n = new Set(s); ids.forEach(id => on ? n.add(id) : n.delete(id)); return n; });
  const runCheck = async (ids: string[], label = '检测') => {
    if (!ids.length) return;
    setBulk(label); mark(ids, true);
    try { const r = await api.CheckProxies(ids); const ok = r.filter(p => ids.includes(p.id) && p.status === 'ok').length; toast(`${label}完成：${ok} 个可用，${ids.length - ok} 个不可用`, ok ? 'ok' : 'bad'); }
    finally { mark(ids, false); setBulk(''); reload(); }
  };
  const runSpeed = async (ids: string[]) => {
    if (!ids.length) return;
    setBulk('测速'); mark(ids, true);
    try { await api.SpeedTestProxies(ids); toast('测速完成'); } finally { mark(ids, false); setBulk(''); reload(); }
  };
  const delMany = async (ids: string[], what: string) => {
    if (!ids.length) return;
    if (!await ask({ title: `删除${what}？`, body: `共 ${ids.length} 个代理。正在使用它们的环境会改为直连，下次启动时生效。`, ok: '删除', danger: true })) return;
    await api.DeleteProxies(ids); setSel(new Set()); reload(); toast(`已删除 ${ids.length} 个代理`);
  };
  const [editing, setEditing] = useState<Proxy | null>(null);
  const [health, setHealth] = useState<{ checked: number; failed: number; at: number } | null>(null);
  useEffect(() => onEvent('proxy:health', (h: any) => { setHealth(h); reload(); }), [reload]);
  const rotate = async (p: Proxy) => {
    mark([p.id], true);
    try { const r = await api.RotateProxy(p.id); toast(`已换 IP：${r.ip}`, 'ok'); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark([p.id], false); reload(); }
  };
  const toggle = (id: string) => setSel(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const allSel = list.length > 0 && list.every(p => sel.has(p.id));

  if (proxies.length === 0) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="searching" size={64} theme="dark" /></div>
      <h3>还没有代理</h3>
      <div>直接 <b>Ctrl+V</b> 粘贴一批代理，或拖入 .txt / .csv 文件。格式、协议、国家和延迟都会自动识别。</div>
      <div className="actions"><button className="btn" onClick={() => setMihomo(true)}><Icon.server s={15} /> Clash / Mihomo</button><button className="btn primary" onClick={() => setImportText('')}><Icon.upload s={15} /> 批量导入</button></div>
      {importText !== null && <ImportModal initial={importText} onClose={() => setImportText(null)} onDone={(ids, check) => { reload(); if (check) runCheck(ids, '导入检测'); }} />}
      {mihomo && <MihomoModal onClose={() => setMihomo(false)} onAdded={reload} />}
    </div>
  );

  const okCount = all.filter(p => p.status === 'ok').length;
  const failIds = all.filter(p => p.status === 'fail').map(p => p.id);
  const failSet = new Set(failIds);
  const hit = profiles.filter(p => failSet.has(p.proxyId));
  const users: Record<string, number> = {};
  profiles.forEach(p => { if (p.proxyId) users[p.proxyId] = (users[p.proxyId] || 0) + 1; });
  const ipCount: Record<string, number> = {};
  all.forEach(p => { if (p.ip) ipCount[p.ip] = (ipCount[p.ip] || 0) + 1; });

  return (
    <>
      <div className="stat-row">
        <StatCard l="代理总数" n={all.length} ic="globe" pct={100} />
        <StatCard l="可用" n={okCount} of={all.length} ic="check" tone="ok" pct={okCount / all.length * 100} />
        <StatCard l="不可用 / 未检测" n={all.length - okCount} of={all.length} ic="alert" tone="bad" pct={(all.length - okCount) / all.length * 100} />
        <StatCard l="覆盖国家 / 地区" n={countries.length} ic="shield" tone="cy" pct={60} />
      </div>
      {failIds.length > 0 && <div className="notice bad">
        <Icon.alert /><b>{failIds.length} 个代理不可用</b>
        <span className="sub grow">{hit.length ? `影响 ${hit.length} 个环境：${hit.slice(0, 6).map(p => p.name).join('、')}${hit.length > 6 ? ' 等' : ''}` : '没有环境在使用它们'}{health ? ` · 自动检测于 ${ago(health.at)}` : ''}</span>
        {status !== 'fail' && <button className="btn sm" onClick={() => setStatus('fail')}>只看不可用</button>}
      </div>}
      <div className="toolbar" style={{ flexWrap: 'wrap' }}>
        <div className="search"><Icon.search /><input type="text" placeholder="搜索名称 / IP / 国家 / 城市 / ISP / 标签" value={q} onChange={e => setQ(e.target.value)} /></div>
        <Pick label="状态" value={status} onChange={setStatus as any} icon={<Icon.check />}
          options={[['all', '全部状态', all.length], ['ok', '可用', all.filter(p => p.status === 'ok').length], ['fail', '不可用', all.filter(p => p.status === 'fail').length], ['unknown', '未检测', all.filter(p => !p.status || p.status === 'unknown').length]]} />
        <Pick label="国家" value={country} onChange={setCountry} icon={<Icon.globe />}
          options={[['', '全部国家'] as [string, string, number?], ...countries.map(c => [c, `${c} · ${cname(c)}`, all.filter(p => p.country === c).length] as [string, string, number])]} />
        <div className="spacer" />
        <button className="btn" onClick={() => runCheck(target)} disabled={!!bulk} title={`检测${scope}`}>{bulk === '检测' ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 检测中…</> : <><Icon.refresh s={14} /> 检测{sel.size ? '所选' : ''}</>}</button>
        <button className="btn" onClick={() => runSpeed(target)} disabled={!!bulk} title={`测速${scope}（每个下载约 5 MB）`}>{bulk === '测速' ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 测速中…</> : <><Icon.bolt s={14} /> 测速{sel.size ? '所选' : ''}</>}</button>
        {sel.size > 0 ? <button className="btn danger" onClick={() => delMany([...sel], '所选代理')}><Icon.trash s={14} /> 删除所选</button>
          : failIds.length > 0 && <button className="btn" onClick={() => delMany(failIds, '全部不可用的代理')}><Icon.trash s={14} /> 清理不可用 ({failIds.length})</button>}
        <button className="btn" onClick={() => setMihomo(true)} title="连接 Clash Verge / Mihomo，切换节点并作为代理使用"><Icon.server s={15} /> Clash / Mihomo</button>
        <button className="btn primary" onClick={() => setImportText('')}><Icon.upload s={15} /> 批量导入</button>
      </div>

      <div className="panel list">
        <table>
          <thead><tr>
            <th style={{ width: 28 }}><input type="checkbox" checked={allSel} onChange={() => setSel(allSel ? new Set() : new Set(list.map(p => p.id)))} /></th>
            <th>名称</th><th>类型</th><th>出口 IP</th><th>位置</th><th>IP 类型 / ISP</th><th>延迟</th><th>速度</th><th>状态</th><th style={{ textAlign: 'right' }}>操作</th>
          </tr></thead>
          <tbody>
            {list.map((p, i) => <tr key={p.id} className={'rise' + (sel.has(p.id) ? ' sel' : '')} style={{ ['--i' as any]: Math.min(i, 20) }}>
              <td><input type="checkbox" checked={sel.has(p.id)} onChange={() => toggle(p.id)} /></td>
              <td><div className="nm">{p.name}</div><div className="sub num">{p.host}:{p.port}{p.user ? ' · 带认证' : ''}{p.tag ? <span className="tag" style={{ marginLeft: 6 }}>{p.tag}</span> : null}</div></td>
              <td><span className="tag num" style={{ textTransform: 'uppercase' }} title={p.typeGuessed ? '协议是按端口推测的，检测后自动确认' : ''}>{p.type}{p.typeGuessed ? '?' : ''}</span></td>
              <td className="sub num">{p.ip || '—'}{p.ip && ipCount[p.ip] > 1 && <div className="ip-dup" title="多个代理的出口是同一个 IP">⚠ {ipCount[p.ip]} 个代理同一出口</div>}{users[p.id] > 1 && <div className="ip-dup" title="多个环境共用这个代理，容易被平台关联">⚠ {users[p.id]} 个环境在用</div>}</td>
              <td>{p.country ? <div title={p.timezone}><CC cc={p.country} /><span>{cname(p.country)}</span><div className="sub">{[p.region, p.city].filter(Boolean).join(' · ')}</div></div> : <span className="sub">—</span>}</td>
              <td>{p.ipType ? <><span className={`badge ${IPTYPE[p.ipType]?.[1] || ''}`}>{IPTYPE[p.ipType]?.[0] || p.ipType}</span><div className="sub" style={{ maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={p.isp}>{p.isp}</div></> : <span className="sub">—</span>}</td>
              <td title={p.connectMs ? `首次连接 ${p.connectMs} ms（含握手）` : ''}><Latency ms={p.status === 'ok' ? p.latency : 0} /></td>
              <td><span className={`lat ${speedTone(p.speedKbps)}`}><span className="num" style={{ fontSize: 12 }}>{speedText(p.speedKbps)}</span></span></td>
              <td><span className={`badge ${p.status || 'unknown'}`} title={p.error || ''}><i className="d" />{p.status === 'ok' ? '可用' : p.status === 'fail' ? '不可用' : '未检测'}</span>
                <div className="sub" style={{ maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={p.error}>{p.status === 'fail' ? p.error : ago(p.checkedAt)}</div></td>
              <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                {busy.has(p.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="searching" size={20} theme="dark" /></span>
                  : <>
                    <button className="btn sm" onClick={() => runCheck([p.id])} title="检测国家、IP 类型和延迟"><Icon.refresh s={13} /> 检测</button>
                    <button className="btn sm" onClick={() => runSpeed([p.id])} title="下载约 5 MB 测速"><Icon.bolt s={13} /></button>
                    {p.changeIpUrl && <button className="btn sm rot" onClick={() => rotate(p)} title={p.rotateOnLaunch ? '换 IP（已设置每次启动前自动换）' : '调用换 IP 链接，然后重新检测出口'}><Icon.dice s={13} />{p.rotateOnLaunch ? <span className="rot-dot" /> : null}</button>}
                  </>}
                <button className="btn ghost icon" title="编辑" onClick={() => setEditing(p)}><Icon.edit s={15} /></button>
                <button className="btn ghost icon danger" onClick={() => delMany([p.id], '这个代理')}><Icon.trash s={15} /></button>
              </div></td>
            </tr>)}
          </tbody>
        </table>
        {list.length === 0 && <div className="sub" style={{ padding: 24, textAlign: 'center' }}>没有符合筛选条件的代理</div>}
      </div>
      {mihomo && <MihomoModal onClose={() => setMihomo(false)} onAdded={reload} />}
      {editing && <ProxyEditModal p={editing} onClose={() => setEditing(null)} onSaved={reload} />}
      {importText !== null && <ImportModal initial={importText} onClose={() => setImportText(null)} onDone={(ids, check) => { reload(); if (check) runCheck(ids, '导入检测'); }} />}
    </>
  );
};

type Preview = { line: number; raw: string; ok: boolean; error: string; dup: boolean; proxy: Proxy };

const ImportModal: React.FC<{ initial: string; onClose: () => void; onDone: (ids: string[], check: boolean) => void }> = ({ initial, onClose, onDone }) => {
  const toast = useToast();
  const [text, setText] = useState(initial);
  const [type, setType] = useState('auto');
  const [tag, setTag] = useState('');
  const [check, setCheck] = useState(true);
  const [skipDup, setSkipDup] = useState(true);
  const [rows, setRows] = useState<Preview[]>([]);
  const [busy, setBusy] = useState(false);
  const [drag, setDrag] = useState(false);
  const file = useRef<HTMLInputElement>(null);
  const [out, close] = useExit(onClose);

  useEffect(() => {
    const t = setTimeout(() => { api.PreviewProxies(text, type).then(r => setRows((r || []) as any)).catch(() => setRows([])); }, 200);
    return () => clearTimeout(t);
  }, [text, type]);

  const good = rows.filter(r => r.ok && !(skipDup && r.dup));
  const dups = rows.filter(r => r.ok && r.dup).length;
  const bad = rows.filter(r => !r.ok).length;

  const readFiles = async (fs: FileList | null) => {
    if (!fs?.length) return;
    const parts = await Promise.all([...fs].map(f => f.text()));
    setText(t => [t.trim(), ...parts].filter(Boolean).join('\n'));
  };
  const go = async () => {
    if (!good.length) { toast('没有可导入的代理', 'bad'); return; }
    setBusy(true);
    try {
      const added = await api.ImportProxyList(good.map(r => r.proxy) as any, tag);
      toast(`已导入 ${added.length} 个代理${check ? '，正在检测国家和延迟…' : ''}`);
      onDone(added.map(p => p.id), check); close();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); setBusy(false); }
  };

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 'min(880px, 94vw)' }}>
        <div className="dhead"><div className="glyph"><Icon.upload s={18} /></div><div><h3>批量导入代理</h3><div className="sub">粘贴或拖入文件，格式、协议自动识别；导入后自动检测国家、IP 类型和延迟</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="grid3">
            <Field label="协议（行内未写时）"><select value={type} onChange={e => setType(e.target.value)}><option value="auto">自动识别</option><option value="http">HTTP</option><option value="https">HTTPS</option><option value="socks5">SOCKS5</option></select></Field>
            <Field label="批次标签（可选）" hint="用于搜索，名称会变成 标签 #序号"><input type="text" value={tag} onChange={e => setTag(e.target.value)} placeholder="如 Smartproxy-US" /></Field>
            <div style={{ paddingTop: 18 }}>
              <Toggle on={check} onChange={setCheck} label="导入后立即检测" />
              <Toggle on={skipDup} onChange={setSkipDup} label="跳过重复" />
            </div>
          </div>
          <Field label="代理列表" hint="一行一个，支持 host:port · host:port:user:pass · user:pass:host:port · user:pass@host:port · socks5://… · 空格/逗号/Tab 分隔 · 行尾附换 IP 链接 · JSON 数组">
            <div className={'dropzone' + (drag ? ' on' : '')} onDragOver={e => { e.preventDefault(); setDrag(true); }} onDragLeave={() => setDrag(false)} onDrop={e => { e.preventDefault(); setDrag(false); readFiles(e.dataTransfer.files); }}>
              <textarea autoFocus style={{ minHeight: 150, fontFamily: 'var(--mono, monospace)', fontSize: 12 }} value={text} onChange={e => setText(e.target.value)}
                placeholder={'45.12.30.8:8080:user:pass\nuser:pass@gate.provider.com:7000\nsocks5://u:p@1.2.3.4:1080\n1.2.3.4 8080 user pass  https://provider.com/changeip?key=…'} />
            </div>
            <div style={{ display: "flex", flexWrap: "wrap", gap: 8, marginTop: 6 }}>
              <button type="button" className="btn sm" onClick={() => file.current?.click()}><Icon.folder s={13} /> 选择 .txt / .csv 文件</button>
              <button type="button" className="btn sm" onClick={async () => { try { const t = await navigator.clipboard.readText(); if (t) setText(x => [x.trim(), t].filter(Boolean).join('\n')); } catch { toast('无法读取剪贴板，请直接 Ctrl+V', 'bad'); } }}><Icon.copy s={13} /> 从剪贴板粘贴</button>
              {text && <button type="button" className="btn sm ghost" onClick={() => setText('')}>清空</button>}
              <input ref={file} type="file" accept=".txt,.csv,.json,text/plain" multiple hidden onChange={e => { readFiles(e.target.files); e.target.value = ''; }} />
            </div>
          </Field>
          {rows.length > 0 && <div className="panel list" style={{ maxHeight: 230, overflow: 'auto', marginTop: 4 }}>
            <table>
              <thead><tr><th style={{ width: 44 }}>行</th><th>协议</th><th>地址</th><th>账号</th><th>识别结果</th></tr></thead>
              <tbody>{rows.slice(0, 500).map(r => <tr key={r.line} style={{ opacity: r.ok && !(skipDup && r.dup) ? 1 : .55 }}>
                <td className="sub num">{r.line}</td>
                <td>{r.ok ? <span className="tag num" style={{ textTransform: 'uppercase' }} title={r.proxy.typeGuessed ? '按端口推测，检测时自动确认' : ''}>{r.proxy.type}{r.proxy.typeGuessed ? '?' : ''}</span> : '—'}</td>
                <td className="num" style={{ fontSize: 12 }}>{r.ok ? `${r.proxy.host}:${r.proxy.port}` : <span className="sub" title={r.raw}>{r.raw.slice(0, 48)}</span>}</td>
                <td className="sub num">{r.ok ? (r.proxy.user ? `${r.proxy.user} / ${'•'.repeat(Math.min(8, r.proxy.pass.length))}` : '无') : ''}</td>
                <td>{!r.ok ? <span className="badge fail"><i className="d" />{r.error}</span> : r.dup ? <span className="badge unknown"><i className="d" />重复</span> : <span className="badge ok"><i className="d" />可导入{r.proxy.changeIpUrl ? ' · 含换 IP 链接' : ''}</span>}</td>
              </tr>)}</tbody>
            </table>
          </div>}
        </div>
        <div className="dfoot">
          <span className="sub num" style={{ fontSize: 11.5 }}>可导入 <b style={{ color: 'var(--ok, #3ddc97)' }}>{good.length}</b> · 重复 {dups} · 无法识别 {bad}</span>
          <div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" onClick={go} disabled={busy || !good.length}>{busy ? '导入中…' : `导入 ${good.length} 个`}</button>
        </div>
      </div>
    </div>
  );
};

const ProxyEditModal: React.FC<{ p: Proxy; onClose: () => void; onSaved: () => void }> = ({ p, onClose, onSaved }) => {
  const toast = useToast();
  const [v, setV] = useState<Proxy>({ ...p });
  const [out, close] = useExit(onClose);
  const set = (x: Partial<Proxy>) => setV(o => ({ ...o, ...x } as Proxy));
  const save = async () => { await api.SaveProxy(v as any); toast('已保存'); onSaved(); close(); };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal">
        <div className="dhead"><div className="glyph"><Icon.globe s={18} /></div><div><h3>编辑代理</h3><div className="sub num">{p.host}:{p.port}</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="grid2">
            <Field label="名称"><input type="text" value={v.name} onChange={e => set({ name: e.target.value })} /></Field>
            <Field label="标签"><input type="text" value={v.tag || ''} onChange={e => set({ tag: e.target.value })} /></Field>
          </div>
          <div className="grid2">
            <Field label="类型"><Segmented className="block" items={[['http', 'HTTP'], ['https', 'HTTPS'], ['socks5', 'SOCKS5']]} value={v.type as any} onChange={t => set({ type: t, typeGuessed: false })} /></Field>
            <Field label="主机:端口"><div className="row"><input type="text" value={v.host} onChange={e => set({ host: e.target.value })} /><input type="number" style={{ width: 90, flex: 'none' }} value={v.port} onChange={e => set({ port: +e.target.value })} /></div></Field>
          </div>
          <div className="grid2">
            <Field label="用户名"><input type="text" value={v.user || ''} onChange={e => set({ user: e.target.value })} /></Field>
            <Field label="密码"><input type="password" value={v.pass || ''} onChange={e => set({ pass: e.target.value })} /></Field>
          </div>
          <Field label="换 IP 链接" hint="服务商提供的 API 地址，打开它就会更换出口 IP"><input type="text" value={v.changeIpUrl || ''} onChange={e => set({ changeIpUrl: e.target.value })} placeholder="https://..." /></Field>
          <Toggle on={!!v.rotateOnLaunch} onChange={x => set({ rotateOnLaunch: x })} label="每次启动环境前自动换 IP" hint="需要填写换 IP 链接；换 IP 失败不会阻止启动" />
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" onClick={save}><Icon.check s={15} /> 保存</button></div>
      </div>
    </div>
  );
};
