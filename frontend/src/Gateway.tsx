// 代理商账号：fill in the provider's sub-account name and password once, pick
// everything else from lists, and get ready-made gateway proxies. 711Proxy and
// Cliproxy put country / state / sticky session in the username; GhostBrowser
// builds it, and "换 IP" later just rolls the session value (gateway.go).
import React, { useEffect, useMemo, useState } from 'react';
import { api, Proxy } from './api';
import { Icon, Field, Toggle, useToast, useExit, Segmented, SelectField } from './ui';
import { COUNTRIES, SearchSelect } from './locales';

type ProvID = '711' | 'cli';
export type GwAcct = { user: string; pass: string; cc?: string; state?: string; mins?: number; entry?: string; proto?: string };
type Provider = {
  name: string; site: string; domain: string;
  entries: [string, string][];           // gateway host -> label
  port: number; durations: number[]; random: boolean; sessKey: 'session' | 'sid';
  build: (o: { user: string; cc: string; st: string; sess: string; mins: number }) => string;
  base: (u: string) => string;           // account name from a built username
};

const US_STATES = ['Alabama', 'Alaska', 'Arizona', 'Arkansas', 'California', 'Colorado', 'Connecticut', 'Delaware', 'Florida', 'Georgia', 'Hawaii', 'Idaho', 'Illinois', 'Indiana', 'Iowa', 'Kansas', 'Kentucky', 'Louisiana', 'Maine', 'Maryland', 'Massachusetts', 'Michigan', 'Minnesota', 'Mississippi', 'Missouri', 'Montana', 'Nebraska', 'Nevada', 'New Hampshire', 'New Jersey', 'New Mexico', 'New York', 'North Carolina', 'North Dakota', 'Ohio', 'Oklahoma', 'Oregon', 'Pennsylvania', 'Rhode Island', 'South Carolina', 'South Dakota', 'Tennessee', 'Texas', 'Utah', 'Vermont', 'Virginia', 'Washington', 'West Virginia', 'Wisconsin', 'Wyoming'];
// 711 writes states as one capitalised word (Newsouthwales, Saopaulo); Cliproxy as shown in its dashboard, spaces removed.
const st711 = (s: string) => { const w = s.replace(/\s+/g, '').toLowerCase(); return w.charAt(0).toUpperCase() + w.slice(1); };
const stCli = (s: string) => s.replace(/\s+/g, '');

export const PROVIDERS: Record<ProvID, Provider> = {
  '711': {
    name: '711Proxy', site: '711proxy.com', domain: '711proxy.com',
    entries: [['global.rotgb.711proxy.com', '全球（默认）'], ['us.rotgb.711proxy.com', '北美'], ['eu.rotgb.711proxy.com', '欧洲'], ['as.rotgb.711proxy.com', '亚洲'], ['ina.rotgb.711proxy.com', '东南亚']],
    port: 10000, durations: [5, 10, 15, 30, 60, 120, 180], random: true, sessKey: 'session',
    build: ({ user, cc, st, sess, mins }) => `${user}-zone-custom${cc ? `-region-${cc}` : ''}${cc && st ? `-st-${st711(st)}` : ''}${sess ? `-session-${sess}-sessTime-${mins}` : ''}`,
    base: u => u.split('-zone-')[0],
  },
  cli: {
    name: 'Cliproxy', site: 'cliproxy.com', domain: 'cliproxy.io',
    entries: [['us.cliproxy.io', '美国中转（美洲 / 欧洲）'], ['sg.cliproxy.io', '新加坡中转（亚洲）']],
    port: 3010, durations: [5, 10, 15, 30, 60, 120], random: false, sessKey: 'sid',
    build: ({ user, cc, st, sess, mins }) => `${user}-region-${cc || 'US'}${st ? `-st-${stCli(st)}` : ''}${sess ? `-sid-${sess}-t-${mins}` : ''}`,
    base: u => u.split('-region-')[0],
  },
};

const DIG = '0123456789', ALNUM = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789';
const rnd = (n: number, a: string) => { const b = new Uint32Array(n); crypto.getRandomValues(b); return [...b].map(x => a[x % a.length]).join(''); };
const newSess = (p: Provider) => p.sessKey === 'session' ? rnd(8, DIG) : rnd(8, ALNUM);

export const provOf = (p: Proxy): ProvID | '' => (Object.keys(PROVIDERS) as ProvID[]).find(k => (p.host || '').endsWith(PROVIDERS[k].domain)) || '';

// Shared form state: used by the 代理商账号 dialog (batch) and inline in the
// environment drawer (one proxy for that environment).
export function useGateway(proxies: Proxy[], compact: boolean) {
  const [prov, setProv] = useState<ProvID>('711');
  const P = PROVIDERS[prov];
  // the last account used with this provider fills itself in
  const known = useMemo(() => {
    const m: Partial<Record<ProvID, { user: string; pass: string }>> = {};
    for (const p of [...proxies].reverse()) { const k = provOf(p); if (k && !m[k] && p.user) m[k] = { user: PROVIDERS[k].base(p.user), pass: p.pass || '' }; }
    return m;
  }, [proxies]);
  const [user, setUser] = useState(known['711']?.user || '');
  const [pass, setPass] = useState(known['711']?.pass || '');
  const [entry, setEntry] = useState(P.entries[0][0]);
  const [proto, setProto] = useState<'http' | 'socks5'>('http');
  const [cc, setCc] = useState('US');
  const [st, setSt] = useState('');
  const [sticky, setSticky] = useState<'sticky' | 'rotate'>('sticky');
  const [mins, setMins] = useState('30');
  const [count, setCount] = useState('5');
  const [rotate, setRotate] = useState(false);
  const [showPw, setShowPw] = useState(false);
  // accounts saved on the settings page win over "last used"
  const [saved, setSaved] = useState<Record<string, GwAcct>>({});
  const [editAcct, setEditAcct] = useState(false);
  const apply = (k: ProvID, g: Record<string, GwAcct>) => {
    const a = g[k], L = known[k], Pk = PROVIDERS[k];
    if (a?.user) { setUser(a.user); setPass(a.pass || ''); } else if (L) { setUser(L.user); setPass(L.pass); }
    setEntry(a?.entry && Pk.entries.some(e => e[0] === a.entry) ? a.entry : Pk.entries[0][0]);
    if (a?.proto) setProto(a.proto as any);
    if (a?.cc && (a.cc !== '*' || Pk.random)) { setCc(a.cc); setSt(a.cc === 'US' ? a.state || '' : ''); } else if (!Pk.random && cc === '*') setCc('US');
    setMins(String(a?.mins && Pk.durations.includes(a.mins) ? a.mins : Pk.durations.includes(+mins) ? mins : 30));
    setEditAcct(false);
  };
  useEffect(() => {
    api.GetSettings().then(stt => {
      const g = ((stt as any).gateways || {}) as Record<string, GwAcct>;
      setSaved(g);
      if (!g['711']?.user && g.cli?.user) setProv('cli'); else apply(prov, g);
    }).catch(() => {});
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const first = React.useRef(true);
  useEffect(() => {
    if (first.current) { first.current = false; return; }
    apply(prov, saved);
  }, [prov]); // eslint-disable-line react-hooks/exhaustive-deps
  const fromSettings = !!saved[prov]?.user && !editAcct;

  const countryOpts = useMemo(() => [...(P.random ? [['*', '随机（全球混播）'] as [string, string]] : []), ...COUNTRIES.map(c => [c.cc, `${c.cc} · ${c.name}`] as [string, string])], [P]);
  const stateOpts: [string, string][] = [['', '随机'], ...US_STATES.map(s => [s, s] as [string, string])];
  const u = user.trim().replace(/-.*$/, ''); // a pasted full username keeps only the account part
  const mk = () => P.build({ user: u, cc: cc === '*' ? '' : cc, st: cc === 'US' ? st : '', sess: sticky === 'sticky' ? newSess(P) : '', mins: +mins });
  const sample = useMemo(() => u ? `${entry}:${P.port}:${mk()}:${pass ? '••••••' : '密码'}` : '', [u, pass, entry, cc, st, sticky, mins, prov]); // eslint-disable-line react-hooks/exhaustive-deps
  const n = compact || sticky === 'rotate' ? 1 : +count;
  const label = `${P.name}-${cc === '*' ? '全球' : cc}${cc === 'US' && st ? '-' + stCli(st) : ''}`;
  const valid = !!u && !!pass.trim();
  const build = (k?: number) => Array.from({ length: k ?? n }, () => ({ type: proto, host: entry, port: P.port, user: mk(), pass: pass.trim(), name: `${entry}:${P.port}`, rotateOnLaunch: rotate && sticky === 'sticky', status: 'unknown' }));

  const fields = <>
    <Field label="服务商"><Segmented className="block" value={prov} onChange={setProv} items={[['711', '711Proxy · 动态住宅'], ['cli', 'Cliproxy · 流量套餐']]} /></Field>
    {fromSettings
      ? <div className="gw-acct"><Icon.lock s={14} /><span className="grow">账号 <b className="mono">{user}</b><span className="sub"> · 来自设置</span></span><button type="button" className="btn sm ghost" onClick={() => setEditAcct(true)}>这次换个账号</button></div>
      : <div className="grid2">
      <Field label="账户名" hint={known[prov] ? '已从上次使用的账号自动填入' : `${P.name} 控制台「子账户」里的账户名；可在 设置 → 代理商账号 保存，以后不用再填`}><input type="text" value={user} onChange={e => setUser(e.target.value)} placeholder="子账户名" autoFocus={!compact} spellCheck={false} /></Field>
      <Field label="密码"><div className="gw-pw"><input type={showPw ? 'text' : 'password'} value={pass} onChange={e => setPass(e.target.value)} placeholder="子账户密码" spellCheck={false} /><button type="button" className="btn sm ghost" onClick={() => setShowPw(x => !x)}>{showPw ? '隐藏' : '显示'}</button></div></Field>
    </div>}
    <div className="grid2">
      <Field label="国家 / 地区"><SearchSelect value={cc} options={countryOpts} onChange={v => { setCc(v); setSt(''); }} placeholder="搜索，如 US / 日本" /></Field>
      <Field label="州" hint={cc === 'US' ? '' : '目前仅美国可选州'}><SelectField value={cc === 'US' ? st : ''} onChange={setSt} options={stateOpts} disabled={cc !== 'US'} /></Field>
    </div>
    <div className={compact ? 'grid2' : 'grid3'}>
      <Field label="会话类型"><SelectField value={sticky} onChange={setSticky} options={[['sticky', '粘性 IP（推荐）'], ['rotate', '轮换 IP（每个请求换）']]} /></Field>
      <Field label="IP 保持时长"><SelectField value={mins} onChange={setMins} disabled={sticky === 'rotate'} options={P.durations.map(d => [String(d), d >= 60 ? `${d / 60} 小时` : `${d} 分钟`] as [string, string])} /></Field>
      {!compact && <Field label="生成数量"><SelectField value={sticky === 'rotate' ? '1' : count} onChange={setCount} disabled={sticky === 'rotate'} options={['1', '5', '10', '20', '50', '100'].map(x => [x, `${x} 个`] as [string, string])} /></Field>}
    </div>
    <div className="grid2">
      <Field label="接入节点"><SelectField value={entry} onChange={setEntry} options={P.entries.map(([h, l]) => [h, `${l} · ${h}`] as [string, string])} /></Field>
      <Field label="协议"><SelectField value={proto} onChange={setProto} options={[['http', 'HTTP（浏览器推荐）'], ['socks5', 'SOCKS5']]} /></Field>
    </div>
    {sticky === 'rotate' && <div className="notice warn"><Icon.alert /><span className="grow sub">轮换模式每个请求都换 IP，登录状态会掉，网站也容易触发风控。浏览器环境请用粘性 IP。</span></div>}
    {sticky === 'sticky' && <Toggle on={rotate} onChange={setRotate} label="每次启动环境前自动换 IP" hint="关闭时这个环境会一直用同一个会话；IP 超过保持时长后由服务商自动更换" />}
  </>;
  return { P, n, label, valid, build, fields, sample };
}

const Sample: React.FC<{ s: string }> = ({ s }) => s ? <div className="gw-sample"><span className="sub">预览</span><code title={s}>{s}</code></div> : null;

// Inline in the environment drawer: one proxy, created, checked and selected for this environment.
export const GatewayInline: React.FC<{ proxies: Proxy[]; onUse: (p: Proxy) => void; onCancel: () => void }> = ({ proxies, onUse, onCancel }) => {
  const toast = useToast();
  const g = useGateway(proxies, true);
  const [busy, setBusy] = useState(false);
  const go = async () => {
    if (!g.valid) { toast('请填写账户名和密码', 'bad'); return; }
    setBusy(true);
    try {
      const [saved] = await api.ImportProxyList(g.build() as any, g.label);
      const r = await api.CheckProxy(saved.id);
      onUse(r);
    } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setBusy(false); }
  };
  return <div className="gw-inline">
    {g.fields}
    <Sample s={g.sample} />
    <div className="gw-inline-foot">
      <span className="sub">会生成 1 个代理加入代理列表（标签 {g.label}），并给这个环境使用</span>
      <div className="spacer" />
      <button type="button" className="btn sm" onClick={onCancel}>收起</button>
      <button type="button" className="btn sm primary" disabled={busy || !g.valid} onClick={go}>{busy ? '生成并检测中…' : '生成并使用'}</button>
    </div>
  </div>;
};

export const GatewayModal: React.FC<{ proxies: Proxy[]; onClose: () => void; onDone: (ids: string[], check: boolean) => void }> = ({ proxies, onClose, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const g = useGateway(proxies, false);
  const [check, setCheck] = useState(true);
  const [busy, setBusy] = useState(false);
  const { P, n, label } = g;

  const go = async () => {
    if (!g.valid) { toast('请填写账户名和密码', 'bad'); return; }
    setBusy(true);
    try {
      const added = await api.ImportProxyList(g.build() as any, label);
      toast(`已生成 ${added.length} 个 ${P.name} 代理${check ? '，正在检测出口…' : ''}`);
      onDone(added.map(p => p.id), check); close();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); setBusy(false); }
  };

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal gw-modal" style={{ width: 'min(720px, 94vw)' }}>
        <div className="dhead"><div className="glyph"><Icon.lock s={18} /></div><div><h3>代理商账号</h3><div className="sub">只填账户名和密码，其余从列表选择；每个代理各自一个会话，各自一个出口 IP</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          {g.fields}
          <Toggle on={check} onChange={setCheck} label="生成后立即检测出口" />
          <Sample s={g.sample} />
        </div>
        <div className="dfoot">
          <span className="sub" style={{ fontSize: 11.5 }}>标签：{label}</span>
          <div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" onClick={go} disabled={busy || !g.valid}>{busy ? '生成中…' : `生成 ${n} 个代理`}</button>
        </div>
      </div>
    </div>
  );
};

// 设置 → 代理商账号: save each provider's sub-account once, plus default choices.
export const GatewaySettingsCard: React.FC<{ s: any; set: (p: any) => void; index: number }> = ({ s, set, index }) => {
  const toast = useToast();
  const [prov, setProv] = useState<ProvID>('711');
  const P = PROVIDERS[prov];
  const all: Record<string, GwAcct> = s.gateways || {};
  const a: GwAcct = all[prov] || { user: '', pass: '' };
  const put = (patch: Partial<GwAcct>) => set({ gateways: { ...all, [prov]: { ...a, ...patch } } });
  const clear = () => { const n = { ...all }; delete n[prov]; set({ gateways: n }); };
  const [showPw, setShowPw] = useState(false);
  const [testing, setTesting] = useState(false);
  const cc = a.cc || 'US';
  const countryOpts: [string, string][] = [...(P.random ? [['*', '随机（全球混播）'] as [string, string]] : []), ...COUNTRIES.map(c => [c.cc, `${c.cc} · ${c.name}`] as [string, string])];
  const u = (a.user || '').trim().replace(/-.*$/, '');
  const test = async () => {
    setTesting(true);
    try {
      const host = a.entry || P.entries[0][0];
      const r: any = await api.CheckProxyDraft({ type: a.proto || 'http', host, port: P.port, user: P.build({ user: u, cc: cc === '*' ? '' : cc, st: cc === 'US' ? a.state || '' : '', sess: newSess(P), mins: 5 }), pass: (a.pass || '').trim(), name: host } as any);
      if (r.status === 'ok') toast(`账号可用 · 出口 ${r.ip} · ${r.country} ${r.city || ''} · ${r.latency} ms`, 'ok');
      else toast(`连接失败：${r.error || '请检查账户名和密码'}`, 'bad');
    } catch (e: any) { toast(String(e?.message || e), 'bad'); } finally { setTesting(false); }
  };
  const has = (k: ProvID) => !!all[k]?.user;
  return (
    <div className="panel pad setting-card spot rise" style={{ ['--i' as any]: index }} id="gateway-card">
      <div className="head"><span className="ic"><Icon.lock /></span><div><h4>代理商账号</h4><span>保存一次，新建环境时只需选国家；账号随设置加密同步</span></div></div>
      <Field label="服务商"><Segmented className="block" value={prov} onChange={setProv} items={[['711', `711Proxy${has('711') ? ' · 已保存' : ''}`], ['cli', `Cliproxy${has('cli') ? ' · 已保存' : ''}`]]} /></Field>
      <div className="grid2">
        <Field label="账户名" hint={`${P.name} 控制台「子账户」里的账户名`}><input type="text" value={a.user} onChange={e => put({ user: e.target.value })} placeholder="子账户名" spellCheck={false} /></Field>
        <Field label="密码"><div className="gw-pw"><input type={showPw ? 'text' : 'password'} value={a.pass} onChange={e => put({ pass: e.target.value })} placeholder="子账户密码" spellCheck={false} /><button type="button" className="btn sm ghost" onClick={() => setShowPw(x => !x)}>{showPw ? '隐藏' : '显示'}</button></div></Field>
      </div>
      <div className="grid2">
        <Field label="默认国家 / 地区"><SearchSelect value={cc} options={countryOpts} onChange={v => put({ cc: v, state: '' })} placeholder="搜索，如 US / 日本" /></Field>
        <Field label="默认州" hint={cc === 'US' ? '' : '目前仅美国可选州'}><SelectField value={cc === 'US' ? a.state || '' : ''} onChange={v => put({ state: v })} options={[['', '随机'], ...US_STATES.map(x => [x, x] as [string, string])]} disabled={cc !== 'US'} /></Field>
      </div>
      <div className="grid3">
        <Field label="默认 IP 保持时长"><SelectField value={String(a.mins && P.durations.includes(a.mins) ? a.mins : 30)} onChange={v => put({ mins: +v })} options={P.durations.map(d => [String(d), d >= 60 ? `${d / 60} 小时` : `${d} 分钟`] as [string, string])} /></Field>
        <Field label="默认接入节点"><SelectField value={a.entry && P.entries.some(e => e[0] === a.entry) ? a.entry : P.entries[0][0]} onChange={v => put({ entry: v })} options={P.entries.map(([h, l]) => [h, l] as [string, string])} /></Field>
        <Field label="默认协议"><SelectField value={a.proto || 'http'} onChange={v => put({ proto: v })} options={[['http', 'HTTP'], ['socks5', 'SOCKS5']]} /></Field>
      </div>
      <div className="gw-inline-foot">
        <span className="sub">保存设置后，新建环境、批量创建和「代理商账号」窗口会自动用这里的账号和默认选项</span>
        <div className="spacer" />
        {has(prov) && <button type="button" className="btn sm ghost" onClick={clear}>清除</button>}
        <button type="button" className="btn sm" disabled={testing || !u || !(a.pass || '').trim()} onClick={test}>{testing ? '检测中…' : '检测账号'}</button>
      </div>
    </div>
  );
};
