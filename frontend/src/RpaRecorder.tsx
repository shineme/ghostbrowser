// Action recorder: pick a profile, work in its browser while every action
// streams in as a step ("rpa:record"), then review and merge into the editor.
import React, { useEffect, useRef, useState } from 'react';
import { ThinkingOrb } from 'thinking-orbs';
import { api, onEvent, Profile } from './api';
import { Icon, Toggle, Avatar, useToast, useConfirm, useExit } from './ui';
import { RStep, StepIcon, toneOf, typeLabel, summary } from './rpaMeta';

const err = (e: any) => String(e?.message || e);

export type RecState = {
  active: boolean; paused: boolean; profileId: string; profileName: string; startedAt: number; pausedMs: number; pausedAt: number;
  steps: RStep[]; vars: Record<string, string>; tabs: number; reason?: string; stoppedBy?: string; options?: any;
};
export type RecTarget = { mode: 'new' } | { mode: 'append'; after: string };

const isThink = (s: RStep) => s.type === 'delay' && s.params?._rec === 'think';
const mmss = (ms: number) => { const t = Math.max(0, Math.floor(ms / 1000)); return `${String(Math.floor(t / 60)).padStart(2, '0')}:${String(t % 60).padStart(2, '0')}`; };
const label = (s: RStep) => s.name && !isThink(s) ? s.name : typeLabel(s.type);
// What a recorded step does, without the fallback XPath noise.
const brief = (s: RStep) => summary({ ...s, params: { ...s.params, selector: (s.params?.selector || '').split(' || ')[0] } });

// The main editable params of a recorded step.
const EDIT_KEYS: Record<string, [string, string][]> = {
  goto: [['url', '网址']], click: [['selector', '选择器']], fill: [['selector', '选择器'], ['text', '输入内容']],
  wait_for: [['selector', '选择器']], extract_text: [['selector', '选择器'], ['var', '保存到变量']], upload: [['selector', '选择器'], ['files', '文件路径']],
  delay: [['ms', '毫秒']], select_tab: [['index', '标签页序号']], close_tab: [['index', '标签页序号']], scroll: [['x', 'X'], ['y', 'Y']],
};

export const RpaRecorder: React.FC<{ target: RecTarget; profiles?: Profile[]; onDone: (steps: RStep[], vars: Record<string, string>) => void; onClose: () => void }> = ({ target, profiles: given, onDone, onClose }) => {
  const toast = useToast();
  const ask = useConfirm();
  const [phase, setPhase] = useState<'setup' | 'rec' | 'review'>('setup');
  const [st, setSt] = useState<RecState | null>(null);
  const [out, close] = useExit(onClose);
  const phaseRef = useRef(phase); phaseRef.current = phase;

  // Pick up a recording that is already running (panel reopened).
  useEffect(() => { api.RecordingState().then((s: any) => { if (s?.active) { setSt(s); setPhase('rec'); } }).catch(() => {}); }, []);
  useEffect(() => onEvent('rpa:record', (s: RecState) => {
    if (phaseRef.current === 'setup') return;
    setSt(s);
    if (!s.active && phaseRef.current === 'rec') {
      setPhase('review');
      if (s.stoppedBy === 'overlay') toast('已在浏览器浮窗里停止录制');
      else if (s.reason) toast(s.reason, 'bad');
    }
  }), [toast]);

  const tryClose = async () => {
    if (phase === 'rec') {
      if (!await ask({ title: '放弃这次录制？', body: `已录制 ${st?.steps.filter(s => !isThink(s)).length || 0} 个步骤，放弃后不会保存。浏览器会保持打开。`, ok: '放弃录制', danger: true })) return;
      await api.StopRecording().catch(() => {});
    } else if (phase === 'review' && (st?.steps.length || 0) > 0) {
      if (!await ask({ title: '丢弃录制的步骤？', body: '关闭后这些步骤不会加入流程。', ok: '丢弃', danger: true })) return;
    }
    close();
  };
  useEffect(() => {
    const k = (e: KeyboardEvent) => { if (e.key === 'Escape' && !document.querySelector('.modal.sm')) tryClose(); };
    window.addEventListener('keydown', k); return () => window.removeEventListener('keydown', k);
  });

  return (
    <div className={`overlay center ${out ? 'out' : ''}`} onMouseDown={e => e.target === e.currentTarget && phase === 'setup' && tryClose()}>
      <div className={`modal rec-modal ${phase}`}>
        {phase === 'setup' && <Setup target={target} given={given} onCancel={tryClose} onStarted={s => { setSt(s); setPhase('rec'); }} />}
        {phase === 'rec' && st && <Live st={st} setSt={setSt} onStop={async () => { const s = await api.StopRecording(); setSt(s as any); setPhase('review'); }} onCancel={tryClose} />}
        {phase === 'review' && st && <Review st={st} target={target} onAgain={() => { setSt(null); setPhase('setup'); }} onDiscard={tryClose}
          onUse={(steps, vars) => { onDone(steps, vars); close(); }} />}
      </div>
    </div>
  );
};

// ---------------- setup ----------------
const Setup: React.FC<{ target: RecTarget; given?: Profile[]; onCancel: () => void; onStarted: (s: RecState) => void }> = ({ target, given, onCancel, onStarted }) => {
  const [profiles, setProfiles] = useState<Profile[] | null>(given || null);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [q, setQ] = useState('');
  const [pick, setPick] = useState(() => localStorage.getItem('gb.recProfile') || '');
  const [overlay, setOverlay] = useState(localStorage.getItem('gb.recOverlay') !== '0');
  const [scroll, setScroll] = useState(false);
  const [pwVar, setPwVar] = useState(true);
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [e, setE] = useState('');
  useEffect(() => { if (!given) api.ListProfiles().then(p => setProfiles((p || []) as any)).catch(x => { setProfiles([]); setE(err(x)); }); }, [given]);
  useEffect(() => { api.Running().then(r => setOpen(new Set((r || []).map((x: any) => x.profileId)))).catch(() => {}); }, []);
  const list = profiles || [];
  const shown = list.filter(p => !q || `${p.name} ${p.group} ${(p.tags || []).join(' ')}`.toLowerCase().includes(q.toLowerCase()));
  const sorted = [...shown].sort((a, b) => Number(open.has(b.id)) - Number(open.has(a.id)));
  const chosen = list.find(p => p.id === pick);
  const go = async () => {
    if (!chosen) return;
    setBusy(true); setE('');
    localStorage.setItem('gb.recProfile', chosen.id); localStorage.setItem('gb.recOverlay', overlay ? '1' : '0');
    try { onStarted((await api.StartRecording(chosen.id, { overlay, scroll, passwordVar: pwVar, maxDelayMs: 5000, startUrl: url.trim() } as any)) as any); }
    catch (x) { setE(err(x)); setBusy(false); }
  };
  return <>
    <div className="dhead">
      <div className="glyph rec-glyph"><span className="rec-dot-ic" /></div>
      <div style={{ minWidth: 0 }}><h3>录制操作</h3><div className="sub ell">{target.mode === 'new' ? '在浏览器里正常操作，每个动作自动变成一个步骤，最后生成新流程' : `录制的步骤会插入到「${target.after}」之后`}</div></div>
      <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={onCancel}><Icon.x /></button></div>
    </div>
    <div className="dbody">
      <div className="rec-label">在哪个环境里录制</div>
      <div className="rpa-search" style={{ marginBottom: 8 }}><Icon.search s={14} /><input type="text" placeholder="搜索环境名称、分组、标签" value={q} onChange={ev => setQ(ev.target.value)} /></div>
      <div className="assign-list rec-plist">
        {profiles === null && <div className="rec-loading"><ThinkingOrb state="working" size={20} theme="dark" /> 正在读取环境…</div>}
        {sorted.map(p => <div key={p.id} className={`assign-row ${pick === p.id ? 'on' : ''}`} onClick={() => setPick(p.id)}>
          <span className={`rec-radio ${pick === p.id ? 'on' : ''}`} />
          <Avatar seed={p.fingerprint?.noiseSeed || p.id} os={p.fingerprint?.os || 'windows'} running={open.has(p.id)} />
          <div style={{ flex: 1, minWidth: 0 }}><div className="nm ell">{p.name}</div><div className="sub ell">{p.group || '未分组'}{(p.tags || []).length ? ' · ' + p.tags.join('、') : ''}</div></div>
          {open.has(p.id) ? <span className="badge ok"><span className="d" />已打开</span> : <span className="badge unknown">将自动启动</span>}
        </div>)}
        {profiles !== null && sorted.length === 0 && <div className="sub" style={{ padding: 22, textAlign: 'center' }}>{list.length ? '没有符合条件的环境' : '还没有环境，请先在「环境」页创建'}</div>}
      </div>
      <div className="rec-label" style={{ marginTop: 16 }}>起始网址 <span className="sub">可选，留空则从浏览器当前页面开始</span></div>
      <input type="text" className="rec-url" placeholder="https://" value={url} onChange={ev => setUrl(ev.target.value)} onKeyDown={ev => ev.key === 'Enter' && chosen && !busy && go()} />
      <div className="rec-opts">
        <Toggle on={overlay} onChange={setOverlay} label="在浏览器里显示录制浮窗" hint="右下角的小工具条：暂停、停止、提取文本、等待元素" />
        <Toggle on={pwVar} onChange={setPwVar} label="密码转为变量" hint="密码框输入记为 {{password}}，不保存明文" />
        <Toggle on={scroll} onChange={setScroll} label="录制页面滚动" hint="连续滚动合并为一步，记录最终位置" />
      </div>
      {e && <div className="notice bad" style={{ margin: '12px 0 0' }}><Icon.alert /><span className="grow">{e}</span></div>}
    </div>
    <div className="dfoot">
      <span className="sub rec-foot-hint">{chosen ? <>将在 <b>{chosen.name}</b> 中录制</> : '请选择一个环境'}</span>
      <div className="spacer" />
      <button className="btn" onClick={onCancel}>取消</button>
      <button className="btn primary rec-go" disabled={!chosen || busy} onClick={go}><span className="rec-dot-ic sm" /> {busy ? (open.has(pick) ? '正在连接…' : '正在启动浏览器…') : '开始录制'}</button>
    </div>
  </>;
};

// ---------------- live ----------------
const useNow = (on: boolean) => {
  const [now, setNow] = useState(Date.now());
  useEffect(() => { if (!on) return; const t = setInterval(() => setNow(Date.now()), 250); return () => clearInterval(t); }, [on]);
  return now;
};

const Live: React.FC<{ st: RecState; setSt: (s: RecState) => void; onStop: () => Promise<void>; onCancel: () => void }> = ({ st, setSt, onStop, onCancel }) => {
  const toast = useToast();
  const now = useNow(!st.paused);
  const [stopping, setStopping] = useState(false);
  const [editing, setEditing] = useState('');
  const listRef = useRef<HTMLDivElement>(null);
  const seen = useRef<Set<string>>(new Set(st.steps.map(s => s.id)));
  const fresh = useRef<Set<string>>(new Set());
  const real = st.steps.filter(s => !isThink(s));
  const elapsed = (st.paused && st.pausedAt ? st.pausedAt : now) - st.startedAt - st.pausedMs;

  // Mark newly arrived steps for the entrance animation and follow the tail.
  const ids = st.steps.map(s => s.id).join(',');
  useEffect(() => {
    fresh.current = new Set(st.steps.filter(s => !seen.current.has(s.id)).map(s => s.id));
    st.steps.forEach(s => seen.current.add(s.id));
    const el = listRef.current;
    if (el && fresh.current.size && !editing) el.scrollTo({ top: el.scrollHeight, behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth' });
  }, [ids]); // eslint-disable-line react-hooks/exhaustive-deps

  const pause = async () => { try { setSt((await (st.paused ? api.ResumeRecording() : api.PauseRecording())) as any); } catch (x) { toast(err(x), 'bad'); } };
  const stop = async () => { setStopping(true); try { await onStop(); } catch (x) { toast(err(x), 'bad'); setStopping(false); } };
  const pick = async (m: 'extract' | 'assert') => { try { await api.RecordPickMode(m); toast(m === 'extract' ? '请到浏览器里点击要提取文字的元素' : '请到浏览器里点击要等待出现的元素'); } catch (x) { toast(err(x), 'bad'); } };
  const del = async (id: string) => { try { setSt((await api.RecordRemoveStep(id)) as any); } catch (x) { toast(err(x), 'bad'); } };
  const save = async (s: RStep) => { try { setSt((await api.RecordUpdateStep(s as any)) as any); setEditing(''); } catch (x) { toast(err(x), 'bad'); } };
  let n = 0;
  return <>
    <div className={`rec-head ${st.paused ? 'paused' : ''}`}>
      <div className="rec-orb" aria-hidden><i /><i /><b /></div>
      <div className="rec-title">
        <h3>{st.paused ? '录制已暂停' : '正在录制'}</h3>
        <div className="sub ell"><span className="rec-pchip"><Icon.globe s={12} />{st.profileName}</span>{st.tabs > 1 && <span className="rec-tabs">{st.tabs} 个标签页</span>}</div>
      </div>
      <div className="rec-meters">
        <div className="rec-meter"><b className="num">{mmss(elapsed)}</b><span>用时</span></div>
        <div className="rec-meter"><b className="num" key={real.length}>{real.length}</b><span>步骤</span></div>
      </div>
    </div>
    <div className="rec-hint">
      {st.paused ? <><Icon.alert s={14} /> <span className="grow">已暂停：现在浏览器里的操作不会被录制</span></> : <><span className="rec-live-dot" /> <span className="grow ell" title="点击、输入、选择、上传、切换标签页都会自动记录">切到浏览器窗口正常操作，每个动作都会自动记录</span></>}
      <span className="rec-pick-l">点选元素</span>
      <button className="rec-pick" disabled={st.paused} onClick={() => pick('extract')} title="在浏览器里点一个元素，把它的文字保存到变量"><StepIcon type="extract_text" s={12} /> 提取文本</button>
      <button className="rec-pick" disabled={st.paused} onClick={() => pick('assert')} title="在浏览器里点一个元素，回放时等它出现再继续"><StepIcon type="wait_for" s={12} /> 等待元素</button>
    </div>
    <div className="rec-list" ref={listRef}>
      {st.steps.length === 0 && <div className="rec-empty"><ThinkingOrb state="listening" size={32} theme="dark" /><b>等待你的第一个操作…</b><span>在浏览器里打开网页、点击或输入，步骤会实时出现在这里</span></div>}
      {st.steps.map(s => {
        if (isThink(s)) return <div key={s.id} className={`rec-think ${fresh.current.has(s.id) ? 'new' : ''}`}><i /><span>停顿 {(+s.params.ms / 1000).toFixed(1)} 秒</span><i /><button className="rec-x" title="删除这个停顿" onClick={() => del(s.id)}><Icon.x s={11} /></button></div>;
        n++;
        return <RecRow key={s.id} s={s} n={n} isNew={fresh.current.has(s.id)} editing={editing === s.id} onEdit={() => setEditing(editing === s.id ? '' : s.id)} onDel={() => del(s.id)} onSave={save} />;
      })}
    </div>
    <div className="dfoot">
      <button className="btn ghost" onClick={onCancel}>放弃</button>
      <div className="spacer" />
      <button className="btn" onClick={pause}>{st.paused ? <><Icon.play s={13} /> 继续录制</> : <><span className="rec-pause-ic" /> 暂停</>}</button>
      <button className="btn rec-stop" disabled={stopping} onClick={stop}><span className="rec-stop-ic" /> {stopping ? '正在结束…' : '停止并检查'}</button>
    </div>
  </>;
};

const RecRow: React.FC<{ s: RStep; n: number; isNew: boolean; editing: boolean; onEdit: () => void; onDel: () => void; onSave: (s: RStep) => void }> = ({ s, n, isNew, editing, onEdit, onDel, onSave }) => {
  const keys: [string, string][] = [...(EDIT_KEYS[s.type] || []), ...(s.params?.frame ? [['frame', '所在框架'] as [string, string]] : [])];
  const [draft, setDraft] = useState<Record<string, string>>(s.params);
  useEffect(() => { if (editing) setDraft(s.params); }, [editing]); // eslint-disable-line react-hooks/exhaustive-deps
  const pw = s.params?._rec === 'password';
  return <div className={`rec-row ${isNew ? 'new' : ''} ${editing ? 'editing' : ''}`}>
    <div className="rec-row-main">
      <span className="rec-n num">{n}</span>
      <span className={`rpa-glyph sm ${toneOf(s.type)}`}><StepIcon type={s.type} s={12} /></span>
      <div className="rec-row-t">
        <div className="nm">{label(s)}{pw && <span className="rec-tag">密码变量</span>}{s.params?.double === 'true' && <span className="rec-tag">双击</span>}</div>
        <div className="sm mono ell" title={s.params?.selector || summary(s)}>{brief(s)}</div>
      </div>
      <div className="rec-acts">
        {keys.length > 0 && <button className="btn ghost icon sm" title="编辑" onClick={onEdit}><Icon.edit s={13} /></button>}
        <button className="btn ghost icon sm danger" title="删除" onClick={onDel}><Icon.trash s={13} /></button>
      </div>
    </div>
    {editing && <div className="rec-edit">
      {keys.map(([k, l]) => <label key={k}><span>{l}</span><input type="text" className={k === 'text' && !pw ? '' : 'mono'} value={draft[k] || ''} onChange={e => setDraft(d => ({ ...d, [k]: e.target.value }))}
        onKeyDown={e => { if (e.key === 'Enter') onSave({ ...s, params: draft }); if (e.key === 'Escape') { e.stopPropagation(); onEdit(); } }} autoFocus={k === keys[0][0]} /></label>)}
      <div className="rec-edit-acts"><button className="btn sm ghost" onClick={onEdit}>取消</button><button className="btn sm primary" onClick={() => onSave({ ...s, params: draft })}><Icon.check s={13} /> 保存</button></div>
    </div>}
  </div>;
};

// ---------------- review ----------------
const Review: React.FC<{ st: RecState; target: RecTarget; onAgain: () => void; onDiscard: () => void; onUse: (steps: RStep[], vars: Record<string, string>) => void }> = ({ st, target, onAgain, onDiscard, onUse }) => {
  const [steps, setSteps] = useState<RStep[]>(st.steps);
  const [keepDelays, setKeepDelays] = useState(false);
  const [merge, setMerge] = useState(true);
  const [pwVar, setPwVar] = useState(st.options?.passwordVar !== false);
  const [res, setRes] = useState<{ steps: RStep[]; vars: Record<string, string> } | null>(null);
  useEffect(() => { setSteps(st.steps); }, [st.steps]);
  useEffect(() => {
    let live = true;
    api.FinalizeRecording(steps as any, { keepDelays, mergeFills: merge, passwordVar: pwVar } as any).then((r: any) => live && setRes({ steps: r?.steps || [], vars: r?.vars || {} })).catch(() => live && setRes({ steps, vars: {} }));
    return () => { live = false; };
  }, [steps, keepDelays, merge, pwVar]);
  const thinks = steps.filter(isThink).length;
  const hasPw = steps.some(s => s.params?._rec === 'password');
  const fills = steps.filter(s => s.type === 'fill').length;
  const out = res?.steps || [];
  const vars = Object.keys(res?.vars || {});
  const dur = (st.pausedAt || Date.now()) - st.startedAt - st.pausedMs;
  const removed = steps.filter(s => !isThink(s)).length - out.filter(s => !(s.type === 'delay' && keepDelays && s.name === '停顿')).length;
  return <>
    <div className="dhead">
      <div className="glyph"><Icon.check s={18} /></div>
      <div style={{ minWidth: 0 }}><h3>检查录制结果</h3><div className="sub ell">{st.profileName} · 用时 {mmss(Math.max(0, st.startedAt ? dur : 0))} · 共 {out.length} 个步骤{st.reason ? ` · ${st.reason}` : ''}</div></div>
      <div style={{ marginLeft: 'auto' }}><button className="btn ghost icon" onClick={onDiscard}><Icon.x /></button></div>
    </div>
    <div className="rec-review">
      <div className="rec-switches">
        <RevSwitch on={keepDelays} set={setKeepDelays} t="保留等待时间" d={thinks ? `录制到 ${thinks} 处停顿，回放时照样等待` : '这次没有明显的停顿'} disabled={!thinks} />
        <RevSwitch on={merge} set={setMerge} t="合并输入" d={fills ? '同一输入框被覆盖的多次输入只保留最后一次' : '这次没有输入'} disabled={!fills} />
        <RevSwitch on={pwVar} set={setPwVar} t="密码转为变量" d={hasPw ? (pwVar ? '记为 {{password}}，运行前在变量里填写' : '明文保存在流程里，请注意安全') : '这次没有输入密码'} disabled={!hasPw} warn={hasPw && !pwVar} />
      </div>
      {vars.length > 0 && <div className="notice warn rec-vars"><Icon.alert /><span className="grow">回放前请在「初始变量」里填写：{vars.map(v => <code key={v}>{v}</code>)}{vars.some(v => v.startsWith('upload_file')) ? '（上传文件填本机完整路径）' : ''}</span></div>}
      <div className="rec-list review">
        {out.length === 0 && <div className="rec-empty"><ThinkingOrb state="breathing" size={32} theme="dark" /><b>没有录制到步骤</b><span>可以重新录制，或直接关闭</span></div>}
        {out.map((s, i) => <div key={s.id} className="rec-row static" style={{ ['--i' as any]: Math.min(i, 16) }}>
          <div className="rec-row-main">
            <span className="rec-n num">{i + 1}</span>
            <span className={`rpa-glyph sm ${toneOf(s.type)}`}><StepIcon type={s.type} s={12} /></span>
            <div className="rec-row-t"><div className="nm">{label(s)}</div><div className="sm mono ell" title={s.params?.selector || summary(s)}>{brief(s)}</div></div>
            <div className="rec-acts"><button className="btn ghost icon sm danger" title="删除" onClick={() => setSteps(l => l.filter(x => x.id !== s.id))}><Icon.trash s={13} /></button></div>
          </div>
        </div>)}
      </div>
    </div>
    <div className="dfoot">
      <button className="btn ghost" onClick={onAgain}><Icon.refresh s={13} /> 重新录制</button>
      {removed > 0 && <span className="sub">已精简 {removed} 步</span>}
      <div className="spacer" />
      <button className="btn" onClick={onDiscard}>丢弃</button>
      <button className="btn primary" disabled={!out.length} onClick={() => onUse(out, res?.vars || {})}><Icon.check s={14} /> {target.mode === 'new' ? '生成流程' : `插入 ${out.length} 个步骤`}</button>
    </div>
  </>;
};

const RevSwitch: React.FC<{ on: boolean; set: (v: boolean) => void; t: string; d: string; disabled?: boolean; warn?: boolean }> = ({ on, set, t, d, disabled, warn }) => (
  <button className={`rec-sw ${on && !disabled ? 'on' : ''} ${disabled ? 'off' : ''} ${warn ? 'warn' : ''}`} disabled={disabled} onClick={() => set(!on)} aria-pressed={on}>
    <div className="rec-sw-top"><b>{t}</b><span className={`sw ${on && !disabled ? 'on' : ''}`} /></div>
    <span className="rec-sw-d">{d}</span>
  </button>
);

export const recName = () => { const d = new Date(); const p = (n: number) => String(n).padStart(2, '0'); return `录制的流程 ${d.getMonth() + 1}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`; };
