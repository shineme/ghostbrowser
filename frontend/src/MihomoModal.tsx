import React, { useEffect, useMemo, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api } from './api';
import { Icon, Latency, Segmented, useExit, useToast, Field } from './ui';

type Status = { configured: boolean; connected: boolean; controller: string; hasSecret: boolean; version: string; meta: boolean; mode: string; listenHost: string; listenPort: number; listenKind: string; error: string };
type Node = { name: string; type: string; now?: string; all?: string[]; delay: number; alive: boolean; udp: boolean; isGroup: boolean };

const MODE: Record<string, string> = { rule: '规则模式', global: '全局模式', direct: '直连模式' };
const GTYPE: Record<string, string> = { Selector: '手动选择', URLTest: '自动测速', Fallback: '故障转移', LoadBalance: '负载均衡', Relay: '链式代理' };

/** Connect to a running Clash.Meta / Mihomo controller, switch nodes and expose its listener as a proxy. */
export const MihomoModal: React.FC<{ onClose: () => void; onAdded?: () => void }> = ({ onClose, onAdded }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [st, setSt] = useState<Status | null>(null);
  const [ctl, setCtl] = useState('http://127.0.0.1:9090');
  const [secret, setSecret] = useState('');
  const [connecting, setConnecting] = useState(false);
  const [err, setErr] = useState('');
  const [groups, setGroups] = useState<Node[]>([]);
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState<string>('');
  const [testing, setTesting] = useState<Set<string>>(new Set());
  const [fails, setFails] = useState<Record<string, string>>({});
  const [switching, setSwitching] = useState('');
  const [adding, setAdding] = useState(false);
  const [edit, setEdit] = useState(false);
  const [view, setView] = useState<'groups' | 'nodes'>('groups');

  const loadProxies = async () => {
    setLoading(true);
    try { const r: any = await api.MihomoProxies(); setGroups(r.groups || []); setNodes(r.nodes || []); setOpen(o => o || (r.groups || []).find((g: Node) => g.type === 'Selector')?.name || ''); setErr(''); }
    catch (e: any) { setErr(String(e?.message || e)); }
    finally { setLoading(false); }
  };
  useEffect(() => {
    api.MihomoStatus().then((s: any) => {
      setSt(s); if (s.controller) setCtl(s.controller);
      if (s.connected) loadProxies(); else if (s.error) setErr(s.error);
    }).catch(() => setSt(null));
    // eslint-disable-next-line
  }, []);

  const connect = async () => {
    setConnecting(true); setErr('');
    try { const s: any = await api.MihomoConnect(ctl.trim(), secret); setSt(s); setEdit(false); setSecret(''); toast(`已连接 Mihomo ${s.version || ''}`); loadProxies(); }
    catch (e: any) { setErr(String(e?.message || e)); }
    finally { setConnecting(false); }
  };
  const mark = (n: string, on: boolean) => setTesting(s => { const x = new Set(s); on ? x.add(n) : x.delete(n); return x; });
  const test = async (name: string) => {
    mark(name, true);
    try { const ms = await api.MihomoTestDelay(name); setNodes(l => l.map(n => n.name === name ? { ...n, delay: ms, alive: true } : n)); setFails(f => { const c = { ...f }; delete c[name]; return c; }); }
    catch (e: any) { setNodes(l => l.map(n => n.name === name ? { ...n, delay: 0, alive: false } : n)); setFails(f => ({ ...f, [name]: String(e?.message || e) })); }
    finally { mark(name, false); }
  };
  const testAll = async (names = nodes.map(n => n.name)) => {
    const q = [...names];
    await Promise.all(Array.from({ length: Math.min(6, q.length) }, async () => { while (q.length) await test(q.shift()!); }));
  };
  const select = async (g: Node, node: string) => {
    if (g.now === node) return;
    setSwitching(`${g.name}\u0000${node}`);
    try { await api.MihomoSelect(g.name, node); setGroups(l => l.map(x => x.name === g.name ? { ...x, now: node } : x)); toast(`${g.name} → ${node}`); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setSwitching(''); }
  };
  const addProxy = async () => {
    setAdding(true);
    try { const p: any = await api.MihomoAddProxy(); toast(`已添加代理：${p.name}`); onAdded?.(); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
    finally { setAdding(false); }
  };

  const byName = useMemo(() => Object.fromEntries([...nodes, ...groups].map(n => [n.name, n])), [nodes, groups]);
  const delayOf = (name: string): number => { const n = byName[name]; if (!n) return 0; if (n.isGroup && n.now) return delayOf(n.now); return n.alive ? n.delay : 0; };
  const connected = !!st?.connected && !edit;
  const alive = nodes.filter(n => n.alive && n.delay).length;
  const listen = st?.listenPort ? `${st.listenKind || 'mixed'} ${st.listenHost || '127.0.0.1'}:${st.listenPort}` : '';

  const badge = (name: string) => {
    if (testing.has(name)) return <span className="mh-testing"><ThinkingOrb state="searching" size={20} theme="dark" /></span>;
    if (fails[name] || (byName[name] && !byName[name].isGroup && !byName[name].alive)) return <span className="badge fail" title={fails[name] || ''}>超时</span>;
    return <Latency ms={delayOf(name)} />;
  };

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal tall" style={{ width: 'min(860px, 94vw)' }}>
        <div className="dhead">
          <div className="glyph"><Icon.server s={18} /></div>
          <div style={{ minWidth: 0 }}><h3>Clash / Mihomo</h3><div className="sub">连接本机或局域网的 Mihomo（Clash.Meta）控制器，切换节点并把它的监听端口作为代理使用</div></div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div>
        </div>
        <div className="dbody">
          {st === null && <div className="empty" style={{ padding: 40 }}><ThinkingOrb state="searching" size={32} theme="dark" /><div>正在读取连接状态…</div></div>}

          {st !== null && !connected && <div className="mh-connect">
            <div className="grid2">
              <Field label="控制器地址 (external-controller)" hint="配置文件里的 external-controller，默认 127.0.0.1:9090">
                <input type="text" value={ctl} onChange={e => { setCtl(e.target.value); setErr(''); }} onKeyDown={e => e.key === 'Enter' && connect()} placeholder="http://127.0.0.1:9090" spellCheck={false} autoFocus />
              </Field>
              <Field label="密钥 (secret)" hint={st.hasSecret ? '已保存密钥；留空则继续使用已保存的' : '配置文件里的 secret，未设置可留空'}>
                <input type="password" value={secret} onChange={e => { setSecret(e.target.value); setErr(''); }} onKeyDown={e => e.key === 'Enter' && connect()} placeholder={st.hasSecret ? '••••••••' : '未设置可留空'} />
              </Field>
            </div>
            {err && <div className="notice bad"><Icon.alert /><span className="grow">{err}</span></div>}
            <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <span className="sub" style={{ fontSize: 12 }}>支持 Clash Verge Rev、Mihomo Party、FlClash 等客户端内置的 Mihomo 内核</span>
              <div className="spacer" style={{ flex: 1 }} />
              {edit && <button className="btn" onClick={() => { setEdit(false); setErr(''); }}>取消</button>}
              <button className="btn primary" onClick={connect} disabled={connecting || !ctl.trim()}>{connecting ? <><ThinkingOrb state="connecting" size={20} theme="dark" /> 连接中…</> : <><Icon.bolt s={15} /> 连接</>}</button>
            </div>
          </div>}

          {connected && st && <>
            <div className="mh-status">
              <span className="badge ok"><i className="d" />已连接</span>
              <span className="mh-kv"><span>版本</span><b className="num">{st.version || '—'}</b>{st.meta && <span className="tag" style={{ marginLeft: 6 }}>Meta</span>}</span>
              <span className="mh-kv"><span>模式</span><b>{MODE[st.mode] || st.mode || '—'}</b></span>
              <span className="mh-kv"><span>监听</span><b className="num">{listen || '未开启'}</b></span>
              <span className="mh-kv ellip" title={st.controller}><span>控制器</span><b className="num">{st.controller}</b></span>
              <div className="spacer" style={{ flex: 1 }} />
              <button className="btn sm ghost" onClick={() => { setEdit(true); setErr(''); }}><Icon.edit s={13} /> 修改</button>
            </div>
            <div className="notice warn"><Icon.alert /><span className="grow">所有使用 <b className="num">{listen || '该监听端口'}</b> 的环境共享同一个出口节点：在这里切换节点会同时影响它们。需要不同出口的环境请使用不同的代理。</span></div>
            {err && <div className="notice bad"><Icon.alert /><span className="grow">{err}</span><button className="btn sm" onClick={loadProxies}>重试</button></div>}

            <div className="mh-bar">
              <Segmented value={view} onChange={setView} items={[['groups', `策略组 ${groups.length}`, <Icon.layers s={14} />], ['nodes', `节点 ${nodes.length}`, <Icon.globe s={14} />]]} />
              <span className="sub num" style={{ fontSize: 12 }}>{alive} / {nodes.length} 个节点可用</span>
              <div className="spacer" style={{ flex: 1 }} />
              <button className="btn sm" onClick={loadProxies} disabled={loading}><Icon.refresh s={13} /> 刷新</button>
              <button className="btn sm" onClick={() => testAll()} disabled={testing.size > 0 || !nodes.length}>{testing.size ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 测速中 {testing.size}</> : <><Icon.bolt s={13} /> 全部测速</>}</button>
            </div>

            {loading && !groups.length && <div className="empty" style={{ padding: 40 }}><ThinkingOrb state="searching" size={32} theme="dark" /><div>正在读取节点…</div></div>}

            {view === 'groups' && groups.length > 0 && <div className="mh-groups">
              {groups.map(g => {
                const isOpen = open === g.name; const manual = g.type === 'Selector';
                return <div key={g.name} className={`mh-group panel ${isOpen ? 'open' : ''}`}>
                  <button className="mh-ghead" onClick={() => setOpen(isOpen ? '' : g.name)}>
                    <span className="mh-gname ellip">{g.name}</span>
                    <span className={`tag ${manual ? '' : 'auto'}`}>{GTYPE[g.type] || g.type}</span>
                    <span className="mh-now ellip"><Icon.check s={12} />{g.now || '—'}</span>
                    {badge(g.now || '')}
                    <svg className="mh-chev" viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg>
                  </button>
                  {isOpen && <div className="mh-members">
                    {!manual && <div className="sub" style={{ gridColumn: '1 / -1', fontSize: 11.5 }}>{GTYPE[g.type] || g.type}策略组由 Mihomo 自动选择节点，不能手动切换</div>}
                    {(g.all || []).map(n => {
                      const on = g.now === n; const sw = switching === `${g.name}\u0000${n}`;
                      return <button key={n} className={`mh-node ${on ? 'on' : ''}`} disabled={!manual || !!switching} onClick={() => select(g, n)} title={n}>
                        <span className="mh-nname ellip">{byName[n]?.isGroup ? <Icon.layers s={12} /> : null}{n}</span>
                        <span className="mh-nmeta">{sw ? <ThinkingOrb state="connecting" size={20} theme="dark" /> : <>{byName[n] && <span className="sub">{byName[n].isGroup ? '策略组' : byName[n].type}</span>}{badge(n)}</>}</span>
                      </button>;
                    })}
                  </div>}
                </div>;
              })}
            </div>}

            {view === 'nodes' && nodes.length > 0 && <div className="panel list mh-nodes">
              <table>
                <thead><tr><th>节点</th><th>协议</th><th>UDP</th><th>延迟</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
                <tbody>{nodes.map(n => <tr key={n.name}>
                  <td><div className="nm ellip" style={{ maxWidth: 360 }} title={n.name}>{n.name}</div>{groups.filter(g => g.now === n.name).length > 0 && <div className="sub" style={{ fontSize: 11 }}>正在被 {groups.filter(g => g.now === n.name).map(g => g.name).join('、')} 使用</div>}</td>
                  <td><span className="tag num">{n.type}</span></td>
                  <td>{n.udp ? <span className="badge ok">支持</span> : <span className="badge unknown">不支持</span>}</td>
                  <td>{badge(n.name)}</td>
                  <td style={{ textAlign: 'right' }}><button className="btn sm" disabled={testing.has(n.name)} onClick={() => test(n.name)}><Icon.bolt s={13} /> 测速</button></td>
                </tr>)}</tbody>
              </table>
            </div>}
            {!loading && !err && groups.length === 0 && nodes.length === 0 && <div className="sub" style={{ padding: 24, textAlign: 'center' }}>Mihomo 没有返回任何节点，请检查配置文件里的 proxies 和 proxy-groups</div>}
          </>}
        </div>
        <div className="dfoot">
          <span className="sub" style={{ fontSize: 11.5 }}>{connected ? `添加后会在代理列表里新增 ${listen ? listen.split(' ')[1] : 'Mihomo 监听端口'}，可分配给环境` : '连接信息只保存在本机'}</span>
          <div className="spacer" />
          <button className="btn" onClick={() => close()}>关闭</button>
          {connected && <button className="btn primary" disabled={adding || !st?.listenPort} onClick={addProxy} title={st?.listenPort ? '' : 'Mihomo 没有开启 mixed-port / port / socks-port'}>{adding ? '添加中…' : <><Icon.plus s={15} /> 添加为代理</>}</button>}
        </div>
      </div>
    </div>
  );
};
