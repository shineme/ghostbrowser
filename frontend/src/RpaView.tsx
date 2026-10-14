import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Profile } from './api';
import { Icon, Field, Toggle, Segmented, Pick, StatCard, Avatar, ago, useToast, useConfirm, useExit } from './ui';
import { RI, RWorkflow, REvent, StepIcon, toneOf, typeLabel, countSteps, StatusBadge, newStep, flat } from './rpaMeta';
import { RpaEditor } from './RpaEditor';
import { RpaRunDrawer } from './RpaRunDrawer';
import { RpaRecorder, recName } from './RpaRecorder';

const err = (e: any) => String(e?.message || e);

// A ready-to-run example so the empty page has an obvious first step.
const sampleWorkflow = (): Partial<RWorkflow> => {
  const s = (type: string, params: Record<string, string>, extra: any = {}) => ({ ...newStep(type), params: { ...newStep(type).params, ...params }, ...extra });
  return {
    id: '', name: '示例：百度搜索并采集首条结果', description: '打开百度搜索关键词，读取第一条结果标题，有结果就截图，演示变量、条件和循环', vars: { keyword: 'GhostBrowser 指纹浏览器' },
    steps: [
      s('goto', { url: 'https://www.baidu.com' }),
      s('fill', { selector: '#kw', text: '{{keyword}}' }),
      s('click', { selector: '#su' }),
      s('wait_for', { selector: '#content_left', state: 'visible' }),
      s('extract_text', { selector: '#content_left h3', var: 'title' }),
      s('condition', { kind: 'exists', selector: '#content_left h3' }, { children: [s('screenshot', { saveAs: 'shot' })], else: [s('execute_js', { code: "return '没有搜索结果';", saveAs: 'note' })] }),
      s('loop', { mode: 'count', count: '3', indexVar: 'i' }, { children: [s('scroll', { mode: 'by', x: '0', y: '600' }), s('delay', { ms: '800', maxMs: '1500' })] }),
    ],
  };
};

export function RpaView({ profiles }: { profiles: Profile[]; reload?: () => void }) {
  const toast = useToast();
  const ask = useConfirm();
  const [list, setList] = useState<RWorkflow[] | null>(null);
  const [loadErr, setLoadErr] = useState('');
  const [q, setQ] = useState('');
  const [status, setStatus] = useState('all');
  const [editing, setEditing] = useState<RWorkflow | 'new' | null>(null);
  const [running, setRunning] = useState<RWorkflow | null>(null); // run dialog target
  const [drawer, setDrawer] = useState<{ wf: RWorkflow; runId: string } | null>(null);
  const [importing, setImporting] = useState<string | null>(null);
  const [live, setLive] = useState<Record<string, string>>({}); // runId -> status while running
  const [recording, setRecording] = useState(false);

  const load = useCallback(async () => {
    try { setList(((await api.ListWorkflows()) || []) as any); setLoadErr(''); }
    catch (e) { setLoadErr(err(e)); setList(l => l || []); }
  }, []);
  useEffect(() => { load(); }, [load]);
  useEffect(() => onEvent('rpa:event', (e: REvent) => {
    if (e.phase !== 'run') return;
    setLive(m => ({ ...m, [e.runId]: e.status }));
    if (e.status !== 'running') load();
  }), [load]);

  const shown = useMemo(() => {
    const s = q.trim().toLowerCase();
    return (list || []).filter(w => (status === 'all' || (status === 'never' ? !w.lastStatus : w.lastStatus === status)) &&
      (!s || [w.name, w.description, ...flat(w.steps).map(x => typeLabel(x.type))].some(v => (v || '').toLowerCase().includes(s))));
  }, [list, q, status]);

  const dup = async (w: RWorkflow) => { try { const n = await api.DuplicateWorkflow(w.id); toast(`已复制为「${n.name}」`); load(); } catch (e) { toast(err(e), 'bad'); } };
  const del = async (w: RWorkflow) => {
    if (!await ask({ title: `删除流程「${w.name}」？`, body: `共 ${countSteps(w.steps)} 个步骤，运行记录和截图也会一并删除，无法恢复。`, ok: '删除', danger: true })) return;
    try { await api.DeleteWorkflow(w.id); toast('流程已删除'); load(); } catch (e) { toast(err(e), 'bad'); }
  };
  const exp = async (w: RWorkflow) => {
    try { const p = await api.SaveWorkflowFile(w.id); if (p) toast(`已导出到 ${p}`); }
    catch (e) { toast(err(e), 'bad'); }
  };
  const createSample = async () => {
    try { const w = await api.SaveWorkflow(sampleWorkflow() as any); toast('已创建示例流程，打开看看每一步是怎么配置的'); await load(); setEditing(w as any); }
    catch (e) { toast(err(e), 'bad'); }
  };
  const openRun = (w: RWorkflow) => setDrawer({ wf: w, runId: w.lastRunId });
  const isLive = (w: RWorkflow) => w.lastStatus === 'running' || live[w.lastRunId] === 'running';

  const overlays = <>
    {editing && <RpaEditor wf={editing === 'new' ? null : editing} onClose={() => setEditing(null)} onSaved={load}
      onRun={w => { setEditing(null); setRunning(w); }} />}
    {running && <RunDialog wf={running} profiles={profiles} onClose={() => setRunning(null)}
      onStarted={runId => { const w = running; setRunning(null); setDrawer({ wf: w, runId }); load(); }} />}
    {drawer && <RpaRunDrawer wf={drawer.wf} runId={drawer.runId} profiles={profiles} onClose={() => setDrawer(null)}
      onRerun={() => { const w = drawer.wf; setDrawer(null); setRunning(w); }} />}
    {recording && <RpaRecorder target={{ mode: 'new' }} profiles={profiles} onClose={() => setRecording(false)}
      onDone={(steps, vars) => setEditing({ id: '', name: recName(), description: '由录制生成', steps, vars, createdAt: 0, updatedAt: 0, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0 } as RWorkflow)} />}
    {importing !== null && <ImportModal initial={importing} onClose={() => setImporting(null)} onDone={w => { load(); setEditing(w); }} />}
  </>;

  if (list === null) return <div className="empty"><ThinkingOrb state="working" size={32} theme="dark" /><div>正在读取流程…</div></div>;

  if (list.length === 0) return (
    <div className="empty">
      <div className="halo"><ThinkingOrb state="working" size={64} theme="dark" /></div>
      <h3>还没有自动化流程</h3>
      <div style={{ maxWidth: 460 }}>把打开网页、点击、输入、采集数据、截图这些重复操作编排成流程，<br />一键在多个环境里批量执行，进度和结果实时可见。</div>
      {loadErr && <div className="notice bad" style={{ marginTop: 4 }}><Icon.alert />{loadErr}</div>}
      <div className="actions">
        <button className="btn primary" onClick={createSample}><Icon.sparkles s={15} /> 新建示例流程</button>
        <button className="btn rec-btn" onClick={() => setRecording(true)}><span className="rec-dot-ic sm" /> 录制操作</button>
        <button className="btn" onClick={() => setEditing('new')}><Icon.plus s={15} /> 空白流程</button>
        <button className="btn ghost" onClick={() => setImporting('')}><Icon.upload s={15} /> 导入</button>
      </div>
      <div className="rpa-hint-steps">
        {['goto', 'fill', 'click', 'extract_text', 'condition', 'loop'].map((t, i) => <React.Fragment key={t}>
          {i > 0 && <i className="wire" />}
          <span className={`rpa-glyph ${toneOf(t)}`} title={typeLabel(t)}><StepIcon type={t} s={14} /></span>
        </React.Fragment>)}
      </div>
      {overlays}
    </div>
  );

  const totalRuns = list.reduce((n, w) => n + (w.runCount || 0), 0);
  const ran = list.filter(w => w.lastStatus && w.lastStatus !== 'running');
  const okN = ran.filter(w => w.lastStatus === 'ok').length;
  const cnt = (s: string) => list.filter(w => (s === 'never' ? !w.lastStatus : w.lastStatus === s)).length;

  return (
    <>
      <div className="stat-row">
        <StatCard l="流程" n={list.length} ic="layers" pct={100} />
        <StatCard l="步骤总数" n={list.reduce((n, w) => n + countSteps(w.steps), 0)} ic="terminal" tone="vi" pct={70} />
        <StatCard l="累计运行" n={totalRuns} ic="play" tone="cy" pct={Math.min(100, totalRuns)} />
        <StatCard l="最近一次成功" n={okN} of={ran.length} ic="check" tone="ok" pct={ran.length ? okN / ran.length * 100 : 0} />
      </div>
      {loadErr && <div className="notice bad"><Icon.alert /><b>读取流程失败</b><span className="sub grow">{loadErr}</span><button className="btn sm" onClick={load}>重试</button></div>}
      <div className="toolbar">
        <div className="search"><Icon.search /><input type="text" placeholder="搜索流程名称、描述或步骤类型" value={q} onChange={e => setQ(e.target.value)} /></div>
        <Pick label="上次结果" value={status} onChange={setStatus} icon={<Icon.check />} options={[['all', '全部', list.length], ['ok', '成功', cnt('ok')], ['partial', '部分成功', cnt('partial')], ['failed', '失败', cnt('failed')], ['never', '从未运行', cnt('never')]]} />
        <div className="spacer" />
        <button className="btn" onClick={() => setImporting('')}><Icon.upload s={14} /> 导入</button>
        <button className="btn rec-btn" onClick={() => setRecording(true)} title="在浏览器里操作一遍，自动生成流程"><span className="rec-dot-ic sm" /> 录制</button>
        <button className="btn primary" onClick={() => setEditing('new')}><Icon.plus s={15} /> 新建流程</button>
      </div>

      {shown.length === 0
        ? <div className="panel empty" style={{ padding: '56px 20px' }}><ThinkingOrb state="searching" size={32} theme="dark" /><div>没有匹配「{q || '当前筛选'}」的流程</div><button className="btn sm" onClick={() => { setQ(''); setStatus('all'); }}>清除筛选</button></div>
        : <div className="rpa-grid">
          {shown.map((w, i) => {
            const n = countSteps(w.steps);
            const types = flat(w.steps).filter(s => !s.disabled).map(s => s.type);
            const run = isLive(w);
            return <div key={w.id} className={`rpa-card panel spot rise ${run ? 'live' : ''}`} style={{ ['--i' as any]: Math.min(i, 12) }}>
              <div className="rc-head">
                <span className="rc-glyph"><RI.flow s={18} /></span>
                <div className="rc-t">
                  <div className="nm" title={w.name}>{w.name}</div>
                  <div className="sub" title={w.description}>{w.description || '没有描述'}</div>
                </div>
              </div>
              <div className="rc-pipe" title={types.map(typeLabel).join(' → ')}>
                {types.slice(0, 9).map((t, j) => <React.Fragment key={j}>{j > 0 && <i className="wire" />}<span className={`rpa-glyph sm ${toneOf(t)}`}><StepIcon type={t} s={12} /></span></React.Fragment>)}
                {types.length > 9 && <span className="more num">+{types.length - 9}</span>}
              </div>
              <div className="rc-meta">
                <span><b className="num">{n}</b> 步</span><i />
                <span>运行 <b className="num">{w.runCount || 0}</b> 次</span>
                <div className="spacer" />
                {w.lastRunId
                  ? <button className="rc-last" onClick={() => openRun(w)} title="查看运行详情">
                    {run ? <StatusBadge s="running" /> : <StatusBadge s={w.lastStatus} />}<span className="sub">{run ? '查看进度' : ago(w.lastRunAt)}</span>
                  </button>
                  : <StatusBadge s="" />}
              </div>
              <div className="rc-acts">
                <button className="btn go sm" onClick={() => setRunning(w)} disabled={!n}><Icon.play s={13} /> 运行</button>
                <button className="btn sm" onClick={() => setEditing(w)}><Icon.edit s={13} /> 编辑</button>
                <div className="spacer" />
                <button className="btn ghost icon" title="运行记录" onClick={() => openRun(w)}><RI.history s={15} /></button>
                <button className="btn ghost icon" title="复制" onClick={() => dup(w)}><Icon.copy s={15} /></button>
                <button className="btn ghost icon" title="导出为 JSON 文件" onClick={() => exp(w)}><Icon.download s={15} /></button>
                <button className="btn ghost icon danger" title="删除" onClick={() => del(w)}><Icon.trash s={15} /></button>
              </div>
            </div>;
          })}
        </div>}
      {overlays}
    </>
  );
}

// ---------------- import ----------------
const ImportModal: React.FC<{ initial: string; onClose: () => void; onDone: (w: RWorkflow) => void }> = ({ initial, onClose, onDone }) => {
  const toast = useToast();
  const [text, setText] = useState(initial);
  const [e, setE] = useState('');
  const [busy, setBusy] = useState(false);
  const [out, close] = useExit(onClose);
  useEffect(() => { const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !document.querySelector('.modal.sm')) close(); }; window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k); }, [close]);
  const pickFile = async () => { try { const t = await api.OpenWorkflowFile(); if (t) { setText(t); setE(''); } } catch (x) { setE(err(x)); } };
  const go = async () => {
    setBusy(true); setE('');
    try { const w = await api.ImportWorkflow(text); toast(`已导入「${w.name}」`); onDone(w as any); close(); }
    catch (x) { setE(err(x)); } finally { setBusy(false); }
  };
  let preview = '';
  try { const o = JSON.parse(text); const w = o.workflow || o; if (w?.steps) preview = `${w.name || '未命名流程'} · ${countSteps(w.steps)} 个步骤`; } catch { /* not yet valid */ }
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={ev => ev.target === ev.currentTarget && close()}>
      <div className="modal" style={{ width: 600 }}>
        <div className="dhead"><div className="glyph"><Icon.upload s={18} /></div><div><h3>导入流程</h3><div className="sub">选择导出的 .rpa.json 文件，或直接粘贴 JSON</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <button className="rpa-drop" onClick={pickFile}><Icon.folder s={20} /><div><b>从文件选择</b><span>GhostBrowser 导出的流程 JSON，最大 4 MB</span></div></button>
          <Field label="或粘贴 JSON" hint={preview ? `识别到：${preview}` : '支持导出格式（含 format/workflow）或直接的流程对象'}>
            <textarea className="rpa-code" rows={10} value={text} onChange={ev => { setText(ev.target.value); setE(''); }} placeholder={'{\n  "format": "ghostbrowser-rpa",\n  "workflow": { "name": "…", "steps": [ … ] }\n}'} spellCheck={false} />
          </Field>
          {e && <div className="notice bad" style={{ marginBottom: 0 }}><Icon.alert /><span className="grow">{e}</span></div>}
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button><button className="btn primary" disabled={!text.trim() || busy} onClick={go}><Icon.check s={15} /> {busy ? '导入中…' : '导入并打开'}</button></div>
      </div>
    </div>
  );
};

// ---------------- run dialog ----------------
const RunDialog: React.FC<{ wf: RWorkflow; profiles: Profile[]; onClose: () => void; onStarted: (runId: string) => void }> = ({ wf, profiles, onClose, onStarted }) => {
  const toast = useToast();
  const [q, setQ] = useState('');
  const [group, setGroup] = useState('__all');
  const [pick, setPick] = useState<Set<string>>(new Set());
  const [conc, setConc] = useState<'1' | '3' | '5' | '10'>('3');
  const [autoStart, setAutoStart] = useState(true);
  const [closeAfter, setCloseAfter] = useState(false);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);
  const [e, setE] = useState('');
  const [out, close] = useExit(onClose);
  useEffect(() => { const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !document.querySelector('.modal.sm')) close(); }; window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k); }, [close]);
  useEffect(() => { api.Running().then(r => setOpen(new Set((r || []).map((x: any) => x.profileId)))).catch(() => {}); }, []);
  const groups = useMemo(() => [...new Set(profiles.map(p => p.group || ''))].sort(), [profiles]);
  const shown = profiles.filter(p => (group === '__all' || (p.group || '') === group) &&
    (!q || `${p.name} ${p.group} ${(p.tags || []).join(' ')} ${p.notes}`.toLowerCase().includes(q.toLowerCase())));
  const allOn = shown.length > 0 && shown.every(p => pick.has(p.id));
  const toggle = (id: string) => setPick(s => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n; });
  const ids = profiles.filter(p => pick.has(p.id)).map(p => p.id);
  const closed = ids.filter(id => !open.has(id)).length;
  const go = async () => {
    setBusy(true); setE('');
    try { const runId = await api.RunWorkflowWith(wf.id, ids, { concurrency: +conc, autoStart, closeAfter } as any); toast(`已开始在 ${ids.length} 个环境运行`); onStarted(runId); }
    catch (x) { setE(err(x)); setBusy(false); }
  };
  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={ev => ev.target === ev.currentTarget && close()}>
      <div className="modal" style={{ width: 620 }}>
        <div className="dhead"><div className="glyph"><Icon.play s={18} /></div><div style={{ minWidth: 0 }}><h3>运行流程</h3><div className="sub ell">{wf.name} · {countSteps(wf.steps)} 个步骤</div></div><div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={() => close()}><Icon.x /></button></div></div>
        <div className="dbody">
          <div className="row" style={{ gap: 8, marginBottom: 8, alignItems: 'center' }}>
            <div className="rpa-search"><Icon.search s={14} /><input type="text" placeholder="搜索环境名称、分组、标签" value={q} onChange={ev => setQ(ev.target.value)} /></div>
            <div style={{ flex: 'none' }}><Pick label="分组" value={group} onChange={setGroup} align="right" icon={<Icon.folder />}
              options={[['__all', '全部分组', profiles.length] as [string, string, number], ...groups.map(g => [g, g || '未分组', profiles.filter(p => (p.group || '') === g).length] as [string, string, number])]} /></div>
          </div>
          <div className="rpa-pickhead">
            <button className="rpa-allbtn" disabled={!shown.length} onClick={() => setPick(s => { const n = new Set(s); shown.forEach(p => allOn ? n.delete(p.id) : n.add(p.id)); return n; })}>
              <span className={`chk ${allOn ? 'on' : ''}`}><Icon.check s={11} /></span>
              <span>{allOn ? '取消全选' : '全选'}{group !== '__all' || q ? `（当前筛选 ${shown.length} 个）` : ''}</span>
            </button>
            <div className="spacer" />
            <span className="sub num">已选 {ids.length} / {profiles.length}</span>
            {ids.length > 0 && <button className="btn sm ghost" onClick={() => setPick(new Set())}>清空</button>}
          </div>
          <div className="assign-list rpa-plist">
            {shown.map(p => <div key={p.id} className={`assign-row ${pick.has(p.id) ? 'on' : ''}`} onClick={() => toggle(p.id)}>
              <span className={`chk ${pick.has(p.id) ? 'on' : ''}`}><Icon.check s={11} /></span>
              <Avatar seed={p.fingerprint?.noiseSeed || p.id} os={p.fingerprint?.os || 'windows'} running={open.has(p.id)} />
              <div style={{ flex: 1, minWidth: 0 }}><div className="nm ell">{p.name}</div><div className="sub ell">{p.group || '未分组'}{(p.tags || []).length ? ' · ' + p.tags.join('、') : ''}</div></div>
              {open.has(p.id) ? <span className="badge ok"><span className="d" />已打开</span> : <span className="badge unknown">未打开</span>}
            </div>)}
            {shown.length === 0 && <div className="sub" style={{ padding: 22, textAlign: 'center' }}>{profiles.length ? '没有符合条件的环境' : '还没有环境，请先在「环境」页创建'}</div>}
          </div>
          <Field label="并发数" hint={`同时运行 ${conc} 个环境${+conc > 1 ? '，其余排队依次执行' : '，逐个执行'}`}>
            <Segmented className="block" items={[['1', '1'], ['3', '3'], ['5', '5'], ['10', '10']]} value={conc} onChange={setConc} />
          </Field>
          <div className="rpa-opts"><Toggle on={autoStart} onChange={setAutoStart} label="自动启动未打开的环境" hint={closed ? `所选中有 ${closed} 个环境未打开${autoStart ? '，会先自动启动' : '，它们会直接失败'}` : '浏览器没开时先帮你启动'} />
          <div className={autoStart ? '' : 'rpa-dim'}><Toggle on={closeAfter && autoStart} onChange={v => autoStart && setCloseAfter(v)} label="结束后关闭" hint="只关闭本次自动启动的浏览器，你自己打开的不会动" /></div></div>
          {e && <div className="notice bad" style={{ margin: '12px 0 0' }}><Icon.alert /><span className="grow">{e}</span></div>}
        </div>
        <div className="dfoot"><div className="spacer" /><button className="btn" onClick={() => close()}>取消</button>
          <button className="btn primary" disabled={!ids.length || busy} onClick={go}><Icon.play s={14} /> {busy ? '正在启动…' : ids.length ? `在 ${ids.length} 个环境运行` : '请选择环境'}</button></div>
      </div>
    </div>
  );
};
