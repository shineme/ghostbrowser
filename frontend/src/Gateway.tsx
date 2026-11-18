// 代理商账号：fill in the provider's sub-account name and password once, pick
// everything else from lists, and get ready-made gateway proxies. 711Proxy and
// Cliproxy put country / state / sticky session in the username; GhostBrowser
// builds it, and "换 IP" later just rolls the session value (gateway.go).
import React, { useEffect, useMemo, useState } from 'react';
import { api, Proxy } from './api';
import { Icon, Field, Toggle, useToast, useExit, Segmented, SelectField } from './ui';
import { COUNTRIES, SearchSelect } from './locales';

type ProvID = '711' | 'cli';
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

export const GatewayModal: React.FC<{ proxies: Proxy[]; onClose: () => void; onDone: (ids: string[], check: boolean) => void }> = ({ proxies, onClose, onDone }) => {
  const toast = useToast();
  const [out, close] = useExit(onClose);
  const [prov, setProv] = useState<ProvID>('711');
  const P = PROVIDERS[prov];
  // the last account used with this provider fills itself in
  const known = useMemo(() => {
    const m: Partial<Record<ProvID, { user: string; pass: string }>> = {};
    for (const p of [...proxies].reverse()) { const k = provOf(p); if (k && !m[k] && p.user) m[k] = { user: PROVIDERS[k].base(p.user), pass: p.pass || '' }; }
    return m;
  }, [proxies]);
  const [user, setUser] = useState('');
  const [pass, setPass] = useState('');
  const [entry, setEntry] = useState(P.entries[0][0]);
  const [proto, setProto] = useState<'http' | 'socks5'>('http');
  const [cc, setCc] = useState('US');
  const [st, setSt] = useState('');
  const [sticky, setSticky] = useState<'sticky' | 'rotate'>('sticky');
  const [mins, setMins] = useState('30');
  const [count, setCount] = useState('5');
  const [rotate, setRotate] = useState(false);
  const [check, setCheck] = useState(true);
  const [busy, setBusy] = useState(false);
  const [showPw, setShowPw] = useState(false);

  useEffect(() => {
    const k = known[prov];
    if (k) { setUser(k.user); setPass(k.pass); } // otherwise keep what was typed
    setEntry(PROVIDERS[prov].entries[0][0]);
    if (!PROVIDERS[prov].durations.includes(+mins)) setMins('30');
    if (!PROVIDERS[prov].random && !cc) setCc('US');
  }, [prov]); // eslint-disable-line react-hooks/exhaustive-deps

  const countryOpts = useMemo(() => [...(P.random ? [['*', '随机（全球混播）'] as [string, string]] : []), ...COUNTRIES.map(c => [c.cc, `${c.cc} · ${c.name}`] as [string, string])], [P]);
  const stateOpts: [string, string][] = [['', '随机'], ...US_STATES.map(s => [s, s] as [string, string])];
  const u = user.trim().replace(/-.*$/, ''); // a pasted full username keeps only the account part
  const mk = () => P.build({ user: u, cc: cc === '*' ? '' : cc, st: cc === 'US' ? st : '', sess: sticky === 'sticky' ? newSess(P) : '', mins: +mins });
  const sample = useMemo(() => u ? `${entry}:${P.port}:${mk()}:${pass ? '••••••' : '密码'}` : '', [u, pass, entry, cc, st, sticky, mins, prov]); // eslint-disable-line react-hooks/exhaustive-deps
  const n = sticky === 'rotate' ? 1 : +count;
  const label = `${P.name}-${cc === '*' ? '全球' : cc}${cc === 'US' && st ? '-' + stCli(st) : ''}`;

  const go = async () => {
    if (!u || !pass.trim()) { toast('请填写账户名和密码', 'bad'); return; }
    setBusy(true);
    try {
      const list = Array.from({ length: n }, () => ({ type: proto, host: entry, port: P.port, user: mk(), pass: pass.trim(), name: `${entry}:${P.port}`, rotateOnLaunch: rotate && sticky === 'sticky', status: 'unknown' }));
      const added = await api.ImportProxyList(list as any, label);
      toast(`已生成 ${added.length} 个 ${P.name} 代理${check ? '，正在检测出口…' : ''}`);
      onDone(added.map(p => p.id), check); close();
    } catch (e: any) { toast(String(e?.message || e), 'bad'); setBusy(false); }
  };

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="modal gw-modal" style={{ width: 'min(720px, 94vw)' }}>
        <div className="dhead"><div className="glyph"><Icon.lock s={18} /></div><div><h3>代理商账号</h3><div className="sub">只填账户名和密码，其余从列表选择；每个代理各自一个会话，各自一个出口 IP</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <Field label="服务商"><Segmented className="block" value={prov} onChange={setProv} items={[['711', '711Proxy · 动态住宅'], ['cli', 'Cliproxy · 流量套餐']]} /></Field>
          <div className="grid2">
            <Field label="账户名" hint={known[prov] ? '已从上次使用的账号自动填入' : `${P.name} 控制台「子账户」里的账户名`}><input type="text" value={user} onChange={e => setUser(e.target.value)} placeholder="子账户名" autoFocus spellCheck={false} /></Field>
            <Field label="密码"><div className="gw-pw"><input type={showPw ? 'text' : 'password'} value={pass} onChange={e => setPass(e.target.value)} placeholder="子账户密码" spellCheck={false} /><button type="button" className="btn sm ghost" onClick={() => setShowPw(x => !x)}>{showPw ? '隐藏' : '显示'}</button></div></Field>
          </div>
          <div className="grid2">
            <Field label="国家 / 地区"><SearchSelect value={cc} options={countryOpts} onChange={v => { setCc(v); setSt(''); }} placeholder="搜索，如 US / 日本" /></Field>
            <Field label="州" hint={cc === 'US' ? '' : '目前仅美国可选州'}><SelectField value={cc === 'US' ? st : ''} onChange={setSt} options={stateOpts} disabled={cc !== 'US'} /></Field>
          </div>
          <div className="grid3">
            <Field label="会话类型"><SelectField value={sticky} onChange={setSticky} options={[['sticky', '粘性 IP（推荐）'], ['rotate', '轮换 IP（每个请求换）']]} /></Field>
            <Field label="IP 保持时长"><SelectField value={mins} onChange={setMins} disabled={sticky === 'rotate'} options={P.durations.map(d => [String(d), d >= 60 ? `${d / 60} 小时` : `${d} 分钟`] as [string, string])} /></Field>
            <Field label="生成数量"><SelectField value={sticky === 'rotate' ? '1' : count} onChange={setCount} disabled={sticky === 'rotate'} options={['1', '5', '10', '20', '50', '100'].map(x => [x, `${x} 个`] as [string, string])} /></Field>
          </div>
          <div className="grid2">
            <Field label="接入节点"><SelectField value={entry} onChange={setEntry} options={P.entries.map(([h, l]) => [h, `${l} · ${h}`] as [string, string])} /></Field>
            <Field label="协议"><SelectField value={proto} onChange={setProto} options={[['http', 'HTTP（浏览器推荐）'], ['socks5', 'SOCKS5']]} /></Field>
          </div>
          {sticky === 'rotate' && <div className="notice warn"><Icon.alert /><span className="grow sub">轮换模式每个请求都换 IP，登录状态会掉，网站也容易触发风控。浏览器环境请用粘性 IP。</span></div>}
          {sticky === 'sticky' && <Toggle on={rotate} onChange={setRotate} label="每次启动环境前自动换 IP" hint="关闭时同一个环境会一直用同一个会话；IP 超过保持时长后由服务商自动更换" />}
          <Toggle on={check} onChange={setCheck} label="生成后立即检测出口" />
          {sample && <div className="gw-sample"><span className="sub">预览</span><code title={sample}>{sample}</code></div>}
        </div>
        <div className="dfoot">
          <span className="sub" style={{ fontSize: 11.5 }}>标签：{label}</span>
          <div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" onClick={go} disabled={busy || !u || !pass.trim()}>{busy ? '生成中…' : `生成 ${n} 个代理`}</button>
        </div>
      </div>
    </div>
  );
};
