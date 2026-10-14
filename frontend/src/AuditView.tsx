import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, Profile, Settings } from './api';
import { Icon, Pick, Segmented, Avatar, useToast, useConfirm } from './ui';

type McpStatus = Awaited<ReturnType<typeof api.McpStatus>>;
type AuditEntry = { id: string; time: number; source: string; action: string; target: string; name: string; ok: boolean; error?: string; detail?: string };
type AuditResult = { items: AuditEntry[]; total: number; all: number; failed: number; categories: Record<string, number>; sources: Record<string, number>; targets: { id: string; name: string; count: number }[] };

// ---------------------------------------------------------------- labels

const CATS: Record<string, string> = { profile: '环境', proxy: '代理', group: '分组', sync: '同步', mcp: 'MCP', api: '接口' };
const SOURCES: [string, string, string][] = [['ui', '界面', 'local'], ['api', '接口', 'cloud'], ['mcp', 'MCP', 'mcp'], ['sync', '同步', 'cy']];
const SRC = Object.fromEntries(SOURCES.map(s => [s[0], s]));
const ACTIONS: Record<string, string> = {
  'profile.start': '启动环境', 'profile.stop': '停止环境', 'profile.create': '新建环境', 'profile.edit': '编辑环境',
  'profile.delete': '删除环境', 'profile.clone': '克隆环境', 'proxy.assign': '分配代理', 'group.set': '设置分组',
  'group.create': '新建分组', 'sync.push': '同步 · 上传', 'sync.pull': '同步 · 下载', 'sync.delete': '同步 · 删除云端',
  'sync.restore': '同步 · 恢复版本', 'mcp.call': 'MCP 调用', 'mcp.start': '启动 MCP 服务', 'mcp.stop': '停止 MCP 服务', 'api.call': '接口调用',
};
const actionLabel = (a: string) => ACTIONS[a] || (a.startsWith('sync.') ? `同步 · ${a.slice(5)}` : a);
const TOOLS: Record<string, string> = {
  list_environments: '列出环境', create_environment: '新建环境', start_environment: '启动环境', stop_environment: '停止环境',
  list_proxies: '列出代理', assign_proxy: '分配代理', list_groups: '列出分组', create_group: '新建分组',
  set_environment_group: '移动到分组', list_tags: '列出标签', list_kernels: '列出内核',
};
const CAT_IC: Record<string, keyof typeof Icon> = { profile: 'layers', proxy: 'globe', group: 'folder', sync: 'cloud', mcp: 'sparkles', api: 'terminal' };

const pad = (n: number) => String(n).padStart(2, '0');
const exact = (ms: number) => { const d = new Date(ms); return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`; };
const clock = (ms: number) => { const d = new Date(ms); const today = new Date().toDateString() === d.toDateString(); return today ? `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}` : `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`; };
const rel = (ms: number) => {
  if (!ms) return '—';
  const s = (Date.now() - ms) / 1000;
  if (s < 10) return '刚刚';
  if (s < 60) return `${Math.floor(s)} 秒前`;
  if (s < 3600) return `${Math.floor(s / 60)} 分钟前`;
  if (s < 86400) return `${Math.floor(s / 3600)} 小时前`;
  return `${Math.floor(s / 86400)} 天前`;
};
const uptime = (ms: number) => {
  if (!ms) return '—';
  const m = Math.floor((Date.now() - ms) / 60000);
  if (m < 1) return '不到 1 分钟';
  if (m < 60) return `${m} 分钟`;
  const h = Math.floor(m / 60);
  return h < 24 ? `${h} 小时 ${m % 60} 分` : `${Math.floor(h / 24)} 天 ${h % 24} 小时`;
};
const mask = (k: string) => !k ? '' : k.length <= 12 ? '•'.repeat(k.length) : `${k.slice(0, 6)}${'•'.repeat(10)}${k.slice(-4)}`;

const Eye: React.FC<{ off?: boolean }> = ({ off }) => <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
  {off ? <><path d="M9.9 4.2A10.4 10.4 0 0 1 12 4c6.5 0 10 8 10 8a17.6 17.6 0 0 1-2.2 3.2M6.6 6.6C3.7 8.5 2 12 2 12s3.5 8 10 8a9.7 9.7 0 0 0 5.4-1.6" /><path d="M9.9 9.9a3 3 0 0 0 4.2 4.2M2 2l20 20" /></> : <><path d="M2 12s3.5-8 10-8 10 8 10 8-3.5 8-10 8-10-8-10-8Z" /><circle cx="12" cy="12" r="3" /></>}
</svg>;

/** Polls fn every ms while the window is visible (and once right away). */
const usePoll = (fn: () => void, ms: number) => {
  const ref = useRef(fn); ref.current = fn;
  useEffect(() => {
    ref.current();
    const t = setInterval(() => { if (document.visibilityState === 'visible') ref.current(); }, ms);
    const v = () => { if (document.visibilityState === 'visible') ref.current(); };
    document.addEventListener('visibilitychange', v);
    return () => { clearInterval(t); document.removeEventListener('visibilitychange', v); };
  }, [ms]);
};

const copyText = (t: string) => { try { navigator.clipboard?.writeText(t); } catch { /* ignore */ } };

// ---------------------------------------------------------------- view

export function AuditView({ profiles }: { profiles: Profile[]; reload?: () => void }) {
  const toast = useToast();
  const [st, setSt] = useState<McpStatus | null>(null);
  const [settings, setSettings] = useState<Settings | null>(null);
  const [reveal, setReveal] = useState(false);

  const loadStatus = useCallback(() => { api.McpStatus().then(setSt).catch(() => {}); }, []);
  useEffect(() => { api.GetSettings().then(setSettings).catch(() => {}); }, []);
  usePoll(loadStatus, 5000);

  const key = settings?.apiKey || '';
  const hideKey = (s: string) => (reveal || !key ? s : s.split(key).join(mask(key)));

  return (
    <div className="audit">
      <div className="au-grid">
        <McpCard st={st} onChange={setSt} />
        <ApiCard settings={settings} keyRequired={st ? st.keyRequired : !settings?.apiKeyOff} reveal={reveal} setReveal={setReveal} />
        <ConfigCard st={st} hideKey={hideKey} onCopy={() => toast('已复制到剪贴板')} />
        <ToolsCard st={st} />
      </div>
      <AuditLog profiles={profiles} />
    </div>
  );
}

// ---------------------------------------------------------------- MCP server card

const McpCard: React.FC<{ st: McpStatus | null; onChange: (s: McpStatus) => void }> = ({ st, onChange }) => {
  const toast = useToast();
  const [port, setPort] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState('');
  useEffect(() => { if (st && port === '') setPort(String(st.port)); }, [st]);
  const portN = Math.round(+port || 0);
  const portBad = port !== '' && (portN < 1 || portN > 65535);
  const changed = !!st && portN > 0 && portN !== st.port;
  const errorText = err || st?.error || '';
  const [recent, setRecent] = useState<AuditEntry[] | null>(null);
  usePoll(() => { api.AuditLog({ category: 'mcp', action: 'mcp.call', source: '', target: '', query: '', onlyFail: false, since: 0, limit: 3 } as any).then(r => setRecent((r as any).items || [])).catch(() => setRecent([])); }, 5000);

  const start = async () => {
    if (portBad) return;
    setBusy(true); setErr('');
    try { const s = await api.StartMcp(changed ? portN : 0); onChange(s); setPort(String(s.port)); toast(changed && st?.running ? `MCP 服务已在端口 ${s.port} 重启` : `MCP 服务已启动 · 端口 ${s.port}`); }
    catch (e: any) { setErr(String(e?.message || e)); api.McpStatus().then(onChange).catch(() => {}); toast('MCP 服务启动失败', 'bad'); }
    finally { setBusy(false); }
  };
  const stop = async () => {
    setBusy(true); setErr('');
    try { onChange(await api.StopMcp()); toast('MCP 服务已停止'); } finally { setBusy(false); }
  };

  const running = !!st?.running;
  const state = !st ? 'loading' : running ? 'on' : errorText ? 'err' : 'off';
  return (
    <div className={`panel pad au-card spot rise au-mcp ${state}`} style={{ ['--i' as any]: 0 }}>
      <div className="au-head">
        <span className="ic"><Icon.sparkles /></span>
        <div className="grow"><h4>MCP 服务</h4><span>让 Cursor、Claude 等 AI 客户端直接管理环境{st?.protocol ? ` · 协议 ${st.protocol}` : ''}</span></div>
        {state === 'loading' ? <ThinkingOrb state="searching" size={20} theme="dark" />
          : state === 'on' ? <span className="badge ok"><span className="live sm" />运行中</span>
          : state === 'err' ? <span className="badge fail"><i className="d" />启动失败</span>
          : <span className="badge unknown"><i className="d" />已停止</span>}
      </div>

      <div className="au-url">
        <span className="k">地址</span>
        <code className={`mono ${running ? '' : 'dim'}`} title={st?.url}>{st?.url || '—'}</code>
        <button className="btn ghost icon sm" title="复制地址" disabled={!st} onClick={() => { copyText(st?.url || ''); toast('已复制 MCP 地址'); }}><Icon.copy s={14} /></button>
      </div>

      <div className="au-ctl">
        <label className={`au-port ${portBad ? 'bad' : ''}`}><span>端口</span>
          <input type="number" min={1} max={65535} value={port} onChange={e => setPort(e.target.value.replace(/\D/g, '').slice(0, 5))} onKeyDown={e => e.key === 'Enter' && start()} />
        </label>
        {running
          ? <>
            {changed && <button className="btn primary" disabled={busy || portBad} onClick={start}><Icon.refresh s={14} /> 换端口重启</button>}
            <button className="btn stop" disabled={busy} onClick={stop}>{busy ? <ThinkingOrb state="searching" size={20} theme="dark" /> : <Icon.stop s={13} />} 停止</button>
          </>
          : <button className="btn primary" disabled={busy || portBad || !st} onClick={start}>{busy ? <ThinkingOrb state="searching" size={20} theme="dark" /> : <Icon.play s={13} />} {errorText ? '重试启动' : '启动'}</button>}
        {portBad && <span className="au-hint bad">端口范围 1–65535</span>}
      </div>

      {errorText && !running && <div className="notice bad au-err">
        <Icon.alert /><div className="grow"><b>端口 {st?.port} 无法监听</b><div className="sub" title={errorText}>{errorText}</div><div className="sub">通常是端口被其他程序占用，换一个端口（如 {(st?.port || 25326) + 1}）再启动。</div></div>
      </div>}

      {!(errorText && !running) && <div className="au-recent">
        <div className="au-recent-h">最近调用</div>
        {recent === null ? <div className="au-recent-e"><ThinkingOrb state="searching" size={20} theme="dark" /></div>
          : recent.length === 0 ? <div className="au-recent-e">还没有 AI 客户端调用过，复制下方配置即可接入</div>
          : recent.map(e => { const rest = (e.detail || '').startsWith(e.name) ? (e.detail || '').slice(e.name.length).trim() : e.detail; return (
            <div key={e.id} className={`au-recent-r ${e.ok ? '' : 'bad'}`} title={[e.error, e.detail].filter(Boolean).join(' · ')}>
              <i className="d" /><code className="mono">{e.name}</code><span className="a mono">{e.error || rest || ''}</span><span className="t" title={exact(e.time)}>{rel(e.time)}</span>
            </div>); })}
      </div>}

      <div className="au-metrics">
        <div><span>调用次数</span><b className="num">{running ? (st?.calls ?? 0).toLocaleString() : '—'}</b></div>
        <div><span>最近客户端</span><b title={st?.lastClient}>{running && st?.lastClient ? st.lastClient : '—'}</b></div>
        <div><span>最近调用</span><b title={st?.lastCall ? exact(st.lastCall) : ''}>{running && st?.lastCall ? rel(st.lastCall) : '—'}</b></div>
        <div><span>已运行</span><b>{running ? uptime(st?.startedAt || 0) : '—'}</b></div>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- local API card

const ApiCard: React.FC<{ settings: Settings | null; keyRequired: boolean; reveal: boolean; setReveal: (v: boolean) => void }> = ({ settings, keyRequired, reveal, setReveal }) => {
  const toast = useToast();
  const on = settings ? settings.apiEnabled !== false : true;
  const addr = `127.0.0.1:${settings?.apiPort || 25325}`;
  const key = settings?.apiKey || '';
  const goSettings = () => (document.querySelector('.nav button[data-t="settings"]') as HTMLButtonElement | null)?.click();
  return (
    <div className="panel pad au-card spot rise" style={{ ['--i' as any]: 1 }}>
      <div className="au-head">
        <span className="ic"><Icon.terminal /></span>
        <div className="grow"><h4>本地 API</h4><span>Puppeteer / Playwright / Selenium 接管</span></div>
        {!settings ? <ThinkingOrb state="searching" size={20} theme="dark" /> : on ? <span className="badge ok"><span className="live sm" />运行中</span> : <span className="badge unknown"><i className="d" />已关闭</span>}
      </div>
      <div className="au-url">
        <span className="k">地址</span>
        <code className={`mono ${on ? '' : 'dim'}`}>http://{addr}</code>
        <button className="btn ghost icon sm" title="复制地址" onClick={() => { copyText(`http://${addr}`); toast('已复制本地 API 地址'); }}><Icon.copy s={14} /></button>
      </div>
      <div className="au-url">
        <span className="k">Key</span>
        <code className="mono" title={reveal ? key : '点击右侧眼睛显示'}>{key ? (reveal ? key : mask(key)) : '未生成'}</code>
        <button className="btn ghost icon sm" title={reveal ? '隐藏' : '显示'} disabled={!key} onClick={() => setReveal(!reveal)}><Eye off={reveal} /></button>
        <button className="btn ghost icon sm" title="复制 Key" disabled={!key} onClick={() => { copyText(key); toast('已复制 API Key'); }}><Icon.copy s={14} /></button>
      </div>
      <div className="au-code cli au-sample">
        <pre className="mono"><span className="pr">$ </span>curl http://{addr}/list{keyRequired ? <> -H "X-API-Key: {key ? (reveal ? key : mask(key)) : '…'}"</> : null}</pre>
        <button className="btn sm au-copy" onClick={() => { copyText(`curl http://${addr}/list` + (keyRequired ? ` -H "X-API-Key: ${key}"` : '')); toast('已复制示例命令'); }}><Icon.copy s={13} /> 复制</button>
      </div>
      <div className="au-hint">返回全部环境；<code className="mono">/profile/start?id=</code> 启动后得到 CDP 地址供自动化框架接管。</div>
      <div className="au-foot">
        {keyRequired ? <span className="badge ok"><Icon.lock />需要 API Key</span> : <span className="badge warn"><Icon.alert />未要求 Key</span>}
        <span className="sub grow">{keyRequired ? '与 MCP 共用' : '本机程序可直接调用'}</span>
        <button className="btn sm ghost" onClick={goSettings}><Icon.settings s={13} /> 在设置中管理</button>
      </div>
    </div>
  );
};

// ---------------------------------------------------------------- client config

type Mode = 'http' | 'stdio' | 'cli';
const MODE_HINT: Record<Mode, React.ReactNode> = {
  http: <>写入 <code className="mono">~/.cursor/mcp.json</code>，Windsurf、Cline 也是同样格式。需要 MCP 服务保持运行。</>,
  stdio: <>写入 <code className="mono">%APPDATA%\Claude\claude_desktop_config.json</code> 后重启 Claude Desktop。它会自行以 stdio 模式拉起 GhostBrowser，无需开启上方服务。</>,
  cli: <>在终端运行这条命令，Claude Code 会把 GhostBrowser 加入当前项目的 MCP 列表。需要 MCP 服务保持运行。</>,
};

const ConfigCard: React.FC<{ st: McpStatus | null; hideKey: (s: string) => string; onCopy: () => void }> = ({ st, hideKey, onCopy }) => {
  const [mode, setMode] = useState<Mode>('http');
  const [copied, setCopied] = useState(false);
  const raw = !st ? '' : mode === 'http' ? st.configHttp : mode === 'stdio' ? st.configStdio : st.claudeCodeCl;
  const copy = () => { copyText(raw); setCopied(true); onCopy(); setTimeout(() => setCopied(false), 1600); };
  const needsServer = mode !== 'stdio' && st && !st.running;
  return (
    <div className="panel pad au-card spot rise au-config" style={{ ['--i' as any]: 2 }}>
      <div className="au-head">
        <span className="ic"><Icon.copy /></span>
        <div className="grow"><h4>客户端配置</h4><span>复制到 AI 客户端即可连接</span></div>
      </div>
      <Segmented className="block" value={mode} onChange={setMode}
        items={[['http', 'Cursor · HTTP'], ['stdio', 'Claude Desktop · stdio'], ['cli', 'Claude Code · CLI']]} />
      <div className={`au-code ${mode === 'cli' ? 'cli' : ''}`}>
        {st ? <pre className="mono">{mode === 'cli' && <span className="pr">$ </span>}{hideKey(raw)}</pre> : <div className="au-code-load"><ThinkingOrb state="searching" size={20} theme="dark" /></div>}
        <button className={`btn sm au-copy ${copied ? 'done' : ''}`} disabled={!st} onClick={copy}>{copied ? <><Icon.check s={13} /> 已复制</> : <><Icon.copy s={13} /> 复制</>}</button>
      </div>
      <div className="au-hint">{MODE_HINT[mode]}{st?.keyRequired && mode !== 'stdio' ? '复制的内容包含完整 API Key。' : ''}</div>
      {needsServer && <div className="notice warn au-warn"><Icon.alert /><span className="grow">MCP 服务未启动，这种接入方式暂时连不上。</span></div>}
    </div>
  );
};

// ---------------------------------------------------------------- tools

const ToolsCard: React.FC<{ st: McpStatus | null }> = ({ st }) => {
  const tools = st?.tools || [];
  const read = (t: string) => t.startsWith('list_');
  return (
    <div className="panel pad au-card spot rise au-tools" style={{ ['--i' as any]: 3 }}>
      <div className="au-head">
        <span className="ic"><Icon.bolt /></span>
        <div className="grow"><h4>可用工具</h4><span>{st ? `${tools.length} 个工具 · ${tools.filter(read).length} 个只读` : '读取中…'}</span></div>
      </div>
      {!st ? <div className="au-code-load"><ThinkingOrb state="searching" size={20} theme="dark" /></div>
        : tools.length === 0 ? <div className="sub">没有可用工具</div>
        : <div className="au-chips">{tools.map((t: string) => <span key={t} className={`au-tool ${read(t) ? 'ro' : 'rw'}`} title={`${TOOLS[t] || t}${read(t) ? '（只读）' : '（会修改数据，记入操作日志）'}`}>
          <b>{TOOLS[t] || t}</b><span className="mono">{t}</span>
        </span>)}</div>}
      <div className="au-legend"><span><i className="ro" />只读</span><span><i className="rw" />会修改数据</span></div>
    </div>
  );
};

// ---------------------------------------------------------------- audit log

const PAGE = 200;

const AuditLog: React.FC<{ profiles: Profile[] }> = ({ profiles }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [res, setRes] = useState<AuditResult | null>(null);
  const [cat, setCat] = useState('');
  const [src, setSrc] = useState('');
  const [target, setTarget] = useState('');
  const [q, setQ] = useState('');
  const [dq, setDq] = useState('');
  const [onlyFail, setOnlyFail] = useState(false);
  const [limit, setLimit] = useState(PAGE);
  const [loading, setLoading] = useState(true);
  const [, tick] = useState(0);

  useEffect(() => { const t = setTimeout(() => setDq(q), 220); return () => clearTimeout(t); }, [q]);
  const filter = useMemo(() => ({ category: cat, action: '', source: src, target, query: dq, onlyFail, since: 0, limit }), [cat, src, target, dq, onlyFail, limit]);
  const load = useCallback(() => {
    api.AuditLog(filter as any).then(r => { setRes(r as any); setLoading(false); tick(x => x + 1); }).catch(() => setLoading(false));
  }, [filter]);
  useEffect(() => { setLoading(true); load(); }, [load]);
  usePoll(load, 5000);

  const byId = useMemo(() => Object.fromEntries(profiles.map(p => [p.id, p])), [profiles]);
  const [pxName, setPxName] = useState<Record<string, string>>({});
  useEffect(() => { api.ListProxies().then(l => setPxName(Object.fromEntries((l || []).map(x => [x.id, x.name])))).catch(() => {}); }, []);
  const filtered = !!(cat || src || target || dq || onlyFail);
  const reset = () => { setCat(''); setSrc(''); setTarget(''); setQ(''); setDq(''); setOnlyFail(false); };

  const exportAs = async (fmt: 'csv' | 'json') => {
    try { const p = await api.ExportAuditLog({ ...filter, limit: 0 } as any, fmt); if (p) toast(`已导出 ${res?.total ?? ''} 条记录到 ${p}`); }
    catch (e: any) { toast(String(e?.message || e), 'bad'); }
  };
  const clear = async () => {
    if (!await ask({ title: '清空操作日志？', body: `将删除全部 ${res?.all ?? 0} 条记录（包括界面、本地 API、MCP 和同步的操作），无法恢复。需要留档的话先导出。`, ok: '清空', danger: true })) return;
    await api.ClearAuditLog(); toast('操作日志已清空'); load();
  };

  const cats = res ? Object.entries(res.categories).sort((a, b) => b[1] - a[1]) : [];
  const all = res?.all ?? 0;

  return (
    <div className="au-log">
      <div className="section-title">操作日志
        {res && all > 0 && <span className="au-sum num">{all} 条 · <em className={res.failed ? 'bad' : ''}>{res.failed} 条失败</em></span>}
      </div>

      {res && all === 0 && !loading ? (
        <div className="panel empty au-empty">
          <div className="halo"><ThinkingOrb state="breathing" size={64} theme="dark" /></div>
          <h3>还没有操作记录</h3>
          <div>启动、新建、删除环境，分配代理，云端同步，以及本地 API 和 MCP 的每一次调用都会记在这里。<br />可以按来源、环境筛选，并导出为 CSV / JSON 留档。</div>
        </div>
      ) : <>
        <div className="toolbar au-toolbar">
          <div className="search"><Icon.search /><input type="text" placeholder="搜索日志…" title="按环境名、工具、错误信息或详情搜索" value={q} onChange={e => setQ(e.target.value)} /></div>
          <Pick label="分类" value={cat} onChange={setCat} icon={<Icon.layers />}
            options={[['', '全部分类', all], ...cats.map(([c, n]) => [c, CATS[c] || c, n] as [string, string, number])]} />
          <Pick label="来源" value={src} onChange={setSrc} icon={<Icon.server />}
            options={[['', '全部来源', all], ...SOURCES.filter(s => res?.sources[s[0]]).map(s => [s[0], s[1], res!.sources[s[0]]] as [string, string, number])]} />
          <Pick label="环境" value={target} onChange={setTarget} icon={<Icon.grid />}
            options={[['', '全部环境'] as [string, string, number?], ...(res?.targets || []).slice(0, 60).map(t => [t.id, byId[t.id]?.name || t.name || t.id.slice(0, 8), t.count] as [string, string, number])]} />
          <button className={`au-failtg ${onlyFail ? 'on' : ''}`} onClick={() => setOnlyFail(!onlyFail)} title="只看失败的操作">
            <span className={`sw ${onlyFail ? 'on' : ''}`} />仅失败{res?.failed ? <span className="num">{res.failed}</span> : null}
          </button>
          {filtered && <button className="tb-clear" onClick={reset}>清除筛选</button>}
          <div className="spacer" />
          <span className="au-live" title="每 5 秒自动刷新"><span className="live sm" />自动刷新</span>
          <ExportMenu disabled={!res?.total} onPick={exportAs} />
          <button className="btn danger" disabled={!all} onClick={clear}><Icon.trash s={14} /> 清空</button>
        </div>

        <div className="panel list au-table">
          <table>
            <thead><tr>
              <th style={{ width: 118 }}>时间</th><th style={{ width: 84 }}>来源</th><th style={{ width: 150 }}>操作</th><th style={{ width: 210 }}>对象</th><th style={{ width: 86 }}>结果</th><th>详情</th>
            </tr></thead>
            <tbody>
              {(res?.items || []).map((e, i) => <Row key={e.id} e={e} i={i} p={byId[e.target]} px={pxName} />)}
            </tbody>
          </table>
          {loading && !res && <div className="au-loading"><ThinkingOrb state="searching" size={32} theme="dark" /><span>读取操作日志…</span></div>}
          {res && res.items.length === 0 && <div className="au-none">
            <Icon.search s={18} /><span>没有符合筛选条件的记录</span>{filtered && <button className="btn sm" onClick={reset}>清除筛选</button>}
          </div>}
          {res && res.total > res.items.length && <div className="au-more">
            <span className="sub num">显示 {res.items.length} / {res.total} 条</span>
            <button className="btn sm" onClick={() => setLimit(l => l + PAGE)}>再加载 {Math.min(PAGE, res.total - res.items.length)} 条</button>
          </div>}
        </div>
      </>}
    </div>
  );
};

const Row: React.FC<{ e: AuditEntry; i: number; p?: Profile; px: Record<string, string> }> = ({ e, i, p, px }) => {
  const cat = e.action.split('.')[0];
  const I = Icon[CAT_IC[cat] || 'bolt'];
  const s = SRC[e.source] || [e.source, e.source, 'local'];
  let det = e.action === 'mcp.call' && e.detail && e.name && e.detail.startsWith(e.name) ? e.detail.slice(e.name.length).trim() : e.detail;
  if (e.action === 'proxy.assign') det = e.detail ? `→ ${px[e.detail] || e.detail}` : '→ 直连';
  const detail = [e.error, e.detail].filter(Boolean).join(' · ');
  let obj: React.ReactNode;
  if (e.action === 'mcp.call') obj = <span className="au-obj"><code className="mono au-toolname" title={TOOLS[e.name] || ''}>{e.name || '—'}</code></span>;
  else if (e.target) obj = <span className="au-obj">{p ? <Avatar seed={p.fingerprint?.noiseSeed ?? p.id} os={p.fingerprint?.os || 'windows'} /> : <span className="au-gone"><Icon.layers s={13} /></span>}
    <span className="t"><span className="nm" title={p?.name || e.name}>{p?.name || e.name || e.target.slice(0, 8)}</span>{!p && <span className="sub">已删除</span>}</span></span>;
  else if (cat === 'group') obj = <span className="au-obj"><span className="au-gone"><Icon.folder s={13} /></span><span className="t"><span className="nm" title={e.name}>{e.name || '—'}</span></span></span>;
  else if (cat === 'mcp') obj = <span className="au-obj"><span className="au-gone"><Icon.sparkles s={13} /></span><span className="t"><span className="nm">MCP 服务</span></span></span>;
  else obj = <span className="sub">{e.name || '—'}</span>;
  return (
    <tr className={`rise ${e.ok ? '' : 'au-fail'}`} style={{ ['--i' as any]: Math.min(i, 16) }}>
      <td title={exact(e.time)}><div className="au-rel">{rel(e.time)}</div><div className="sub num">{clock(e.time)}</div></td>
      <td><span className={`badge ${s[2]}`}>{s[1]}</span></td>
      <td><span className={`au-act c-${cat}`}><span className="ai"><I s={13} /></span>{actionLabel(e.action)}</span></td>
      <td>{obj}</td>
      <td>{e.ok ? <span className="badge ok"><i className="d" />成功</span> : <span className="badge fail" title={e.error}><i className="d" />失败</span>}</td>
      <td className="au-detail" title={detail}>{e.error && <span className="err">{e.error}</span>}{e.error && det && <span className="sep"> · </span>}{det ? <span className="mono">{det}</span> : !e.error && <span className="sub">—</span>}</td>
    </tr>
  );
};

const ExportMenu: React.FC<{ disabled?: boolean; onPick: (f: 'csv' | 'json') => void }> = ({ disabled, onPick }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setOpen(false); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key); };
  }, [open]);
  return (
    <div className={`pick au-export ${open ? 'open' : ''}`} ref={ref}>
      <button className="btn" disabled={disabled} onClick={() => setOpen(o => !o)}><Icon.download s={14} /> 导出
        <svg className="pick-c" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg></button>
      {open && <div className="pick-menu right">
        {([['csv', 'CSV 表格', 'Excel / WPS 可直接打开'], ['json', 'JSON', '便于程序处理']] as const).map(([v, l, h]) =>
          <button key={v} type="button" onClick={() => { setOpen(false); onPick(v); }}>
            <span className="pick-tick" /><span className="pick-o">{l}<small>{h}</small></span><span className="pick-n num">.{v}</span>
          </button>)}
        <div className="au-export-note">按当前筛选导出全部匹配记录</div>
      </div>}
    </div>
  );
};
