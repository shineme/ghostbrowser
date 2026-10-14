import React, { useEffect, useMemo, useRef, useState } from 'react';
import { api } from './api';
import { Icon, Field, Toggle, Segmented, useToast, useConfirm, useExit } from './ui';
import {
  RI, RStep, RWorkflow, RIssue, DEF, STEP_DEFS, GROUPS, FieldDef, StepIcon, toneOf, typeLabel, summary, summaryBase, frameOf, frameText, newStep, flat, countSteps,
  updateStep, removeStep, moveStep, insertAfter, insertInto, cloneStep, findStep, siblingPos, varsBefore,
} from './rpaMeta';
import { RpaRecorder } from './RpaRecorder';

const err = (e: any) => String(e?.message || e);
const blankWf = (): RWorkflow => ({ id: '', name: '', description: '', steps: [], vars: {}, createdAt: 0, updatedAt: 0, lastRunAt: 0, lastStatus: '', lastRunId: '', runCount: 0 });

type Ctx = {
  sel: string; setSel: (id: string) => void; issues: Record<string, RIssue[]>; order: Record<string, number>;
  steps: RStep[]; act: (op: 'up' | 'down' | 'dup' | 'del' | 'toggle', id: string) => void; addInto: (parent: string, branch: 'children' | 'else', type: string) => void;
};

export const RpaEditor: React.FC<{ wf: RWorkflow | null; onClose: () => void; onSaved: () => void; onRun: (w: RWorkflow) => void }> = ({ wf, onClose, onSaved, onRun }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [d, setD] = useState<RWorkflow>(() => JSON.parse(JSON.stringify(wf || blankWf())));
  const [dirty, setDirty] = useState(() => !!wf && !wf.id && (wf.steps || []).length > 0);
  const [rec, setRec] = useState(false);
  const [sel, setSel] = useState<string>(() => (wf?.steps?.[0]?.id || ''));
  const [issues, setIssues] = useState<RIssue[]>([]);
  const [busy, setBusy] = useState(false);
  const [menu, setMenu] = useState(false);
  const [out, close] = useExit(onClose);
  const lastField = useRef<{ k: string; el: HTMLInputElement | HTMLTextAreaElement | null }>({ k: '', el: null });

  const edit = (fn: (w: RWorkflow) => RWorkflow) => { setD(fn); setDirty(true); };
  const setSteps = (fn: (s: RStep[]) => RStep[]) => edit(w => ({ ...w, steps: fn(w.steps) }));

  // Live validation (debounced) through the backend's own rules.
  useEffect(() => {
    const t = setTimeout(() => { api.ValidateWorkflow(d as any).then(r => setIssues((r || []) as any)).catch(() => {}); }, 320);
    return () => clearTimeout(t);
  }, [d]);
  const byStep = useMemo(() => { const m: Record<string, RIssue[]> = {}; issues.forEach(i => { if (i.stepId) (m[i.stepId] ||= []).push(i); }); return m; }, [issues]);
  const global = issues.filter(i => !i.stepId);
  const order = useMemo(() => { const m: Record<string, number> = {}; flat(d.steps).forEach((s, i) => (m[s.id] = i + 1)); return m; }, [d.steps]);
  const cur = sel ? findStep(d.steps, sel) : undefined;

  const add = (type: string) => {
    const s = newStep(type);
    setSteps(list => (sel && findStep(list, sel) ? insertAfter(list, sel, s) : [...list, s]));
    setSel(s.id); setMenu(false);
  };
  const addInto = (parent: string, branch: 'children' | 'else', type: string) => { const s = newStep(type); setSteps(l => insertInto(l, parent, branch, s)); setSel(s.id); };
  const act: Ctx['act'] = (op, id) => {
    if (op === 'up' || op === 'down') setSteps(l => moveStep(l, id, op === 'up' ? -1 : 1));
    if (op === 'dup') { const s = findStep(d.steps, id); if (s) { const c = cloneStep(s); setSteps(l => insertAfter(l, id, c)); setSel(c.id); } }
    if (op === 'toggle') setSteps(l => updateStep(l, id, s => ({ ...s, disabled: !s.disabled })));
    if (op === 'del') {
      const s = findStep(d.steps, id); const kids = s ? countSteps([...(s.children || []), ...(s.else || [])]) : 0;
      const doIt = () => { setSteps(l => removeStep(l, id)); if (sel === id) setSel(''); };
      if (kids) ask({ title: `删除「${s?.name || typeLabel(s!.type)}」？`, body: `里面的 ${kids} 个步骤会一起删除。`, ok: '删除', danger: true }).then(ok => ok && doIt());
      else doIt();
    }
  };
  const patch = (fn: (s: RStep) => RStep) => sel && setSteps(l => updateStep(l, sel, fn));

  const save = async (thenRun = false) => {
    if (!d.name.trim()) { toast('请先填写流程名称', 'bad'); return; }
    setBusy(true);
    try {
      const w = (await api.SaveWorkflow(d as any)) as any as RWorkflow;
      setD(x => ({ ...x, id: w.id, createdAt: w.createdAt, updatedAt: w.updatedAt })); setDirty(false); onSaved();
      if (thenRun) { onRun(w); return; }
      toast(issues.length ? `已保存，还有 ${issues.length} 处需要修改才能运行` : '流程已保存', issues.length ? 'bad' : 'ok');
    } catch (e) { toast(err(e), 'bad'); } finally { setBusy(false); }
  };
  const tryClose = async () => {
    if (dirty && !await ask({ title: '放弃未保存的修改？', body: '关闭后这次的改动不会保留。', ok: '放弃修改', danger: true })) return;
    close();
  };
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') { e.preventDefault(); save(); } };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });
  const insertVar = (v: string) => {
    const tok = `{{${v}}}`; const lf = lastField.current;
    if (cur && lf.k && lf.el && document.body.contains(lf.el)) {
      const el = lf.el; const a = el.selectionStart ?? el.value.length, b = el.selectionEnd ?? a;
      const nv = el.value.slice(0, a) + tok + el.value.slice(b);
      if (lf.k === '__name') patch(s => ({ ...s, name: nv })); else patch(s => ({ ...s, params: { ...s.params, [lf.k]: nv } }));
      requestAnimationFrame(() => { el.focus(); el.setSelectionRange(a + tok.length, a + tok.length); });
    } else { navigator.clipboard?.writeText(tok).catch(() => {}); toast(`已复制 ${tok}，粘贴到参数里即可`); }
  };

  // Recorded steps go after the selected step (or at the end); recorded
  // variables are added without overwriting values already set.
  const insertRecorded = (steps: RStep[], vars: Record<string, string>) => {
    if (!steps.length) return;
    const fresh = steps.map(s => ({ ...cloneStep(s), name: s.name || '' }));
    edit(w => {
      let list = w.steps;
      if (sel && findStep(list, sel)) { let after = sel; for (const s of fresh) { list = insertAfter(list, after, s); after = s.id; } }
      else list = [...list, ...fresh];
      const v = { ...(w.vars || {}) }; Object.entries(vars || {}).forEach(([k, x]) => { if (!(k in v)) v[k] = x; });
      return { ...w, steps: list, vars: v };
    });
    setSel(fresh[fresh.length - 1].id);
    toast(`已插入 ${fresh.length} 个录制的步骤`);
  };
  const ctx: Ctx = { sel, setSel, issues: byStep, order, steps: d.steps, act, addInto };
  const varRows = Object.entries(d.vars || {});
  const setVars = (rows: [string, string][]) => edit(w => ({ ...w, vars: Object.fromEntries(rows) }));

  return (
    <div className={`overlay ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && tryClose()}>
      {rec && <RpaRecorder target={cur ? { mode: 'append', after: cur.name || typeLabel(cur.type) } : { mode: 'append', after: '最后一步' }} onClose={() => setRec(false)} onDone={insertRecorded} />}
      <div className="drawer rpa-editor">
        <div className="dhead">
          <div className="glyph"><RI.flow s={20} /></div>
          <div style={{ minWidth: 0 }}>
            <h3>{wf?.id ? '编辑流程' : '新建流程'}{dirty && <span className="rpa-dirty">未保存</span>}</h3>
            <div className="sub">{countSteps(d.steps)} 个步骤 · 选择器支持 CSS、<code>xpath://…</code>、<code>text:文字</code> · Ctrl+S 保存</div>
          </div>
          <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={tryClose}><Icon.x /></button></div>
        </div>

        <div className="re-body">
          <div className="re-left">
            <div className="grid2 re-meta">
              <Field label="流程名称"><input type="text" value={d.name} placeholder="如：亚马逊商品价格采集" onChange={e => edit(w => ({ ...w, name: e.target.value }))} /></Field>
              <Field label="描述（可选）"><input type="text" value={d.description} placeholder="这个流程做什么" onChange={e => edit(w => ({ ...w, description: e.target.value }))} /></Field>
            </div>
            <details className="re-vars" open={varRows.length > 0 || undefined}>
              <summary><RI.braces s={14} /> 初始变量 <span className="sub">{varRows.length ? `${varRows.length} 个` : '可选，运行前预置的值'}</span></summary>
              <div className="re-vars-body">
                {varRows.map(([k, v], i) => <div className="re-var" key={i}>
                  <input type="text" className="mono" value={k} placeholder="名称" onChange={e => setVars(varRows.map((r, j) => (j === i ? [e.target.value.replace(/[^\w]/g, ''), r[1]] : r)))} />
                  <span className="eq">=</span>
                  <input type="text" value={v} placeholder="值" onChange={e => setVars(varRows.map((r, j) => (j === i ? [r[0], e.target.value] : r)))} />
                  <button className="btn ghost icon" onClick={() => setVars(varRows.filter((_, j) => j !== i))}><Icon.x s={14} /></button>
                </div>)}
                <button className="btn sm ghost" onClick={() => setVars([...varRows, [`var${varRows.length + 1}`, '']])}><Icon.plus s={13} /> 添加变量</button>
              </div>
            </details>

            <div className="re-steps-head">
              <span className="section-title" style={{ margin: 0, flex: 1 }}>步骤</span>
              <div className="re-addwrap">
                <button className="btn sm rec-btn" onClick={() => setRec(true)} title={cur ? `录制并插入到「${cur.name || typeLabel(cur.type)}」之后` : '录制并添加到末尾'}><span className="rec-dot-ic sm" /> 录制</button>
                <button className="btn sm primary" onClick={() => setMenu(m => !m)}><Icon.plus s={13} /> 添加步骤</button>
                {menu && <AddMenu onPick={add} onClose={() => setMenu(false)} hint={cur ? `插入到「${cur.name || typeLabel(cur.type)}」之后` : '添加到末尾'} />}
              </div>
            </div>
            {global.length > 0 && <div className="notice warn" style={{ marginBottom: 10 }}><Icon.alert /><span className="grow">{global.map(g => g.message).join('；')}</span></div>}
            <div className="re-tree">
              {d.steps.length === 0
                ? <div className="re-empty"><RI.flow s={22} /><b>从第一个步骤开始</b><span>常见的开头是「打开网址」</span>
                  <div className="row" style={{ gap: 8, flex: 'none' }}><button className="btn sm" onClick={() => add('goto')}><StepIcon type="goto" s={13} /> 打开网址</button><button className="btn sm ghost" onClick={() => setMenu(true)}>更多步骤…</button></div></div>
                : <StepList list={d.steps} ctx={ctx} />}
            </div>
          </div>

          <div className="re-right">
            {cur ? <Inspector key={cur.id} s={cur} issues={byStep[cur.id] || []} vars={varsBefore(d, cur.id)} onPatch={patch} onVar={insertVar}
              onFocusField={(k, el) => (lastField.current = { k, el })} onDelete={() => act('del', cur.id)} />
              : <div className="re-noinsp"><span className="rpa-glyph g-flow" style={{ width: 44, height: 44 }}><RI.pointer s={20} /></span><b>选择一个步骤</b><span>点击左侧步骤，在这里修改它的参数。</span></div>}
          </div>
        </div>

        <div className="dfoot">
          {issues.length
            ? <button className="rpa-issues" onClick={() => { const f = issues.find(i => i.stepId); if (f) setSel(f.stepId); }}><Icon.alert s={14} /> {issues.length} 处需要修改{issues.find(i => i.stepId) ? ' · 点击定位' : ''}</button>
            : <span className="rpa-ok"><Icon.check s={14} /> 检查通过，可以运行</span>}
          <div className="spacer" />
          <button className="btn" onClick={tryClose}>取消</button>
          <button className="btn" onClick={() => save()} disabled={busy}><Icon.check s={15} /> 保存</button>
          <button className="btn primary" onClick={() => save(true)} disabled={busy || issues.length > 0} title={issues.length ? '先修正标红的步骤' : ''}><Icon.play s={14} /> 保存并运行</button>
        </div>
      </div>
    </div>
  );
};

// ---------------- tree ----------------
const StepList: React.FC<{ list: RStep[]; ctx: Ctx }> = ({ list, ctx }) => <>{list.map(s => <StepNode key={s.id} s={s} ctx={ctx} />)}</>;

const StepNode: React.FC<{ s: RStep; ctx: Ctx }> = ({ s, ctx }) => {
  const [i, n] = siblingPos(ctx.steps, s.id);
  const bad = ctx.issues[s.id];
  const label = typeLabel(s.type);
  return (
    <div className={`rs-node ${s.disabled ? 'off' : ''}`}>
      <div className={`rs-row ${ctx.sel === s.id ? 'sel' : ''} ${bad ? 'bad' : ''}`} onClick={() => ctx.setSel(s.id)}>
        <span className="rs-idx num">{ctx.order[s.id]}</span>
        <span className={`rpa-glyph ${toneOf(s.type)}`}><StepIcon type={s.type} s={14} /></span>
        <div className="rs-t">
          <div className="rs-name"><span className="ell">{s.name || label}</span>{s.name && <span className="rs-type">{label}</span>}
            {s.disabled && <span className="rs-flag">已停用</span>}{s.continueOnError && <span className="rs-flag warn">出错继续</span>}</div>
          <div className="rs-sum" title={bad ? bad.map(b => b.message).join('\n') : summary(s)}>{bad ? <span className="rs-err">{bad[0].message}</span> : frameOf(s) ? <><span className="rs-frame" title={'所在框架：' + frameText(frameOf(s))}><RI.frame s={11} /><span>{frameText(frameOf(s))}</span></span>{summaryBase(s)}</> : summary(s)}</div>
        </div>
        <div className="rs-acts" onClick={e => e.stopPropagation()}>
          <button title="上移" disabled={i <= 0} onClick={() => ctx.act('up', s.id)}><RI.up s={14} /></button>
          <button title="下移" disabled={i >= n - 1} onClick={() => ctx.act('down', s.id)}><RI.down s={14} /></button>
          <button title={s.disabled ? '启用' : '停用'} onClick={() => ctx.act('toggle', s.id)}><RI.eyeoff s={14} /></button>
          <button title="复制" onClick={() => ctx.act('dup', s.id)}><Icon.copy s={13} /></button>
          <button title="删除" className="del" onClick={() => ctx.act('del', s.id)}><Icon.trash s={13} /></button>
        </div>
      </div>
      {s.type === 'condition' && <>
        <Branch label="则" tone="then" parent={s.id} branch="children" list={s.children || []} ctx={ctx} />
        <Branch label="否则" tone="else" parent={s.id} branch="else" list={s.else || []} ctx={ctx} />
      </>}
      {s.type === 'loop' && <Branch label="循环体" tone="loop" parent={s.id} branch="children" list={s.children || []} ctx={ctx} />}
    </div>
  );
};

const Branch: React.FC<{ label: string; tone: string; parent: string; branch: 'children' | 'else'; list: RStep[]; ctx: Ctx }> = ({ label, tone, parent, branch, list, ctx }) => {
  const [menu, setMenu] = useState(false);
  return (
    <div className={`rs-branch ${tone}`}>
      <div className="rs-blabel"><span>{label}</span>{list.length === 0 && <em>空</em>}</div>
      <StepList list={list} ctx={ctx} />
      <div className="re-addwrap">
        <button className="rs-add" onClick={() => setMenu(m => !m)}><Icon.plus s={12} /> 添加到「{label}」</button>
        {menu && <AddMenu onPick={t => { ctx.addInto(parent, branch, t); setMenu(false); }} onClose={() => setMenu(false)} hint={`添加到「${label}」末尾`} />}
      </div>
    </div>
  );
};

const AddMenu: React.FC<{ onPick: (t: string) => void; onClose: () => void; hint: string }> = ({ onPick, onClose, hint }) => {
  const ref = useRef<HTMLDivElement>(null);
  const [q, setQ] = useState('');
  useEffect(() => {
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node) && !(e.target as HTMLElement).closest?.('.re-addwrap')) onClose(); };
    const key = (e: KeyboardEvent) => { if (e.key === 'Escape') { e.stopPropagation(); onClose(); } };
    window.addEventListener('mousedown', down); window.addEventListener('keydown', key, true);
    return () => { window.removeEventListener('mousedown', down); window.removeEventListener('keydown', key, true); };
  }, [onClose]);
  useEffect(() => { ref.current?.scrollIntoView({ block: 'nearest' }); }, []);
  const hit = (d: typeof STEP_DEFS[0]) => !q || (d.label + d.desc + d.type).toLowerCase().includes(q.toLowerCase());
  return (
    <div className="re-menu" ref={ref}>
      <div className="re-menu-top"><Icon.search s={13} /><input autoFocus type="text" placeholder="搜索步骤" value={q} onChange={e => setQ(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { const f = STEP_DEFS.find(hit); if (f) onPick(f.type); } }} /><span className="sub">{hint}</span></div>
      <div className="re-menu-body">
        {GROUPS.map(g => { const items = STEP_DEFS.filter(d => d.group === g && hit(d)); if (!items.length) return null; return <div key={g} className="re-mgroup">
          <div className="re-mg-t">{g}</div>
          <div className="re-mg-items">{items.map(d => <button key={d.type} onClick={() => onPick(d.type)}>
            <span className={`rpa-glyph ${toneOf(d.type)}`}><StepIcon type={d.type} s={14} /></span>
            <span className="re-mi-t"><b>{d.label}</b><small>{d.desc}</small></span>
          </button>)}</div>
        </div>; })}
        {!STEP_DEFS.some(hit) && <div className="sub" style={{ padding: 16, textAlign: 'center' }}>没有匹配的步骤</div>}
      </div>
    </div>
  );
};

// ---------------- inspector ----------------
// Backend issue text "<step>：缺少<what>" → the param it refers to.
const MISSING: Record<string, string> = { 网址: 'url', 选择器: 'selector', 变量名: 'var', 代码: 'code', 判断代码: 'code', 文件路径: 'files', 标签页序号: 'index', 比较值: 'left', 'iframe 选择器': 'frame' };
const Inspector: React.FC<{ s: RStep; issues: RIssue[]; vars: string[]; onPatch: (fn: (s: RStep) => RStep) => void; onVar: (v: string) => void; onFocusField: (k: string, el: any) => void; onDelete: () => void }> = ({ s, issues, vars, onPatch, onVar, onFocusField, onDelete }) => {
  const def = DEF[s.type];
  const p = s.params || {};
  const set = (k: string, v: string) => onPatch(x => ({ ...x, params: { ...x.params, [k]: v } }));
  const fields = (def?.fields || []).filter(f => !f.show || f.show(p));
  // Pair consecutive half-width fields into a grid row.
  const rows: FieldDef[][] = [];
  fields.forEach(f => { const last = rows[rows.length - 1]; if (f.half && last && last.length === 1 && last[0].half) last.push(f); else rows.push([f]); });
  const focus = (k: string) => (e: React.FocusEvent<any>) => onFocusField(k, e.target);
  return (
    <div className="insp tabpane">
      <div className="insp-head">
        <span className={`rpa-glyph lg ${toneOf(s.type)}`}><StepIcon type={s.type} s={18} /></span>
        <div style={{ minWidth: 0, flex: 1 }}><b>{def?.label || s.type}</b><span className="sub">{def?.desc}</span></div>
        <button className="btn ghost icon danger" title="删除步骤" onClick={onDelete}><Icon.trash s={15} /></button>
      </div>
      {issues.length > 0 && <div className="notice bad" style={{ marginBottom: 14 }}><Icon.alert /><span className="grow">{issues.map(i => i.message).join('；')}</span></div>}
      <Field label="步骤名称（可选）"><input type="text" value={s.name || ''} placeholder={def?.label} onFocus={focus('__name')} onChange={e => onPatch(x => ({ ...x, name: e.target.value }))} /></Field>
      {rows.map((r, i) => <div key={i} className={r.length === 2 ? 'grid2' : ''}>{r.map(f => <ParamField key={f.k} f={f} v={p[f.k] ?? ''} set={v => set(f.k, v)} onFocus={focus(f.k)} bad={issues.some(x => MISSING[(x.message.split('缺少')[1] || '').trim()] === f.k)} />)}</div>)}
      {s.type === 'break_loop' && <div className="sub" style={{ marginBottom: 14 }}>没有参数。执行到这里会立即结束最近的一层循环，继续执行循环之后的步骤。</div>}
      {def?.container && <div className="sub insp-tip">{s.type === 'condition' ? '在左侧「则 / 否则」分支里添加要执行的步骤。' : '在左侧「循环体」里添加每一轮要执行的步骤；用「跳出循环」可提前结束。'}</div>}

      <div className="section-title" style={{ marginTop: 18 }}>执行选项</div>
      <Toggle on={!!s.disabled} onChange={v => onPatch(x => ({ ...x, disabled: v }))} label="禁用" hint="运行时跳过这一步（及其内部步骤）" />
      <div style={{ height: 8 }} />
      <Toggle on={!!s.continueOnError} onChange={v => onPatch(x => ({ ...x, continueOnError: v }))} label="出错继续" hint="这一步失败时记录错误，继续执行后面的步骤" />

      <div className="section-title" style={{ marginTop: 18 }}>可用变量</div>
      {vars.length
        ? <><div className="var-chips">{vars.map(v => <button key={v} className="var-chip" onClick={() => onVar(v)} title={`插入 {{${v}}} 到光标处`}>{`{{${v}}}`}</button>)}</div>
          <div className="sub" style={{ marginTop: 8, fontSize: 11.5 }}>点击插入到最后编辑的输入框；执行 JS 里请用 <code>vars.名字</code>。</div></>
        : <div className="sub" style={{ fontSize: 12 }}>前面的步骤还没有定义变量。用「提取文本」或「执行 JS / 截图」的「存入变量」产生变量，之后的步骤可用 <code>{'{{名字}}'}</code> 引用。</div>}
    </div>
  );
};

const ParamField: React.FC<{ f: FieldDef; v: string; set: (v: string) => void; onFocus: (e: React.FocusEvent<any>) => void; bad?: boolean }> = ({ f, v, set, onFocus, bad }) => {
  const cls = bad ? 'is-bad' : '';
  switch (f.kind) {
    case 'bool': return <div style={{ marginBottom: 12 }}><Toggle on={v === 'true'} onChange={b => set(b ? 'true' : 'false')} label={f.label} hint={f.hint} /></div>;
    case 'seg': return <Field label={f.label} hint={f.hint}><Segmented className="block" items={f.opts!.map(([a, b]) => [a, b] as [string, string])} value={v || f.opts![0][0]} onChange={set} /></Field>;
    case 'pick': return <Field label={f.label} hint={f.hint}><PickField value={v || f.opts![0][0]} options={f.opts!} onChange={set} /></Field>;
    case 'code': return <Field label={f.label} hint={f.hint}><textarea className={`rpa-code ${cls}`} rows={7} spellCheck={false} value={v} placeholder={f.ph} onFocus={onFocus} onChange={e => set(e.target.value)}
      onKeyDown={e => { if (e.key === 'Tab') { e.preventDefault(); const el = e.currentTarget; const a = el.selectionStart; const nv = v.slice(0, a) + '  ' + v.slice(el.selectionEnd); set(nv); requestAnimationFrame(() => el.setSelectionRange(a + 2, a + 2)); } }} /></Field>;
    case 'area': return <Field label={f.label} hint={f.hint}><textarea className={`rpa-area ${cls}`} rows={3} value={v} placeholder={f.ph} onFocus={onFocus} onChange={e => set(e.target.value)} /></Field>;
    case 'num': return <Field label={f.label} hint={f.hint}><div className="in-unit"><input type="text" inputMode="numeric" className={`num ${cls}`} value={v} placeholder={f.ph} onFocus={onFocus} onChange={e => set(e.target.value)} />{f.unit && <span>{f.unit}</span>}</div></Field>;
    case 'var': return <Field label={f.label} hint={f.hint}><div className="in-var"><span>{'{{'}</span><input type="text" className={`mono ${cls}`} value={v} placeholder={f.ph} onFocus={onFocus} onChange={e => set(e.target.value.replace(/[^\w\u4e00-\u9fff]/g, ''))} /><span>{'}}'}</span></div></Field>;
    case 'selector': return <Field label={f.label} hint={f.hint || '如 #login、.btn-primary、xpath://button[1]、text:登录'}><input type="text" className={`mono ${cls}`} value={v} placeholder={f.ph || 'CSS / xpath: / text:'} onFocus={onFocus} onChange={e => set(e.target.value)} spellCheck={false} /></Field>;
    default: return <Field label={f.label} hint={f.hint}><input type="text" className={cls} value={v} placeholder={f.ph} onFocus={onFocus} onChange={e => set(e.target.value)} /></Field>;
  }
};

/** Full-width dropdown in the Pick style that always shows its current value. */
const PickField: React.FC<{ value: string; options: [string, string][]; onChange: (v: string) => void }> = ({ value, options, onChange }) => {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const down = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setOpen(false); };
    window.addEventListener('mousedown', down); return () => window.removeEventListener('mousedown', down);
  }, [open]);
  return (
    <div className={`pick pickfield ${open ? 'open' : ''}`} ref={ref}>
      <button type="button" className="pick-btn" onClick={() => setOpen(o => !o)}><span className="pick-o">{options.find(o => o[0] === value)?.[1] || value}</span>
        <svg className="pick-c" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round"><path d="M7 10l5 5 5-5" /></svg></button>
      {open && <div className="pick-menu left">{options.map(([v, l]) => <button type="button" key={v} className={v === value ? 'on' : ''} onClick={() => { onChange(v); setOpen(false); }}>
        <span className="pick-tick">{v === value && <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round"><path d="M5 12.5l4.5 4.5L19 7.5" /></svg>}</span><span className="pick-o">{l}</span></button>)}</div>}
    </div>
  );
};
