import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Proxy, Profile, Subscription, SubPreview, CoreStatus } from './api';
import { Icon, Field, Toggle, ago, useToast, useExit, Segmented, Floating, Tip, flag } from './ui';

// ---- shared helpers (proxy list, drawer picker, assign dialog) ----
const DIRECT = new Set(['http', 'socks5']);
/** A subscription node the browser cannot speak itself: it runs through the mihomo core. */
export const isCore = (p: Partial<Proxy>) => !!p.rawNode && !!p.protocol && !DIRECT.has(p.protocol);
const PROTO: Record<string, string> = { hysteria2: 'HY2', hysteria: 'HY', vless: 'VLESS', vmess: 'VMess', ss: 'SS', ssr: 'SSR', trojan: 'Trojan', tuic: 'TUIC', anytls: 'AnyTLS', http: 'HTTP', https: 'HTTPS', socks5: 'SOCKS5', 'socks5-tls': 'SOCKS5-TLS', wireguard: 'WireGuard', snell: 'Snell', ssh: 'SSH', mieru: 'Mieru' };
export const protoName = (k: string) => PROTO[k] || k.toUpperCase();
export const protoOf = (p: Partial<Proxy>) => protoName(p.protocol || p.type || '');
export const ProtoChip: React.FC<{ p: Proxy }> = ({ p }) => <span className="proto-cell">
  <span className={`tag num proto-tag ${isCore(p) ? 'core' : ''}`} title={p.typeGuessed ? '协议是按端口推测的，检测后自动确认' : ''}>{protoOf(p)}{p.typeGuessed ? '?' : ''}</span>
  {isCore(p) && <Tip tip="浏览器不支持这个协议：启动和检测时由代理内核 mihomo 在本机转换成 SOCKS5"><span className="badge core-badge">内核</span></Tip>}
</span>;

export const fmtBytes = (n: number) => {
  if (!n) return '0';
  const u = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']; let i = 0; let v = n;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v >= 100 || i === 0 ? v.toFixed(0) : v.toFixed(1)} ${u[i]}`;
};
const dateText = (ts: number) => { const d = new Date(ts * 1000); return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`; };
const hostOf = (u: string) => { try { return new URL(u).host; } catch { return u.slice(0, 32); } };

/** Group key of a proxy: '' = 手动添加, else the subscription ID. */
export const groupKey = (p: Proxy) => p.subId || '';
/** Group filter options for a Pick: 全部 / 手动 / each subscription (with counts). */
export const groupOptions = (proxies: Proxy[], subs: Subscription[]): [string, string, number?][] => {
  const n: Record<string, number> = {};
  proxies.forEach(p => { const k = groupKey(p); n[k] = (n[k] || 0) + 1; });
  const out: [string, string, number?][] = [['*', '全部分组', proxies.length], ['', '手动添加', n[''] || 0]];
  subs.forEach(s => out.push([s.id, s.name, n[s.id] || 0]));
  // nodes of a subscription this computer does not know (synced before it) still get a group
  Object.keys(n).filter(k => k && !subs.some(s => s.id === k)).forEach(k => out.push([k, proxies.find(p => p.subId === k)?.subName || '订阅', n[k]]));
  return out;
};

export const useSubs = (): [Subscription[], () => void] => {
  const [subs, setSubs] = useState<Subscription[]>([]);
  const load = () => { api.ListSubscriptions().then(s => setSubs((s || []) as any)).catch(() => {}); };
  useEffect(() => { load(); return onEvent('subs:updated', (s: Subscription[]) => setSubs(s || [])); }, []);
  return [subs, load];
};

const Traffic: React.FC<{ s: Subscription }> = ({ s }) => {
  const used = (s.upload || 0) + (s.download || 0);
  if (!s.total && !used) return <span className="sub">未提供</span>;
  const pct = s.total ? Math.min(100, used * 100 / s.total) : 0;
  return <div className="sub-traffic">
    <div className="num" style={{ fontSize: 12 }}>{fmtBytes(used)}<span className="sub"> / {s.total ? fmtBytes(s.total) : '不限'}</span></div>
    {s.total > 0 && <div className={`env-bar ${pct > 90 ? 'bad' : ''}`}><i style={{ width: `${Math.max(pct, 2)}%` }} /></div>}
  </div>;
};

const Expire: React.FC<{ ts: number }> = ({ ts }) => {
  if (!ts) return <span className="sub">长期有效</span>;
  const days = Math.floor((ts * 1000 - Date.now()) / 86400000);
  return <div><div className="num" style={{ fontSize: 12 }}>{dateText(ts)}</div><div className={`sub ${days < 7 ? 'warn-t' : ''}`}>{days < 0 ? '已过期' : days === 0 ? '今天到期' : `剩 ${days} 天`}</div></div>;
};

const ProtoSummary: React.FC<{ m?: Record<string, number>; max?: number }> = ({ m, max = 5 }) => {
  const list = Object.entries(m || {}).sort((a, b) => b[1] - a[1]);
  if (!list.length) return null;
  return <div className="proto-sum">{list.slice(0, max).map(([k, n]) => <span key={k} className={`tag num ${DIRECT.has(k) ? '' : 'core'}`}>{protoName(k)} {n}</span>)}{list.length > max && <span className="sub">+{list.length - max}</span>}</div>;
};

// ---- subscriptions tab ----
export const SubsPanel: React.FC<{ subs: Subscription[]; proxies: Proxy[]; profiles: Profile[]; reload: () => void; reloadSubs: () => void; onShowGroup: (id: string) => void }> = ({ subs, proxies, profiles, reload, reloadSubs, onShowGroup }) => {
  const toast = useToast();
  const [adding, setAdding] = useState(false);
  const [editing, setEditing] = useState<Subscription | null>(null);
  const [deleting, setDeleting] = useState<Subscription | null>(null);
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [all, setAll] = useState(false);
  const [core, setCore] = useState<CoreStatus | null>(null);
  useEffect(() => { api.CoreStatus().then(c => setCore(c as any)).catch(() => {}); }, [subs.length]);
  const mark = (id: string, on: boolean) => setBusy(s => { const n = new Set(s); on ? n.add(id) : n.delete(id); return n; });
  const update = async (s: Subscription) => {
    mark(s.id, true);
    try { const r = await api.UpdateSubscription(s.id); toast(`「${s.name}」已更新：新增 ${r.added}，更新 ${r.updated}${r.removed ? `，移除 ${r.removed}` : ''}${r.gone ? `，${r.gone} 个已失效但仍有环境在用` : ''}`, 'ok'); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { mark(s.id, false); reload(); reloadSubs(); }
  };
  const updateAll = async () => {
    setAll(true);
    try { const r = await api.UpdateAllSubscriptions(); const bad = (r || []).filter(x => x.enabled && x.lastError).length; toast(bad ? `更新完成，${bad} 个订阅失败` : '全部订阅已更新', bad ? 'bad' : 'ok'); }
    finally { setAll(false); reload(); reloadSubs(); }
  };
  const users = useMemo(() => { const m: Record<string, number> = {}; const byId = new Map(proxies.map(p => [p.id, p])); profiles.forEach(pr => { const px = byId.get(pr.proxyId); if (px?.subId) m[px.subId] = (m[px.subId] || 0) + 1; }); return m; }, [proxies, profiles]);
  const coreNodes = proxies.filter(isCore).length;

  return <>
    {core && !core.installed && coreNodes > 0 && <div className="notice warn">
      <Icon.cpu /><b>{coreNodes} 个节点需要代理内核</b>
      <span className="sub grow">VLESS / VMess / SS / Trojan / Hysteria2 等协议由 mihomo 内核在本机转换后才能给浏览器使用。</span>
      <button className="btn sm" onClick={() => window.dispatchEvent(new CustomEvent('gb:goto-settings', { detail: 'settings-core' }))}>去设置下载</button>
    </div>}
    <div className="toolbar">
      <div className="sub" style={{ fontSize: 12.5 }}>{subs.length ? `${subs.length} 个订阅 · ${subs.reduce((a, s) => a + (s.nodeCount || 0), 0)} 个节点` : '把机场 / 代理服务商的订阅链接加进来，节点会按订阅分组显示'}</div>
      <div className="spacer" />
      {subs.length > 0 && <button className="btn" disabled={all} onClick={updateAll}>{all ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 更新中…</> : <><Icon.refresh s={14} /> 全部更新</>}</button>}
      <button className="btn primary" onClick={() => setAdding(true)}><Icon.plus s={15} /> 添加订阅</button>
    </div>
    {subs.length === 0 ? <div className="empty" style={{ minHeight: 280 }}>
      <div className="halo"><Icon.link s={30} /></div>
      <h3>还没有订阅</h3>
      <div>支持 Clash / Mihomo 配置、Base64 和分享链接列表（vless:// vmess:// ss:// trojan:// hy2:// tuic:// anytls:// http:// socks5://）。可以添加多个订阅，每个订阅的节点单独成组，并按订阅自带的间隔自动更新。</div>
      <div className="actions"><button className="btn primary" onClick={() => setAdding(true)}><Icon.plus s={15} /> 添加订阅</button></div>
    </div> : <div className="panel list">
      <table>
        <thead><tr><th>订阅</th><th>节点</th><th>流量</th><th>到期</th><th>上次更新</th><th>自动更新</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
        <tbody>{subs.map((s, i) => <tr key={s.id} className="rise" style={{ ['--i' as any]: Math.min(i, 20), opacity: s.enabled ? 1 : .6 }}>
          <td><div className="nm">{s.name}{!s.enabled && <span className="badge unknown" style={{ marginLeft: 6 }}>已停用</span>}</div><div className="sub num">{hostOf(s.url)} · {s.format === 'clash' ? 'Clash' : s.format === 'uri' ? '分享链接' : '—'}{users[s.id] ? ` · ${users[s.id]} 个环境在用` : ''}</div></td>
          <td><div className="num" style={{ fontWeight: 600 }}>{s.nodeCount}{s.skipped ? <span className="sub" style={{ fontWeight: 400 }}> · 跳过 {s.skipped}</span> : null}</div><ProtoSummary m={s.protocols} max={4} /></td>
          <td className="nowrap"><Traffic s={s} /></td>
          <td className="nowrap"><Expire ts={s.expire} /></td>
          <td>{s.lastError ? <Tip tip={s.lastError}><span className="badge fail"><i className="d" />失败</span></Tip> : <span className="sub">{s.lastUpdated ? ago(s.lastUpdated) : '—'}</span>}
            {s.lastError && <div className="sub ellip" style={{ maxWidth: 170 }}>{s.lastUpdated ? `成功于 ${ago(s.lastUpdated)}` : s.lastError}</div>}</td>
          <td className="sub nowrap">{s.autoUpdateHours ? `每 ${s.autoUpdateHours} 小时` : '关闭'}</td>
          <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
            {busy.has(s.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="searching" size={20} theme="dark" /></span>
              : <button className="btn sm" onClick={() => update(s)} title="立即下载订阅并更新节点（已有节点的 ID 保持不变，绑定的环境不受影响）"><Icon.refresh s={13} /> 更新</button>}
            <button className="btn sm" onClick={() => onShowGroup(s.id)} title="在代理列表中只看这个订阅的节点">节点</button>
            <button className="btn ghost icon" title="编辑" onClick={() => setEditing(s)}><Icon.edit s={15} /></button>
            <button className="btn ghost icon danger" title="删除" onClick={() => setDeleting(s)}><Icon.trash s={15} /></button>
          </div></td>
        </tr>)}</tbody>
      </table>
    </div>}
    {adding && <AddSubModal onClose={() => setAdding(false)} onDone={id => { reload(); reloadSubs(); if (id) onShowGroup(id); }} />}
    {editing && <EditSubModal s={editing} onClose={() => setEditing(null)} onDone={() => { reload(); reloadSubs(); }} />}
    {deleting && <DeleteSubModal s={deleting} nodes={proxies.filter(p => p.subId === deleting.id).length} inUse={users[deleting.id] || 0} onClose={() => setDeleting(null)} onDone={() => { reload(); reloadSubs(); }} />}
  </>;
};

const AddSubModal: React.FC<{ onClose: () => void; onDone: (id: string) => void }> = ({ onClose, onDone }) => {
  const toast = useToast();
  const [url, setUrl] = useState('');
  const [name, setName] = useState('');
  const [pv, setPv] = useState<SubPreview | null>(null);
  const [err, setErr] = useState('');
  const [phase, setPhase] = useState<'' | 'preview' | 'save'>('');
  const [out, close] = useExit(onClose);
  const valid = /^https?:\/\/\S+$/i.test(url.trim());
  useEffect(() => { setPv(null); setErr(''); }, [url]);
  const preview = async () => {
    setPhase('preview'); setErr('');
    try { setPv(await api.PreviewSubscription(url.trim(), '') as any); }
    catch (e: any) { setErr(String(e?.message || e)); }
    finally { setPhase(''); }
  };
  const save = async () => {
    setPhase('save');
    try { const s = await api.AddSubscription(url.trim(), name.trim()); toast(`已添加订阅「${s.name}」：${s.nodeCount} 个节点`, 'ok'); onDone(s.id); close(); }
    catch (e: any) { setErr(String(e?.message || e)); setPhase(''); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 'min(620px, 94vw)' }}>
        <div className="dhead"><div className="glyph"><Icon.link s={18} /></div><div><h3>添加订阅</h3><div className="sub">先解析预览节点数量，确认后再保存</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <Field label="订阅链接" hint="以 Clash Verge 的身份下载（User-Agent: clash-verge/v1.7），大多数机场会返回 Clash 配置；也支持 Base64 / 分享链接列表">
            <div className="row"><input type="text" autoFocus value={url} onChange={e => setUrl(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && valid) preview(); }} placeholder="https://example.com/api/v1/client/subscribe?token=…" />
              <button className="btn" style={{ flex: 'none' }} disabled={!valid || !!phase} onClick={preview}>{phase === 'preview' ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 解析中…</> : <><Icon.search s={14} /> 解析预览</>}</button></div>
          </Field>
          <Field label="名称（可选）" hint="留空 = 使用订阅自带的名称（文件名）或域名"><input type="text" value={name} onChange={e => setName(e.target.value)} placeholder={pv?.name || '如 机场 A'} /></Field>
          {err && <div className="notice bad" style={{ margin: '4px 0 0' }}><Icon.alert /><span className="grow" style={{ wordBreak: 'break-all' }}>{err}</span></div>}
          {pv && <div className="geo-card sub-pv">
            <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, flexWrap: 'wrap' }}>
              <b style={{ fontSize: 15 }}>{pv.name}</b>
              <span className="sub">{pv.format === 'clash' ? 'Clash 配置' : '分享链接列表'} · <b className="num" style={{ color: 'var(--txt)' }}>{pv.count}</b> 个节点{pv.skipped ? ` · 跳过 ${pv.skipped} 个不支持的` : ''}</span>
            </div>
            <div className="pv-protos">{Object.entries(pv.protocols || {}).sort((a, b) => b[1] - a[1]).map(([k, n]) => <div key={k} className={`pv-proto ${DIRECT.has(k) ? '' : 'core'}`}><span>{protoName(k)}</span><b className="num">{n}</b></div>)}</div>
            <div className="sub" style={{ marginTop: 8, display: 'flex', gap: 14, flexWrap: 'wrap' }}>
              <span>流量：{(pv.upload || pv.download || pv.total) ? `${fmtBytes(pv.upload + pv.download)} / ${pv.total ? fmtBytes(pv.total) : '不限'}` : '未提供'}</span>
              <span>到期：{pv.expire ? dateText(pv.expire) : '长期'}</span>
              <span>自动更新：每 {pv.interval || 24} 小时{pv.interval ? '（订阅建议）' : ''}</span>
            </div>
            {pv.coreCount > 0 && <div className="sub" style={{ marginTop: 6, color: '#c9d1ff' }}><Icon.cpu s={12} /> 其中 {pv.coreCount} 个节点（{Object.keys(pv.protocols || {}).filter(k => !DIRECT.has(k)).map(protoName).join(' / ')}）需要代理内核 mihomo 转换后使用。</div>}
          </div>}
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" disabled={!pv || !!phase} onClick={save}>{phase === 'save' ? '保存中…' : <><Icon.check s={15} /> 添加{pv ? ` ${pv.count} 个节点` : ''}</>}</button></div>
      </div>
    </div>
  );
};

const HOURS: [string, string][] = [['0', '关闭'], ['6', '6 小时'], ['12', '12 小时'], ['24', '24 小时'], ['72', '3 天']];

const EditSubModal: React.FC<{ s: Subscription; onClose: () => void; onDone: () => void }> = ({ s, onClose, onDone }) => {
  const toast = useToast();
  const [v, setV] = useState<Subscription>({ ...s });
  const [out, close] = useExit(onClose);
  const set = (x: Partial<Subscription>) => setV(o => ({ ...o, ...x } as Subscription));
  const preset = HOURS.some(h => +h[0] === v.autoUpdateHours);
  const save = async () => {
    try { await api.EditSubscription(v as any); toast('已保存'); onDone(); close(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal" style={{ width: 'min(600px, 94vw)' }}>
        <div className="dhead"><div className="glyph"><Icon.edit s={18} /></div><div><h3>编辑订阅</h3><div className="sub">{s.nodeCount} 个节点 · 改名会同步修改分组名</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <Field label="名称"><input type="text" value={v.name} onChange={e => set({ name: e.target.value })} /></Field>
          <Field label="订阅链接"><input type="text" value={v.url} onChange={e => set({ url: e.target.value })} /></Field>
          <Field label="User-Agent" hint="决定订阅返回的格式：clash-verge / clash.meta 返回 Clash 配置，留空恢复默认"><input type="text" value={v.ua} onChange={e => set({ ua: e.target.value })} placeholder="clash-verge/v1.7" /></Field>
          <Field label="自动更新" hint={v.autoUpdateHours ? `每 ${v.autoUpdateHours} 小时在后台更新一次，失败 30 分钟后重试` : '只在手动点「更新」时下载'}>
            <div className="row" style={{ gap: 10 }}>
              <Segmented className="block" items={HOURS} value={(preset ? String(v.autoUpdateHours) : '') as any} onChange={h => set({ autoUpdateHours: +h })} />
              <div className="row" style={{ flex: 'none', gap: 6, alignItems: 'center' }}><input type="number" min={0} max={720} style={{ width: 76 }} value={v.autoUpdateHours} onChange={e => set({ autoUpdateHours: Math.max(0, Math.min(720, +e.target.value || 0)) })} /><span className="sub">小时</span></div>
            </div>
          </Field>
          <Toggle on={v.enabled} onChange={x => set({ enabled: x })} label="启用" hint="停用后不再自动更新，「全部更新」也会跳过；已有节点照常可用" />
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" onClick={save}><Icon.check s={15} /> 保存</button></div>
      </div>
    </div>
  );
};

const DeleteSubModal: React.FC<{ s: Subscription; nodes: number; inUse: number; onClose: () => void; onDone: () => void }> = ({ s, nodes, inUse, onClose, onDone }) => {
  const toast = useToast();
  const [withNodes, setWithNodes] = useState(inUse === 0);
  const [out, close] = useExit(onClose);
  const go = async () => {
    try { await api.DeleteSubscription(s.id, withNodes); toast(withNodes ? `已删除订阅和 ${nodes} 个节点` : `已删除订阅，${nodes} 个节点保留为手动代理`); onDone(); close(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal sm">
        <div className="dhead"><div className="glyph bad"><Icon.trash s={18} /></div><div><h3>删除订阅「{s.name}」？</h3></div></div>
        <div className="dbody">
          <Toggle on={withNodes} onChange={setWithNodes} label={`同时删除它的 ${nodes} 个节点`}
            hint={withNodes ? (inUse ? `⚠ ${inUse} 个环境正在使用这些节点，删除后改为直连，下次启动时生效` : '没有环境在使用这些节点') : '节点保留在代理列表里，变成手动代理，不再随订阅更新'} />
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn solid-bad" onClick={go}>删除</button></div>
      </div>
    </div>
  );
};

// ---- settings card ----
export const CoreCard: React.FC = () => {
  const toast = useToast();
  const [st, setSt] = useState<CoreStatus | null>(null);
  const [path, setPath] = useState('');
  const [dl, setDl] = useState(false);
  const load = () => api.CoreStatus().then(x => { setSt(x as any); setPath((x as any).managed ? '' : (x as any).path || ''); }).catch(() => {});
  useEffect(() => { load(); return onEvent('core:status', (x: CoreStatus) => setSt(x)); }, []);
  const download = async () => {
    setDl(true);
    try { const x = await api.DownloadCore(); setSt(x as any); toast(`mihomo ${x.version} 已安装`, 'ok'); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); load(); }
    finally { setDl(false); }
  };
  const usePath = async (p: string) => {
    try { const x = await api.SetCorePath(p); setSt(x as any); toast(p ? `已使用本地内核 ${x.version}` : '已恢复使用下载的内核', 'ok'); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const pick = async () => { const f = await api.PickCorePath(); if (f) { setPath(f); usePath(f); } };
  const stop = async () => { setSt(await api.StopCore() as any); toast('内核已停止'); };
  const busy = dl || st?.downloading;
  return (
    <div className="panel pad setting-card spot rise" id="settings-core" style={{ ['--i' as any]: 1 }}>
      <div className="head"><span className="ic"><Icon.cpu /></span><div><h4>代理内核 (mihomo)</h4><span>让浏览器使用订阅里的 VLESS / VMess / SS / Trojan / Hysteria2 / TUIC / AnyTLS 节点</span></div></div>
      <div className={`geo-card ${st?.installed ? '' : 'warn'}`}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          <span className={`badge ${st?.installed ? 'ok' : 'unknown'}`}><span className="d" />{st ? (st.installed ? `已安装 ${st.version}` : busy ? '下载中' : '未安装') : '检测中…'}</span>
          {st?.installed && <span className={`badge ${st.running ? 'ok' : 'unknown'}`}><span className="d" />{st.running ? `运行中 · ${st.listeners} 个节点端口` : '未运行（有环境使用内核节点时自动启动）'}</span>}
        </div>
        {busy && <div className="env-bar" style={{ marginTop: 8 }}><i style={{ width: `${Math.max(st?.progress || 0, 2)}%` }} /></div>}
        {st?.error && <div className="sub" style={{ color: '#ffb3bc', marginTop: 4, wordBreak: 'break-all' }}>{st.error}</div>}
        <div className="sub" style={{ marginTop: 4 }}>程序：<code className="mono">{st?.path || '未找到'}</code>{st?.installed && (st.managed ? ' · 由 GhostBrowser 下载' : ' · 本地文件')}</div>
        <div className="sub" style={{ marginTop: 2 }}>每个在用节点一个本机端口（127.0.0.1，仅本机可连），不修改系统代理和 DNS；没有环境使用时自动停止。</div>
      </div>
      <div className="core-actions">
        <button className="btn primary" disabled={!!busy} onClick={download}>{busy ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 下载中 {Math.round(st?.progress || 0)}%</> : <><Icon.download s={14} /> {st?.installed && st.managed ? '更新到最新版' : '一键下载最新版'}</>}</button>
        {st?.running && <button className="btn ghost" onClick={stop}><Icon.stop s={13} /> 停止内核</button>}
      </div>
      <Field label="使用本地 mihomo 程序（可选）" hint="已经有 mihomo / Clash Meta 内核时可直接选择，会先运行 -v 校验；留空 = 使用下载的版本">
        <div className="row"><input type="text" value={path} onChange={e => setPath(e.target.value)} onBlur={() => { if (path !== ((st && !st.managed && st.path) || '')) usePath(path); }} placeholder="如 C:\\Program Files\\Clash Verge\\verge-mihomo.exe" />
          <button className="btn" style={{ flex: 'none' }} onClick={pick}><Icon.folder s={15} /> 浏览</button></div>
      </Field>
      <div className="sub" style={{ marginTop: 6 }}>mihomo 来自 <a href="https://github.com/MetaCubeX/mihomo" target="_blank" rel="noreferrer">MetaCubeX/mihomo</a>（GPL-3.0），作为独立程序运行，GhostBrowser 不包含它的代码。</div>
    </div>
  );
};

// ---- profile drawer: grouped, searchable proxy picker ----
export const ProxyPicker: React.FC<{ value: string; proxies: Proxy[]; onChange: (id: string) => void }> = ({ value, proxies, onChange }) => {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const cur = proxies.find(p => p.id === value);
  const groups = useMemo(() => {
    const s = q.trim().toLowerCase();
    const hit = (p: Proxy) => !s || [p.name, p.host, p.ip, p.country, p.city, p.subName, p.protocol, p.tag].some(v => (v || '').toLowerCase().includes(s));
    const m = new Map<string, { name: string; items: Proxy[] }>();
    m.set('', { name: '手动添加', items: [] });
    proxies.forEach(p => { if (!hit(p)) return; const k = groupKey(p); if (!m.has(k)) m.set(k, { name: p.subName || '订阅', items: [] }); m.get(k)!.items.push(p); });
    return [...m.entries()].filter(([, g]) => g.items.length);
  }, [proxies, q]);
  const label = (p: Proxy) => `${flag(p.country)} · ${p.name}`;
  return (
    <div className={`pick pickfield selfield ${open ? 'open' : ''}`} ref={ref}>
      <button type="button" className="pick-btn" onClick={() => setOpen(o => !o)}>
        <span className="pick-o">{cur ? <>{label(cur)} <span className="sub">· {protoOf(cur)}{cur.subName ? ` · ${cur.subName}` : ''}{cur.status === 'ok' ? ` · ${cur.city || cur.ip}` : ''}</span></> : value ? '代理已删除（直连）' : '不使用代理（直连）'}</span>
        <svg className="pick-c" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg>
      </button>
      <Floating open={open} anchor={ref} onClose={() => { setOpen(false); setQ(''); }} matchWidth className="pick-menu field-menu sel-menu px-pick-menu">
        {proxies.length > 8 && <div className="px-pick-search"><Icon.search s={13} /><input type="text" autoFocus value={q} onChange={e => setQ(e.target.value)} placeholder="搜索名称 / 国家 / 分组 / 协议" /></div>}
        <button type="button" className={!value ? 'on' : ''} onClick={() => { onChange(''); setOpen(false); }}><span className="pick-tick">{!value && <Tick />}</span><span className="pick-o">不使用代理（直连）</span></button>
        {groups.map(([k, g]) => <React.Fragment key={k || 'manual'}>
          <div className="px-pick-group"><span>{g.name}</span><span className="num">{g.items.length}</span></div>
          {g.items.map(p => <button type="button" key={p.id} className={p.id === value ? 'on' : ''} title={`${p.name} · ${p.host}:${p.port}`} onClick={() => { onChange(p.id); setOpen(false); setQ(''); }}>
            <span className="pick-tick">{p.id === value && <Tick />}</span>
            <span className="pick-o">{label(p)}</span>
            <span className={`px-pick-proto ${isCore(p) ? 'core' : ''}`}>{protoOf(p)}</span>
            {p.gone ? <span className="px-pick-st bad">已失效</span> : p.status === 'ok' ? <span className="px-pick-st ok">{p.latency ? `${p.latency}ms` : '可用'}</span> : p.status === 'fail' ? <span className="px-pick-st bad">不可用</span> : null}
          </button>)}
        </React.Fragment>)}
        {groups.length === 0 && <div className="sub" style={{ padding: 12, textAlign: 'center' }}>没有匹配的代理</div>}
      </Floating>
    </div>
  );
};
const Tick = () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>;
