import React, { useMemo, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Proxy } from './api';
import { Icon, Field, ago, useToast, useConfirm, useExit, CC, Latency, StatCard } from './ui';

export const ProxyView: React.FC<{ proxies: Proxy[]; reload: () => void }> = ({ proxies, reload }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [q, setQ] = useState('');
  const [importOpen, setImportOpen] = useState(false);
  const [checking, setChecking] = useState<Set<string>>(new Set());
  const [checkingAll, setCheckingAll] = useState(false);

  const list = useMemo(() => proxies.filter(p => !q || p.name.toLowerCase().includes(q.toLowerCase()) || (p.ip || '').includes(q) || (p.country || '').toLowerCase().includes(q.toLowerCase())), [proxies, q]);

  const check = async (p: Proxy) => {
    setChecking(s => new Set(s).add(p.id));
    const r = await api.CheckProxy(p.id);
    setChecking(s => { const n = new Set(s); n.delete(p.id); return n; });
    toast(r.status === 'ok' ? `${p.name} 正常 · ${r.city || r.ip} · ${r.latency}ms` : `${p.name} 不可用`, r.status === 'ok' ? 'ok' : 'bad');
    reload();
  };
  const checkAll = async () => { setCheckingAll(true); await api.CheckAllProxies(); setCheckingAll(false); toast('全部检测完成'); reload(); };
  const del = async (id: string) => { if (!await ask({ title: '删除这个代理？', body: '正在使用它的环境会改为直连，下次启动时生效。', ok: '删除', danger: true })) return; await api.DeleteProxy(id); reload(); };

  if (proxies.length === 0) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="searching" size={64} theme="dark" /></div>
      <h3>还没有代理</h3>
      <div>粘贴一批代理即可批量导入，支持 <code className="mono">host:port:user:pass</code> 等常见格式。</div>
      <button className="btn primary" onClick={() => setImportOpen(true)}><Icon.upload s={15} /> 导入代理</button>
      {importOpen && <ImportModal onClose={() => setImportOpen(false)} onDone={reload} />}
    </div>
  );

  const okCount = proxies.filter(p => p.status === 'ok').length;

  return (
    <>
      <div className="stat-row">
        <StatCard l="代理总数" n={proxies.length} ic="globe" pct={100} />
        <StatCard l="可用" n={okCount} of={proxies.length} ic="check" tone="ok" pct={okCount / proxies.length * 100} />
        <StatCard l="不可用 / 未检测" n={proxies.length - okCount} of={proxies.length} ic="alert" tone="bad" pct={(proxies.length - okCount) / proxies.length * 100} />
        <StatCard l="覆盖国家 / 地区" n={new Set(proxies.filter(p => p.country).map(p => p.country)).size} ic="shield" tone="cy" pct={60} />
      </div>
      <div className="toolbar">
        <div className="search"><Icon.search /><input type="text" placeholder="搜索名称 / IP / 国家" value={q} onChange={e => setQ(e.target.value)} /></div>
        <button className="btn" onClick={checkAll} disabled={checkingAll}>{checkingAll ? <><ThinkingOrb state="searching" size={20} theme="dark" /> 检测中…</> : <><Icon.refresh s={14} /> 全部检测</>}</button>
        <div className="spacer" />
        <button className="btn primary" onClick={() => setImportOpen(true)}><Icon.upload s={15} /> 导入代理</button>
      </div>

      <div className="panel list">
        <table>
          <thead><tr><th>名称</th><th>类型</th><th>出口 IP</th><th>位置</th><th>延迟</th><th>状态</th><th>检测时间</th><th style={{ textAlign: 'right' }}>操作</th></tr></thead>
          <tbody>
            {list.map((p, i) => <tr key={p.id} className="rise" style={{ ['--i' as any]: i }}>
              <td><div className="nm">{p.name}</div><div className="sub num">{p.host}:{p.port}</div></td>
              <td><span className="tag num" style={{ textTransform: 'uppercase' }}>{p.type}</span></td>
              <td className="sub num">{p.ip || '—'}</td>
              <td>{p.country ? <><CC cc={p.country} /><span>{p.city || p.country}</span></> : <span className="sub">—</span>}</td>
              <td><Latency ms={p.latency} /></td>
              <td><span className={`badge ${p.status}`}><i className="d" />{p.status === 'ok' ? '可用' : p.status === 'fail' ? '不可用' : '未检测'}</span></td>
              <td className="sub">{ago(p.checkedAt)}</td>
              <td><div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                {checking.has(p.id) ? <span style={{ width: 26, display: 'grid', placeItems: 'center' }}><ThinkingOrb state="searching" size={20} theme="dark" /></span>
                  : <button className="btn sm" onClick={() => check(p)}><Icon.bolt s={13} /> 检测</button>}
                <button className="btn ghost icon danger" onClick={() => del(p.id)}><Icon.trash s={15} /></button>
              </div></td>
            </tr>)}
          </tbody>
        </table>
      </div>
      {importOpen && <ImportModal onClose={() => setImportOpen(false)} onDone={reload} />}
    </>
  );
};

const ImportModal: React.FC<{ onClose: () => void; onDone: () => void }> = ({ onClose, onDone }) => {
  const toast = useToast();
  const [text, setText] = useState('');
  const [type, setType] = useState('http');
  const [busy, setBusy] = useState(false);
  const go = async () => {
    const lines = text.split('\n').map(l => l.trim()).filter(Boolean);
    if (!lines.length) { toast('请粘贴至少一行', 'bad'); return; }
    setBusy(true);
    const prefixed = lines.map(l => l.includes('://') ? l : `${type}://${l}`).join('\n');
    const added = await api.ImportProxies(prefixed);
    setBusy(false); toast(`已导入 ${added.length} 个代理`); onDone(); close();
  };
  const [out, close] = useExit(onClose);
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal">
        <div className="dhead"><div className="glyph"><Icon.upload s={18} /></div><div><h3>导入代理</h3><div className="sub">粘贴即可，格式会自动识别</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <Field label="默认类型 (当某行未写协议时)"><select value={type} onChange={e => setType(e.target.value)}><option value="http">HTTP</option><option value="https">HTTPS</option><option value="socks5">SOCKS5</option></select></Field>
          <Field label="代理列表 (每行一个)" hint="host:port · host:port:user:pass · user:pass@host:port · socks5://user:pass@host:port">
            <textarea style={{ minHeight: 180 }} value={text} onChange={e => setText(e.target.value)} placeholder={'45.12.30.8:1080:user:pass\nsocks5://u:p@1.2.3.4:1080'} />
          </Field>
        </div>
        <div className="dfoot"><span className="sub num" style={{ fontSize: 11.5, color: 'var(--txt-3)' }}>{text.split('\n').filter(l => l.trim()).length} 行</span><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" onClick={go} disabled={busy}>{busy ? '导入中…' : '导入'}</button></div>
      </div>
    </div>
  );
};
