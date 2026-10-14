import React, { useEffect, useMemo, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Profile } from './api';
import { Icon, Segmented, Avatar, ago, useToast, useConfirm, useExit } from './ui';
import { RI, RWorkflow, RRun, RTarget, REvent, RStepEvent, StatusBadge, StepIcon, toneOf, typeLabel, flat, dur, clock } from './rpaMeta';

const err = (e: any) => String(e?.message || e);
const live = (s: string) => s === 'running' || s === 'starting' || s === 'pending';

export const RpaRunDrawer: React.FC<{ wf: RWorkflow; runId: string; profiles: Profile[]; onClose: () => void; onRerun: () => void }> = ({ wf, runId, profiles, onClose, onRerun }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [out, close] = useExit(onClose);
  const [tab, setTab] = useState<'run' | 'history'>(runId ? 'run' : 'history');
  const [cur, setCur] = useState(runId);
  const [run, setRun] = useState<RRun | null>(null);
  const [loading, setLoading] = useState(!!runId);
  const [loadErr, setLoadErr] = useState('');
  const [hist, setHist] = useState<RRun[] | null>(null);
  const [now, setNow] = useState(Date.now());
  const curRef = useRef(cur); curRef.current = cur;

  const names = useMemo(() => { const m: Record<string, string> = {}; flat(wf.steps).forEach(s => (m[s.id] = s.name || typeLabel(s.type))); return m; }, [wf]);
  const loadRun = async (id: string) => {
    if (!id) { setRun(null); setLoading(false); return; }
    setLoading(true);
    try { setRun((await api.WorkflowRun(id)) as any); setLoadErr(''); } catch (e) { setLoadErr(err(e)); setRun(null); } finally { setLoading(false); }
  };
  const loadHist = () => api.WorkflowRuns(wf.id).then(r => setHist(((r || []) as any as RRun[]).sort((a, b) => b.startedAt - a.startedAt))).catch(() => setHist([]));
  useEffect(() => { loadRun(cur); }, [cur]);
  useEffect(() => { loadHist(); }, []);
  useEffect(() => { if (!run || run.status !== 'running') return; const t = setInterval(() => setNow(Date.now()), 1000); return () => clearInterval(t); }, [run?.status]);

  // Fold live events into the run being shown.
  useEffect(() => onEvent('rpa:event', (e: REvent) => {
    if (e.runId !== curRef.current) return;
    if (e.phase === 'run' && e.status !== 'running') { loadRun(e.runId); loadHist(); return; }
    setRun(r => {
      if (!r) return r;
      if (e.phase === 'run') return { ...r, status: e.status };
      const targets = r.targets.map(t => {
        if (t.profileId !== e.profileId) return t;
        if (e.phase === 'target') return { ...t, status: e.status, error: e.error || (e.status === 'ok' ? '' : t.error), vars: e.vars || t.vars, startedAt: t.startedAt || Math.floor(Date.now() / 1000), endedAt: live(e.status) ? 0 : Math.floor(Date.now() / 1000), cur: live(e.status) ? t.cur : undefined };
        const ev: RStepEvent = { stepId: e.stepId || '', index: e.index, type: e.type || '', status: e.status, log: e.log || '', ms: e.ms || 0, iter: e.iter, vars: e.vars, shot: e.shot };
        if (e.status === 'running') return { ...t, status: 'running', cur: ev };
        return { ...t, steps: [...t.steps, ev], vars: e.vars || t.vars, cur: undefined };
      });
      return { ...r, targets };
    });
  }), []);

  const cancel = async () => {
    if (!run || !await ask({ title: '停止这次运行？', body: '正在执行的步骤结束后停止，排队中的环境不再执行。', ok: '停止运行', danger: true })) return;
    try { const ok = await api.CancelWorkflowRun(run.id); toast(ok ? '正在停止…' : '这次运行已经结束', ok ? 'ok' : 'bad'); } catch (e) { toast(err(e), 'bad'); }
  };
  const t = run?.targets || [];
  const cnt = (f: (x: RTarget) => boolean) => t.filter(f).length;
  const done = cnt(x => !live(x.status));
  const isRunning = run?.status === 'running';
  const elapsed = run ? ((run.endedAt || (isRunning ? now / 1000 : run.startedAt)) - run.startedAt) * 1000 : 0;

  return (
    <div className={`overlay ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && close()}>
      <div className="drawer rpa-run">
        <div className="dhead">
          <div className="glyph">{isRunning ? <ThinkingOrb state="working" size={20} theme="dark" /> : <RI.history s={20} />}</div>
          <div style={{ minWidth: 0, flex: 1 }}>
            <h3 className="ell">{wf.name}</h3>
            <div className="sub">{run ? <>开始于 {clock(run.startedAt)} · 用时 {dur(Math.max(0, Math.round(elapsed)))} · 并发 {run.concurrency}</> : '运行详情与历史记录'}</div>
          </div>
          <Segmented className="compact" items={[['run', '运行详情'], ['history', `历史记录${hist ? ` ${hist.length}` : ''}`]]} value={tab} onChange={setTab} />
          <button className="btn ghost icon" onClick={() => close()}><Icon.x /></button>
        </div>

        {tab === 'run' && <div className="dbody">
          {loading && !run ? <div className="empty" style={{ padding: 60 }}><ThinkingOrb state="working" size={32} theme="dark" /><div>正在读取运行记录…</div></div>
            : loadErr ? <div className="notice bad"><Icon.alert /><span className="grow">{loadErr}</span><button className="btn sm" onClick={() => loadRun(cur)}>重试</button></div>
              : !run ? <div className="empty" style={{ padding: 60 }}><ThinkingOrb state="breathing" size={32} theme="dark" /><h3>这个流程还没有运行过</h3><button className="btn primary" onClick={onRerun}><Icon.play s={14} /> 运行流程</button></div>
                : <>
                  <div className={`rr-sum panel ${run.status}`}>
                    <div className="rr-sum-l">
                      <StatusBadge s={run.status} />
                      <div className="rr-big num">{done}<small>/ {t.length} 个环境完成</small></div>
                      <div className="rr-bar"><i className="ok" style={{ width: `${cnt(x => x.status === 'ok') / Math.max(1, t.length) * 100}%` }} /><i className="bad" style={{ width: `${cnt(x => x.status === 'failed') / Math.max(1, t.length) * 100}%` }} /><i className="run" style={{ width: `${cnt(x => x.status === 'running' || x.status === 'starting') / Math.max(1, t.length) * 100}%` }} /></div>
                    </div>
                    <div className="rr-counts">
                      <div><b className="num ok">{cnt(x => x.status === 'ok')}</b><span>成功</span></div>
                      <div><b className="num bad">{cnt(x => x.status === 'failed')}</b><span>失败</span></div>
                      <div><b className="num cy">{cnt(x => x.status === 'running' || x.status === 'starting')}</b><span>运行中</span></div>
                      <div><b className="num">{cnt(x => x.status === 'pending')}</b><span>排队</span></div>
                    </div>
                    <div className="rr-sum-acts">
                      {isRunning ? <button className="btn stop" onClick={cancel}><Icon.stop s={14} /> 停止运行</button>
                        : <button className="btn" onClick={onRerun}><Icon.refresh s={14} /> 再次运行</button>}
                    </div>
                  </div>
                  <div className="rr-targets">
                    {t.map((x, i) => <TargetCard key={x.profileId} t={x} total={run.stepCount} names={names} profile={profiles.find(p => p.id === x.profileId)} defaultOpen={t.length === 1 || (i === 0 && !isRunning) || x.status === 'failed' && cnt(y => y.status === 'failed') === 1} />)}
                  </div>
                </>}
        </div>}

        {tab === 'history' && <div className="dbody">
          {hist === null ? <div className="empty" style={{ padding: 60 }}><ThinkingOrb state="working" size={32} theme="dark" /></div>
            : hist.length === 0 ? <div className="empty" style={{ padding: 60 }}><ThinkingOrb state="breathing" size={32} theme="dark" /><h3>还没有运行记录</h3><div>运行后，每次的结果、日志和截图都会保存在这里。</div></div>
              : <div className="rr-hist">{hist.map(h => {
                const ok = h.targets.filter(x => x.status === 'ok').length;
                return <button key={h.id} className={`rr-hrow ${h.id === cur ? 'on' : ''}`} onClick={() => { setCur(h.id); setTab('run'); }}>
                  <StatusBadge s={h.status} />
                  <div className="rr-h-t"><b className="num">{clock(h.startedAt)}</b><span className="sub">{ago(h.startedAt)} · {h.targets.length} 个环境 · 并发 {h.concurrency}</span></div>
                  <div className="rr-h-r"><span className="num">{ok}/{h.targets.length}</span><span className="sub">成功</span></div>
                  <div className="rr-h-r"><span>{h.endedAt ? dur((h.endedAt - h.startedAt) * 1000) : '进行中'}</span><span className="sub">用时</span></div>
                  <RI.chev s={15} />
                </button>;
              })}</div>}
        </div>}
      </div>
    </div>
  );
};

const TargetCard: React.FC<{ t: RTarget; total: number; names: Record<string, string>; profile?: Profile; defaultOpen?: boolean }> = ({ t, total, names, profile, defaultOpen }) => {
  const [open, setOpen] = useState(!!defaultOpen);
  const [shot, setShot] = useState<string | null>(null);
  const last = t.cur || t.steps[t.steps.length - 1];
  const reached = Math.max(0, ...t.steps.map(s => s.index + 1), t.cur ? t.cur.index + 1 : 0);
  const pct = t.status === 'ok' ? 100 : total ? Math.min(100, reached / total * 100) : 0;
  const shots = t.steps.filter(s => s.shot);
  const vars = Object.entries(t.vars || {});
  const failed = t.steps.filter(s => s.status === 'failed').length;
  const logRef = useRef<HTMLDivElement>(null);
  useEffect(() => { const el = logRef.current; if (el && live(t.status)) el.scrollTop = el.scrollHeight; }, [t.steps.length, t.cur, open]);
  return (
    <div className={`rr-t panel ${t.status} ${open ? 'open' : ''}`}>
      <div className="rr-t-head" onClick={() => setOpen(o => !o)}>
        <Avatar seed={profile?.fingerprint?.noiseSeed || t.profileId} os={profile?.fingerprint?.os || 'windows'} />
        <div className="rr-t-main">
          <div className="rr-t-row"><b className="ell">{t.profileName}</b>{t.autoStarted && <span className="tag">自动启动</span>}<div className="spacer" /><StatusBadge s={t.status} /></div>
          <div className="rr-prog"><i style={{ width: `${pct}%` }} /></div>
          <div className="rr-t-row sub">
            <span className="num">步骤 {Math.min(reached, total)}/{total}</span>
            {t.status === 'running' && last && <span className="ell">· {t.cur ? '正在' : '刚完成'}「{names[last.stepId] || typeLabel(last.type)}」{last.iter ? ` · 第 ${last.iter} 轮` : ''}</span>}
            {t.status === 'starting' && <span>· 正在启动浏览器…</span>}
            {t.status === 'pending' && <span>· 等待空闲并发位</span>}
            {t.error && !live(t.status) && t.status !== 'ok' && <span className="ell err" title={t.error}>· {t.error}</span>}
            <div className="spacer" />
            {shots.length > 0 && <span className="num rr-meta-i" title="截图数"><RI.image s={12} /> {shots.length}</span>}
            {t.endedAt > 0 && t.startedAt > 0 && <span className="num rr-meta-i" title="用时"><RI.clock s={12} /> {dur((t.endedAt - t.startedAt) * 1000)}</span>}
          </div>
        </div>
        <span className={`rr-chev ${open ? 'on' : ''}`}><RI.chev s={16} /></span>
      </div>
      {open && <div className="rr-t-body">
        {(vars.length > 0 || shots.length > 0) && <div className="rr-side">
          {vars.length > 0 && <div className="rr-vars"><div className="rr-lbl"><RI.braces s={12} /> 提取的变量</div>
            {vars.map(([k, v]) => <div key={k} className="rr-var"><span className="k">{k}</span><span className="v" title={v}>{v || '""'}</span><button className="btn ghost icon" title="复制值" onClick={() => navigator.clipboard?.writeText(v)}><Icon.copy s={12} /></button></div>)}
          </div>}
          {shots.length > 0 && <div><div className="rr-lbl"><RI.camera s={12} /> 截图</div><div className="rr-shots">{shots.map((s, i) => <Thumb key={i} path={s.shot!} label={`#${s.index + 1}${s.iter ? ` · 第 ${s.iter} 轮` : ''}`} onOpen={setShot} />)}</div></div>}
        </div>}
        <div className="rr-lbl"><Icon.terminal s={12} /> 执行日志 <span className="sub">{t.steps.length} 条{failed ? ` · ${failed} 条失败` : ''}</span></div>
        <div className="rr-log" ref={logRef}>
          {t.steps.length === 0 && !t.cur && <div className="sub" style={{ padding: 14, textAlign: 'center' }}>{t.status === 'pending' ? '排队中，还没开始执行' : t.error || '没有执行记录'}</div>}
          {[...t.steps, ...(t.cur ? [t.cur] : [])].map((s, i) => <div key={i} className={`rr-l ${s.status}`}>
            <span className="st">{s.status === 'running' ? <ThinkingOrb state="working" size={20} theme="dark" /> : s.status === 'ok' ? <Icon.check s={12} /> : s.status === 'failed' ? <Icon.x s={12} /> : <RI.dot s={10} />}</span>
            <span className="ix num">#{s.index + 1}</span>
            <span className={`rpa-glyph xs ${toneOf(s.type)}`}><StepIcon type={s.type} s={11} /></span>
            <span className="nm">{names[s.stepId] || typeLabel(s.type)}{s.iter ? <em>第 {s.iter} 轮</em> : null}</span>
            <span className="lg" title={s.log}>{s.status === 'running' ? '执行中…' : s.log}</span>
            <span className="ms num">{s.status === 'running' ? '' : dur(s.ms)}</span>
          </div>)}
        </div>
      </div>}
      {shot && <Lightbox src={shot} onClose={() => setShot(null)} />}
    </div>
  );
};

const shotCache: Record<string, string> = {};
const Thumb: React.FC<{ path: string; label: string; onOpen: (src: string) => void }> = ({ path, label, onOpen }) => {
  const [src, setSrc] = useState(shotCache[path] || '');
  const [bad, setBad] = useState(false);
  useEffect(() => { if (src) return; api.RpaScreenshot(path).then(u => { shotCache[path] = u; setSrc(u); }).catch(() => setBad(true)); }, [path]);
  return <button className="rr-thumb" onClick={() => src && onOpen(src)} title={path}>
    {src ? <img src={src} alt="" /> : bad ? <span className="sub">无法读取</span> : <ThinkingOrb state="working" size={20} theme="dark" />}
    <span className="cap num">{label}</span>
  </button>;
};

const Lightbox: React.FC<{ src: string; onClose: () => void }> = ({ src, onClose }) => {
  const [out, close] = useExit(onClose);
  useEffect(() => { const k = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.stopPropagation(); close(); } }; window.addEventListener('keydown', k, true); return () => window.removeEventListener('keydown', k, true); }, [close]);
  return <div className={`overlay center rr-light ${out ? 'out' : ''}`} onMouseDown={() => close()}><img src={src} alt="" /></div>;
};
