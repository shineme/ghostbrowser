// Graphite UI: the 实时活动 rail next to the profile list. Each card is a live view of something the app is
// doing right now (local API, uploads/downloads, RPA runs, the syncer, proxy health), fed by the same
// events the rest of the UI uses. Nothing here invents data: a card with nothing to show collapses to a pill.
import React, { useEffect, useMemo, useState } from 'react';
import { api, onEvent, Profile, Proxy, Settings } from './api';
import { Icon, Roll } from './ui';
import { useTransfers, fmtBytes, fmtSpeed, pct, Transfer } from './Transfers';
import { typeLabel } from './rpaMeta';

const Chip: React.FC<{ c: string; children: React.ReactNode; s?: number }> = ({ c, children, s = 32 }) =>
  <span className="la-chip" style={{ ['--c' as any]: c, width: s, height: s }}>{children}</span>;

const C = { env: '#A78BFA', proxy: '#2DD4BF', rpa: '#F5A524', syncer: '#F472B6', cloud: '#4F8CFF', ok: '#30D158', bad: '#FF5A52', warn: '#FFB547' };

const Ring: React.FC<{ f: number; c: string; s?: number; w?: number }> = ({ f, c, s = 14, w = 2.4 }) => {
  const r = (s - w) / 2, L = 2 * Math.PI * r;
  return <svg width={s} height={s} viewBox={`0 0 ${s} ${s}`} style={{ transform: 'rotate(-90deg)', flex: 'none' }}>
    <circle cx={s / 2} cy={s / 2} r={r} stroke={c} strokeOpacity=".2" strokeWidth={w} fill="none" />
    <circle cx={s / 2} cy={s / 2} r={r} stroke={c} strokeWidth={w} fill="none" strokeLinecap="round" strokeDasharray={`${L * Math.max(0, Math.min(1, f))} ${L}`} style={{ transition: 'stroke-dasharray .5s cubic-bezier(.16,1,.3,1)' }} />
  </svg>;
};
export { Ring as LaRing };

const Bar: React.FC<{ f: number; c: string; indet?: boolean }> = ({ f, c, indet }) =>
  <div className={`la-bar ${indet ? 'indet' : ''}`}><i style={{ background: c, transform: indet ? undefined : `translateX(${(Math.max(0, Math.min(1, f)) - 1) * 100}%)` }} /></div>;

// ---------- local API
const ApiPill: React.FC<{ settings: Settings | null }> = ({ settings }) => {
  const on = (settings as any)?.apiEnabled !== false;
  return <div className="la la-pill"><span className={`la-dot ${on ? '' : 'off'}`} /><b>本地 API</b><span className="sp" /><span className="num" style={{ color: on ? C.ok : '#6E6E76', fontWeight: 650 }}>{on ? `:${settings?.apiPort ?? '—'}` : '已关闭'}</span></div>;
};

// ---------- uploads / downloads
const PH: Record<string, string> = { scan: '正在扫描', pack: '正在打包', upload: '上传', commit: '正在提交', download: '下载', write: '正在还原', done: '完成', error: '失败' };
const XferCard: React.FC<{ t: Transfer }> = ({ t }) => {
  const indet = !t.final && (t.total <= 0 || t.phase === 'scan' || t.phase === 'pack' || t.phase === 'commit');
  const p = t.phase === 'done' ? 100 : Math.floor(pct(t));
  const c = t.phase === 'error' ? C.bad : t.phase === 'done' ? C.ok : C.cloud;
  return <div className="la">
    <div className="la-h"><Chip c={C.cloud}>{t.dir === 'up' ? <Icon.upload s={16} /> : <Icon.download s={16} />}</Chip>
      <div className="la-t"><b>云同步</b><span className={!t.final ? 'shim' : ''} style={t.final ? { color: c } : undefined}>{t.phase === 'error' ? (t.error || '失败') : t.phase === 'done' ? (t.dir === 'up' ? '已提交' : '已还原') : `${PH[t.phase] || ''} ${t.name}`}</span></div>
      <span className="sp" /><span className="la-big" style={{ color: c }}>{indet ? '…' : <Roll v={p} />}<small>%</small></span></div>
    <Bar f={p / 100} c={`linear-gradient(90deg, ${c}, ${c === C.cloud ? '#8AB4FF' : c})`} indet={indet} />
    <div className="la-f num"><span><b>{fmtBytes(t.done)}</b>{t.total > 0 && ` / ${fmtBytes(t.total)}`}</span><span className="sp" />{!t.final && t.speed > 0 && fmtSpeed(t.speed)}</div>
    {t.saved > 0 && <div className="la-tags"><span className="la-tag" style={{ ['--c' as any]: C.ok }}>省 {fmtBytes(t.saved)}</span></div>}
  </div>;
};
const CloudIdle: React.FC<{ profiles: Profile[] }> = ({ profiles }) => {
  const n = profiles.filter(p => p.storage === 'cloud').length;
  return <div className="la la-pill"><Chip c={C.cloud} s={28}><Icon.cloud s={14} /></Chip><b>云同步</b><span className="sub">空闲</span><span className="sp" /><span className="num" style={{ color: C.cloud, fontWeight: 650 }}>{n}<small>/{profiles.length}</small></span></div>;
};

// ---------- RPA runs
type RunView = { id: string; name: string; steps: number; status: string; targets: Record<string, { name: string; index: number; type: string; status: string }>; ended?: number };
function useRpaRuns(profiles: Profile[]) {
  const [runs, setRuns] = useState<Record<string, RunView>>({});
  const pname = useMemo(() => Object.fromEntries(profiles.map(p => [p.id, p.name])), [profiles]);
  useEffect(() => onEvent('rpa:event', (e: any) => {
    if (!e?.runId) return;
    setRuns(rs => {
      const r: RunView = rs[e.runId] ? { ...rs[e.runId], targets: { ...rs[e.runId].targets } } : { id: e.runId, name: '自动化流程', steps: 0, status: 'running', targets: {} };
      if (e.phase === 'run') { r.status = e.status; if (e.status !== 'running') r.ended = Date.now(); }
      if (e.profileId) {
        const t = r.targets[e.profileId] || { name: pname[e.profileId] || e.profileId, index: -1, type: '', status: 'starting' };
        if (e.phase === 'target') t.status = e.status;
        if (e.phase === 'step') { t.index = Math.max(t.index, e.index ?? 0); t.type = e.type || t.type; if (e.status === 'failed') t.status = 'failed'; }
        r.targets[e.profileId] = t;
      }
      return { ...rs, [e.runId]: r };
    });
    if (e.phase === 'run' && e.status === 'running') (api as any).WorkflowRun?.(e.runId).then((x: any) => x && setRuns(rs => rs[e.runId] ? { ...rs, [e.runId]: { ...rs[e.runId], name: x.workflowName || rs[e.runId].name, steps: x.stepCount || 0 } } : rs)).catch(() => {});
    if (e.phase === 'run' && e.status !== 'running') setTimeout(() => setRuns(rs => { const n = { ...rs }; delete n[e.runId]; return n; }), 8000);
  }), [pname]);
  return Object.values(runs);
}
const RpaCard: React.FC<{ r: RunView }> = ({ r }) => {
  const ts = Object.values(r.targets);
  const t = ts.find(x => x.status === 'running') || ts[ts.length - 1];
  const done = t ? Math.max(0, t.index + (r.status === 'running' ? 0 : 1)) : 0;
  const c = r.status === 'failed' ? C.bad : r.status === 'running' ? C.rpa : C.ok;
  return <div className="la">
    <div className="la-h"><Chip c={C.rpa}><Icon.bolt s={16} /></Chip>
      <div className="la-t"><b>{r.name}</b><span>{ts.length > 1 ? `${ts.length} 个环境` : t?.name || '准备中'}</span></div>
      <span className="sp" />{r.steps > 0 && <span className="la-big" style={{ color: c }}><Roll v={Math.min(done, r.steps)} /><small>/{r.steps}</small></span>}</div>
    {t && <div className="la-step"><i className={r.status === 'running' ? 'on' : ''} style={{ ['--c' as any]: c }} /><span className={r.status === 'running' ? 'shim' : ''}>{r.status === 'running' ? (t.type ? typeLabel(t.type) : '启动环境') : r.status === 'ok' ? '全部完成' : r.status === 'failed' ? '运行失败' : r.status === 'cancelled' ? '已取消' : '部分完成'}</span></div>}
    {r.steps > 0 && <Bar f={done / r.steps} c={c} />}
  </div>;
};

// ---------- syncer (group control)
const SyncerCard: React.FC = () => {
  const [st, setSt] = useState<any>(null);
  useEffect(() => { (api as any).SyncerState?.().then(setSt).catch(() => {}); return onEvent('syncer:event', setSt); }, []);
  if (!st?.running) return null;
  const fs: any[] = st.followers || [];
  return <div className="la la-pill tall"><Chip c={C.syncer} s={34}><Icon.layers s={16} /></Chip>
    <div className="la-t"><b>同步器</b><span>{st.masterName || '主控'} → {fs.length}</span></div><span className="sp" />
    <div className="la-avs">{fs.slice(0, 3).map(f => <span key={f.id} className={f.ok ? '' : 'bad'} title={f.name}>{(f.name || '?').slice(-2)}</span>)}</div>
    <span className="num" style={{ color: C.syncer, fontWeight: 650, fontSize: 15 }}><Roll v={st.events || 0} /></span></div>;
};

// ---------- proxy health: one bar per checked proxy, height = latency
const ProxyCard: React.FC<{ proxies: Proxy[] }> = ({ proxies }) => {
  const real = proxies.filter(p => !(p as any).gone);
  if (!real.length) return null;
  const ok = real.filter(p => p.status === 'ok'), fail = real.filter(p => p.status === 'fail');
  const lat = ok.map(p => p.latency || p.connectMs || 0).filter(Boolean);
  const avg = lat.length ? Math.round(lat.reduce((a, b) => a + b, 0) / lat.length) : 0;
  const bars = real.slice(0, 44); const max = Math.max(400, ...bars.map(p => p.latency || 0));
  return <div className="la">
    <div className="la-h"><Chip c={C.proxy}><Icon.globe s={16} /></Chip>
      <div className="la-t"><b>代理健康</b><span>{real.length - ok.length - fail.length > 0 ? `${real.length - ok.length - fail.length} 个未检测` : '全部已检测'}</span></div>
      <span className="sp" /><span className="la-big" style={{ color: C.proxy }}><Roll v={ok.length} /><small>/{real.length}</small></span></div>
    <div className="la-hist">{bars.map(p => { const v = p.status === 'ok' ? Math.max(.12, Math.min(1, (p.latency || 0) / max)) : p.status === 'fail' ? 1 : .08; return <i key={p.id} title={`${p.name} · ${p.status === 'ok' ? `${p.latency} ms` : p.status === 'fail' ? '失效' : '未检测'}`} style={{ height: `${v * 100}%`, background: p.status === 'fail' ? C.bad : p.status === 'ok' ? ((p.latency || 0) < 150 ? C.proxy : (p.latency || 0) < 350 ? C.warn : '#FF8A5B') : '#2A2A30' }} />; })}</div>
    <div className="la-f"><span>平均 <b className="num">{avg || '—'}</b> ms</span><span className="sp" />{fail.length > 0 && <span style={{ color: C.bad }}>{fail.length} 个失效</span>}</div>
  </div>;
};

export const LiveRail: React.FC<{ profiles: Profile[]; proxies: Proxy[]; settings: Settings | null }> = ({ profiles, proxies, settings }) => {
  const xs = Object.values(useTransfers());
  const runs = useRpaRuns(profiles);
  const n = 1 + Math.max(1, xs.length) + runs.length + (proxies.length ? 1 : 0);
  return <aside className="live-rail">
    <div className="lr-head"><span>实时活动</span><span className="sp" /><span className="sub num">{n} 项</span></div>
    <div className="lr-body">
      <ApiPill settings={settings} />
      {xs.length ? xs.map(t => <XferCard key={t.id} t={t} />) : <CloudIdle profiles={profiles} />}
      {runs.map(r => <RpaCard key={r.id} r={r} />)}
      <SyncerCard />
      <ProxyCard proxies={proxies} />
    </div>
  </aside>;
};
